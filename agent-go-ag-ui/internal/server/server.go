package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"agent-go-ag-ui/internal/config"
	"agent-go-ag-ui/internal/agui"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	handler    *agui.Handler
}

// New creates a new server instance
func New(cfg *config.Config, h *agui.Handler) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.HandleAgentRequest)

	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + cfg.Port,
			Handler: CORS(Logging(mux)),
		},
		handler: h,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting AG-UI server on port %s", s.httpServer.Addr)
	log.Printf("Agent will be accessible at http://localhost:%s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// ShutdownTimeout shuts down the server with a default timeout
func (s *Server) ShutdownTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.Shutdown(ctx)
}
