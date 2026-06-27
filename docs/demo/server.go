// Package demo shows Go syntax highlighting in the preview pane.
package demo

import (
	"fmt"
	"net/http"
	"time"
)

// Cattery is a tiny HTTP service that introduces its resident cats.
type Cattery struct {
	Addr      string
	Timeout   time.Duration
	Residents []string
}

// NewCattery returns a Cattery stocked with a few regulars.
func NewCattery(addr string) *Cattery {
	return &Cattery{
		Addr:      addr,
		Timeout:   10 * time.Second,
		Residents: []string{"Biscuit", "Mortimer", "Nimbus", "Patches"},
	}
}

// Run starts the cattery server and blocks until it exits.
func (c *Cattery) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/meow", func(w http.ResponseWriter, r *http.Request) {
		for _, cat := range c.Residents {
			fmt.Fprintf(w, "%s says meow\n", cat)
		}
	})

	srv := &http.Server{
		Addr:         c.Addr,
		Handler:      mux,
		ReadTimeout:  c.Timeout,
		WriteTimeout: c.Timeout,
	}
	return srv.ListenAndServe()
}
