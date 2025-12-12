package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()

	// Create the ADK agent
	adkAgent, err := createAgent(ctx)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Start the server
	log.Println("Starting Go ADK Agent with AG-UI support...")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := startServer(adkAgent); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	log.Println("Shutting down server...")
}
