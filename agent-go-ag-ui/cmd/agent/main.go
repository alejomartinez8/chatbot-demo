package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-go-ag-ui/internal/agent"
	"agent-go-ag-ui/internal/agui"
	"agent-go-ag-ui/internal/config"
	"agent-go-ag-ui/internal/server"
	"agent-go-ag-ui/internal/session"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create agent
	ctx := context.Background()
	adkAgent, err := agent.New(ctx, cfg.GoogleAPIKey)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Initialize components
	sessionMgr := session.NewManager()
	stateMgr := agui.NewStateManager()
	streamer := agui.NewStreamer(adkAgent, sessionMgr, cfg.AppName)
	sseHandler := agui.NewHandler(adkAgent, streamer, stateMgr, cfg.AppName)
	connectHandler := agui.NewConnectHandler(adkAgent, streamer, stateMgr, cfg.AppName)

	// Create and start server
	srv := server.New(cfg, sseHandler, connectHandler)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Starting Go ADK Agent with AG-UI support...")

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	log.Println("Shutting down server...")

	if err := srv.ShutdownTimeout(5 * time.Second); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
}
