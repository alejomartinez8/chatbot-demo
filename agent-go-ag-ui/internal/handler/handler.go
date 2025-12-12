package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"google.golang.org/adk/agent"

	"agent-go-ag-ui/internal/stream"
)

// Handler handles HTTP requests for the AG-UI protocol
type Handler struct {
	agent      agent.Agent
	streamer   *stream.Streamer
	appName    string
	defaultUID string
}

// NewHandler creates a new handler
func NewHandler(agent agent.Agent, streamer *stream.Streamer, appName string) *Handler {
	return &Handler{
		agent:      agent,
		streamer:   streamer,
		appName:    appName,
		defaultUID: "demo_user",
	}
}

// HandleAgentRequest handles AG-UI protocol requests
func (h *Handler) HandleAgentRequest(w http.ResponseWriter, r *http.Request) {
	sseWriter := sse.NewSSEWriter()

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
	var input RunAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use IDs from input or generate new ones
	threadID := input.ThreadID
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}
	runID := input.RunID
	if runID == "" {
		runID = events.GenerateRunID()
	}

	// If no messages, just acknowledge connection
	if len(input.Messages) == 0 {
		// Send a connection established event (custom event)
		connEvent := events.NewCustomEvent("connection_established", events.WithValue(map[string]interface{}{
			"status": "connected",
		}))
		ctx := r.Context()
		bufWriter := bufio.NewWriter(w)
		if err := sseWriter.WriteEvent(ctx, bufWriter, connEvent); err != nil {
			log.Printf("Error writing connection event: %v", err)
		}
		bufWriter.Flush()
		return
	}

	// Get the last user message
	lastUserMessage := h.extractLastUserMessage(input.Messages)
	if lastUserMessage == "" {
		// Send error event using RUN_ERROR
		errorEvent := events.NewRunErrorEvent("No user message found", events.WithRunID(runID))
		ctx := r.Context()
		bufWriter := bufio.NewWriter(w)
		if err := sseWriter.WriteEvent(ctx, bufWriter, errorEvent); err != nil {
			log.Printf("Error writing error event: %v", err)
		}
		bufWriter.Flush()
		return
	}

	// Create context for agent execution
	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Create buffered writer for SSE
	bufWriter := bufio.NewWriter(w)

	// Send RUN_STARTED event
	runStarted := events.NewRunStartedEvent(threadID, runID)
	if err := sseWriter.WriteEvent(ctx, bufWriter, runStarted); err != nil {
		log.Printf("Error writing RUN_STARTED event: %v", err)
		return
	}

	// Generate message ID for this response
	messageID := events.GenerateMessageID()

	// Send TEXT_MESSAGE_START event
	textStart := events.NewTextMessageStartEvent(messageID, events.WithRole("assistant"))
	if err := sseWriter.WriteEvent(ctx, bufWriter, textStart); err != nil {
		log.Printf("Error writing TEXT_MESSAGE_START event: %v", err)
		return
	}

	// Run the agent and stream responses
	if err := h.streamer.StreamResponse(ctx, bufWriter, sseWriter, lastUserMessage, threadID, messageID, h.defaultUID); err != nil {
		log.Printf("Error running agent: %v", err)
		// Send error event using RUN_ERROR
		errorEvent := events.NewRunErrorEvent(err.Error(), events.WithRunID(runID))
		sseWriter.WriteEvent(ctx, bufWriter, errorEvent)
		bufWriter.Flush()
		return
	}

	// Send TEXT_MESSAGE_END event
	textEnd := events.NewTextMessageEndEvent(messageID)
	if err := sseWriter.WriteEvent(ctx, bufWriter, textEnd); err != nil {
		log.Printf("Error writing TEXT_MESSAGE_END event: %v", err)
		return
	}

	// Send RUN_FINISHED event
	runFinished := events.NewRunFinishedEvent(threadID, runID)
	if err := sseWriter.WriteEvent(ctx, bufWriter, runFinished); err != nil {
		log.Printf("Error writing RUN_FINISHED event: %v", err)
		return
	}

	bufWriter.Flush()
}

// extractLastUserMessage extracts the last user message from the messages array
func (h *Handler) extractLastUserMessage(messages []map[string]interface{}) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if role, ok := msg["role"].(string); ok && role == "user" {
			if content, ok := msg["content"].(string); ok {
				return content
			}
		}
	}
	return ""
}
