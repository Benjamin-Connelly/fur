// Package demo shows syntax highlighting in the preview pane.
package demo

import (
	"fmt"
	"net/http"
	"time"
)

// Server is a minimal HTTP server used in the demo.
type Server struct {
	Addr    string
	Timeout time.Duration
}

// NewServer returns a Server with sane defaults.
func NewServer(addr string) *Server {
	return &Server{
		Addr:    addr,
		Timeout: 10 * time.Second,
	}
}

// Run starts the HTTP server and blocks until it exits.
func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{
		Addr:         s.Addr,
		Handler:      mux,
		ReadTimeout:  s.Timeout,
		WriteTimeout: s.Timeout,
	}
	return srv.ListenAndServe()
}
