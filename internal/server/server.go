package server

import (
	"context"
	"fmt"
	"net/http"
)

// Server wraps the standard HTTP server with helper methods for boot
// and graceful shutdown.
type Server struct {
	httpServer *http.Server
}

func New(port string, handler http.Handler) *Server {
	addr := port
	if addr == "" {
		addr = "8080"
	}
	if addr[0] != ':' {
		addr = fmt.Sprintf(":%s", addr)
	}

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (s *Server) Start() error {
	if s.httpServer == nil {
		return fmt.Errorf("server not configured")
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
