package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"agent-go-ag-ui/internal/agui_adapter"
	"agent-go-ag-ui/internal/transport"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

// Handler handles HTTP requests for the AG-UI protocol via SSE
// Only responsible for HTTP/SSE serialization - protocol logic is in agui_adapter
type Handler struct {
	adapter  *agui_adapter.AGUIAdapter
	stateMgr *transport.StateManager
}

// NewHandler creates a new SSE handler
func NewHandler(adapter *agui_adapter.AGUIAdapter, stateMgr *transport.StateManager) *Handler {
	return &Handler{
		adapter:  adapter,
		stateMgr: stateMgr,
	}
}

// sseEventSender implements agui_adapter.EventSender for SSE transport
type sseEventSender struct {
	writer *bufio.Writer
}

func (s *sseEventSender) SendEvent(event events.Event) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	_, err = fmt.Fprintf(s.writer, "data: %s\n\n", eventJSON)
	if err != nil {
		return err
	}
	return s.writer.Flush()
}

func (s *sseEventSender) SendRunError(runID string, err error) error {
	errorEvent := events.NewRunErrorEvent(err.Error(), events.WithRunID(runID))
	return s.SendEvent(errorEvent)
}

// HandleAgentRequest handles AG-UI protocol requests
func (h *Handler) HandleAgentRequest(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var input agui_adapter.RunAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input early (fail fast)
	if err := input.Validate(); err != nil {
		log.Printf("Validation error: %v", err)
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Create context for agent execution
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Create buffered writer for SSE
	bufWriter := bufio.NewWriter(w)

	// Create SSE event sender
	sender := &sseEventSender{writer: bufWriter}

	// Delegate protocol logic to adapter
	if err := h.adapter.RunAgentProtocol(ctx, &input, h.stateMgr, sender); err != nil {
		log.Printf("Error running agent protocol: %v", err)
		// Error already sent via sender.SendRunError
		return
	}
}
