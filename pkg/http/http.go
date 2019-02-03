package http

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Server struct {
	srv *http.Server
}

// NewServer returns a new simple HTTP server.
func NewServer(addr string) *Server {
	// create a simple multiplexer
	mux := http.NewServeMux()

	// pass mux into handlers to add mappings
	MuxHandlers(mux)

	// create our http.Server instance
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// return the instantiated server
	return &Server{
		srv: srv,
	}
}

// Start starts the server running. We also create a channel listening for
// interrupt signals before gracefully shutting down.
func (s *Server) Start() {
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	go func() {
		log.Println("Starting server")
		if err := s.srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	<-stopChan
	log.Println("Stopping server")

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	s.srv.Shutdown(ctx)
}
