package web

import (
	"context"
	"crypto/md5"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/plugin"
	"github.com/Benjamin-Connelly/fur/internal/render"
	"github.com/Benjamin-Connelly/fur/internal/web/static"
	"github.com/spf13/afero"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Server is the HTTP server for web mode.
type Server struct {
	cfg         *config.Config
	idx         index.Indexer
	links       *index.LinkGraph
	plugins     *plugin.Registry
	code        *render.CodeRenderer
	md          goldmark.Markdown
	fs          afero.Fs
	mux         *http.ServeMux
	server      *http.Server
	sse         *SSEBroker
	initialFile string // optional: file to land on at startup (sets printed URL and --open target)
}

// SetInitialFile configures the file the server points at on startup. The
// printed URL and the --open browser target append this path. Empty means
// land on the directory index (default).
func (s *Server) SetInitialFile(relPath string) {
	s.initialFile = relPath
}

// SSEBroker manages Server-Sent Events for live reload.
type SSEBroker struct {
	clients    map[chan string]bool
	register   chan chan string
	unregister chan chan string
	broadcast  chan string
	done       chan struct{}
	stopOnce   sync.Once
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

// Stop shuts down the broker's run loop and closes all client connections.
// Idempotent: safe to call more than once (e.g. both an explicit Stop and a
// graceful-shutdown path).
func (b *SSEBroker) Stop() {
	b.stopOnce.Do(func() {
		close(b.done)
	})
}

// Notify sends a reload event to all connected clients.
func (b *SSEBroker) Notify(path string) {
	select {
	case b.broadcast <- path:
	case <-b.done:
	}
}

// NewMarkdown builds the Goldmark instance used to render markdown to HTML in
// web mode. It is deliberately NOT configured with html.WithUnsafe(), so raw
// HTML in source (e.g. <script>, <iframe>) is escaped rather than passed
// through. Exposed as a constructor so the exact web rendering config is the
// single source of truth and can be golden-tested directly.
func NewMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, highlighting.Emoji),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

// New creates a new web server.
func New(cfg *config.Config, idx index.Indexer, links *index.LinkGraph, plugins *plugin.Registry) *Server {
	s := &Server{
		cfg:     cfg,
		idx:     idx,
		links:   links,
		plugins: plugins,
		code:    render.NewCodeRenderer(cfg.Theme, false),
		md:      NewMarkdown(),
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

// Handler returns the fully-wrapped HTTP handler (routes + security-header,
// logging, and ETag middleware) that Start serves. Exposed so the handler can
// be driven by an httptest.Server in tests without binding a real port.
func (s *Server) Handler() http.Handler {
	return s.middleware(s.mux)
}

// isLoopbackHost reports whether host binds only the loopback interface.
// The empty host is treated as loopback because fur defaults Server.Host to
// "localhost"; an empty value reaching here would otherwise mean "all
// interfaces", which is exactly what we refuse without an explicit opt-in.
func isLoopbackHost(host string) bool {
	switch host {
	case "", "localhost", "127.0.0.1", "::1":
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	// A non-localhost hostname can resolve to any interface; treat as public.
	return false
}

// ValidateBind refuses to bind a non-loopback host unless the operator has
// explicitly opted in with listenPublic. Exposing the browser UI to the
// network lets a co-located or remote adversary enumerate and read the
// entire browsed tree over the file/search/document APIs (audit Chain C).
func ValidateBind(host string, listenPublic bool) error {
	if listenPublic || isLoopbackHost(host) {
		return nil
	}
	return fmt.Errorf("refusing to bind web server to non-loopback address %q: "+
		"this exposes the browsed files to other hosts and users on the network. "+
		"Re-run with --listen-public if that is intended", host)
}

// Start begins listening on the configured port and handles graceful shutdown.
func (s *Server) Start() error {
	if err := ValidateBind(s.cfg.Server.Host, s.cfg.Server.ListenPublic); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)

	if s.cfg.Server.ListenPublic && !isLoopbackHost(s.cfg.Server.Host) {
		fmt.Fprintf(os.Stderr, "warning: --listen-public is set; serving on %s is reachable by other hosts on the network\n", addr)
	}

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

	startURL := fmt.Sprintf("http://%s", addr)
	if s.initialFile != "" {
		startURL = startURL + "/" + strings.TrimPrefix(s.initialFile, "/")
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("fur serving %s\n", startURL)
		errCh <- s.server.ListenAndServe()
	}()

	// Open browser if requested, but skip when running over SSH
	isSSH := os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_CONNECTION") != ""
	if s.cfg.Server.Open && isSSH {
		fmt.Println("SSH session detected, skipping browser open")
	} else if s.cfg.Server.Open {
		go func() {
			// Small delay to let the server start
			time.Sleep(200 * time.Millisecond)
			_ = exec.Command("xdg-open", startURL).Start()
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
		// script-src drops 'unsafe-inline' (audit Chain D / hardening 4.4): all
		// of fur's own scripts are now external files under /__static, so an
		// injected inline <script> in rendered content will not execute. The
		// jsdelivr/d3js hosts remain for the Mermaid and D3 libraries (loaded
		// from /__static/*.js module/script files). style-src keeps
		// 'unsafe-inline' for now — inline <style> and style attributes are
		// still in use and CSS injection is lower-severity (tracked separately).
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://cdn.jsdelivr.net https://d3js.org; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
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
