package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"agent-go-ag-ui/gen/proto/agui/v1/aguiv1connect"
	"agent-go-ag-ui/internal/config"
	"agent-go-ag-ui/internal/transport/connectrpc"
	"agent-go-ag-ui/internal/transport/sse"
)

const (
	// EndpointSSE is the endpoint for Server-Sent Events transport
	EndpointSSE = "/sse"
	// EndpointConnect is the endpoint for Connect RPC transport
	EndpointConnect = "/connect"
)

// Server represents the HTTP server
type Server struct {
	httpServer     *http.Server
	sseHandler     *sse.Handler
	connectHandler *connectrpc.Handler
}

// New creates a new server instance with multiple transport endpoints
func New(cfg *config.Config, sseHandler *sse.Handler, connectHandler *connectrpc.Handler) *Server {
	mux := http.NewServeMux()

	// SSE endpoint (explicit)
	mux.HandleFunc(EndpointSSE, sseHandler.HandleAgentRequest)

	// Connect RPC endpoint
	if connectHandler != nil {
		path, handler := aguiv1connect.NewAGUIServiceHandler(connectHandler)
		mux.Handle(path, handler)
		// Also register explicit endpoint for convenience
		mux.HandleFunc(EndpointConnect, handler.ServeHTTP)
	}

	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + cfg.Port,
			Handler: CORS(Logging(mux)),
		},
		sseHandler:     sseHandler,
		connectHandler: connectHandler,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting AG-UI server on port %s", s.httpServer.Addr)
	log.Printf("SSE endpoint: http://localhost:%s%s", s.httpServer.Addr, EndpointSSE)
	if s.connectHandler != nil {
		log.Printf("Connect RPC endpoint: http://localhost:%s%s", s.httpServer.Addr, EndpointConnect)
	} else {
		log.Printf("Connect RPC endpoint: http://localhost:%s%s (not configured)", s.httpServer.Addr, EndpointConnect)
	}
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
