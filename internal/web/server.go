package web

import (
	"context"
	"crypto/md5"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/plugin"
	"github.com/Benjamin-Connelly/fur/internal/render"
	"github.com/Benjamin-Connelly/fur/internal/web/static"
	"github.com/spf13/afero"
)

// Server is the HTTP server for web mode.
type Server struct {
	cfg     *config.Config
	idx     *index.Index
	links   *index.LinkGraph
	plugins *plugin.Registry
	code    *render.CodeRenderer
	fs      afero.Fs
	mux     *http.ServeMux
	server  *http.Server
	sse     *SSEBroker
}

// SSEBroker manages Server-Sent Events for live reload.
type SSEBroker struct {
	clients    map[chan string]bool
	register   chan chan string
	unregister chan chan string
	broadcast  chan string
	done       chan struct{}
}

// NewSSEBroker creates a new SSE event broker.
func NewSSEBroker() *SSEBroker {
	b := &SSEBroker{
		clients:    make(map[chan string]bool),
		register:   make(chan chan string),
		unregister: make(chan chan string),
		broadcast:  make(chan string, 16),
		done:       make(chan struct{}),
	}
	go b.run()
	return b
}

func (b *SSEBroker) run() {
	for {
		select {
		case <-b.done:
			// Close all client channels so SSE handlers unblock
			for client := range b.clients {
				close(client)
			}
			b.clients = nil
			return
		case client := <-b.register:
			b.clients[client] = true
		case client := <-b.unregister:
			delete(b.clients, client)
			close(client)
		case msg := <-b.broadcast:
			for client := range b.clients {
				select {
				case client <- msg:
				default:
					delete(b.clients, client)
					close(client)
				}
			}
		}
	}
}

// Stop shuts down the broker and closes all client connections.
func (b *SSEBroker) Stop() {
	close(b.done)
}

// Notify sends a reload event to all connected clients.
func (b *SSEBroker) Notify(path string) {
	select {
	case b.broadcast <- path:
	case <-b.done:
	}
}

// New creates a new web server.
func New(cfg *config.Config, idx *index.Index, links *index.LinkGraph, plugins *plugin.Registry) *Server {
	s := &Server{
		cfg:     cfg,
		idx:     idx,
		links:   links,
		plugins: plugins,
		code:    render.NewCodeRenderer(cfg.Theme, false),
		fs:      idx.Fs(),
		mux:     http.NewServeMux(),
		sse:     NewSSEBroker(),
	}

	s.registerRoutes()
	return s
}

// OnFileChange is a callback for the file watcher. Wire it to index.Watcher's onChange.
func (s *Server) OnFileChange(relPath string) {
	s.sse.Notify(relPath)
}

// Start begins listening on the configured port and handles graceful shutdown.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.middleware(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("fur serving http://%s\n", addr)
		errCh <- s.server.ListenAndServe()
	}()

	// Open browser if requested, but skip when running over SSH
	isSSH := os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_CONNECTION") != ""
	if s.cfg.Server.Open && isSSH {
		fmt.Println("SSH session detected, skipping browser open")
	} else if s.cfg.Server.Open {
		url := fmt.Sprintf("http://%s", addr)
		go func() {
			// Small delay to let the server start
			time.Sleep(200 * time.Millisecond)
			_ = exec.Command("xdg-open", url).Start()
		}()
	}

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		fmt.Printf("\nreceived %v, shutting down\n", sig)
		return s.Stop()
	}
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	// Close SSE broker first so SSE handler goroutines unblock
	s.sse.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// handleCustomCSS serves the user-specified custom CSS file.
func (s *Server) handleCustomCSS(w http.ResponseWriter, r *http.Request) {
	cssPath := s.cfg.Server.CustomCSS
	if cssPath == "" {
		http.NotFound(w, r)
		return
	}

	// Resolve relative paths against the served root
	if !filepath.IsAbs(cssPath) {
		cssPath = filepath.Join(s.idx.Root(), cssPath)
	}

	// Ensure the resolved path doesn't escape expected directories
	resolved, err := filepath.EvalSymlinks(cssPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rootPrefix := s.idx.Root() + string(filepath.Separator)
	if !strings.HasPrefix(resolved, rootPrefix) && resolved != s.idx.Root() {
		http.NotFound(w, r)
		return
	}

	data, err := afero.ReadFile(s.fs, resolved)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}

// middleware chains security headers, request logging, and ETag support.
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net https://d3js.org; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		next.ServeHTTP(w, r)
		if s.cfg.Debug {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

// etagMiddleware wraps a handler to add ETag caching.
func etagMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rec, r)

		if rec.statusCode == 200 && len(rec.body) > 0 {
			etag := fmt.Sprintf(`"%x"`, md5.Sum(rec.body))
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "no-cache")

			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Content-Type", rec.contentType)
			w.WriteHeader(rec.statusCode)
			w.Write(rec.body)
			return
		}

		// Non-200 or empty body: already written by recorder fallthrough
		if !rec.captured {
			return
		}
		w.Header().Set("Content-Type", rec.contentType)
		w.WriteHeader(rec.statusCode)
		w.Write(rec.body)
	}
}

// responseRecorder captures response data for ETag generation.
type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	body        []byte
	contentType string
	captured    bool
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.captured = true
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.captured = true
	r.contentType = r.Header().Get("Content-Type")
	r.body = append(r.body, b...)
	return len(b), nil
}

func (s *Server) registerRoutes() {
	// Static assets
	staticFS, err := fs.Sub(static.Files, ".")
	if err != nil {
		log.Fatalf("failed to create static sub-filesystem: %v", err)
	}
	s.mux.Handle("/__static/", http.StripPrefix("/__static/", http.FileServer(http.FS(staticFS))))

	// Custom CSS override route
	if s.cfg.Server.CustomCSS != "" {
		s.mux.HandleFunc("/__custom.css", s.handleCustomCSS)
	}

	// API routes
	s.mux.HandleFunc("/__api/files", s.handleAPIFiles)
	s.mux.HandleFunc("/__api/search", s.handleAPISearch)
	s.mux.HandleFunc("/__api/graph", s.handleAPIGraph)
	s.mux.HandleFunc("/__api/document", s.handleAPIDocument)
	s.mux.HandleFunc("/__api/tasks", s.handleAPITasks)
	s.mux.HandleFunc("/__events", s.handleSSE)

	// Graph page
	s.mux.HandleFunc("/graph", s.handleGraph)

	// All other routes go through root handler with ETag support
	s.mux.HandleFunc("/", etagMiddleware(s.handleRoot))
}
