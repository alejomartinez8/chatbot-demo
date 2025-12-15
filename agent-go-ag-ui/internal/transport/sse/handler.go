package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"agent-go-ag-ui/internal/agui_adapter"
	"agent-go-ag-ui/internal/domain"
	"agent-go-ag-ui/internal/transport"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

// Handler handles HTTP requests for the AG-UI protocol via SSE
type Handler struct {
	adapter  *agui_adapter.AGUIAdapter
	stateMgr *transport.StateManager
	appName  string
}

// NewHandler creates a new SSE handler
func NewHandler(adapter *agui_adapter.AGUIAdapter, stateMgr *transport.StateManager, appName string) *Handler {
	return &Handler{
		adapter:  adapter,
		stateMgr: stateMgr,
		appName:  appName,
	}
}

// writeSSEEvent writes an event in SSE format: "data: {json}\n\n"
func (h *Handler) writeSSEEvent(w io.Writer, event events.Event) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", eventJSON)
	return err
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
	var input domain.RunAgentInput
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

	// Validate messages
	if err := h.ValidateMessages(input.Messages); err != nil {
		errorEvent := events.NewRunErrorEvent("Invalid messages: "+err.Error(), events.WithRunID(runID))
		bufWriter := bufio.NewWriter(w)
		if err := h.writeSSEEvent(bufWriter, errorEvent); err != nil {
			log.Printf("Error writing validation error event: %v", err)
		}
		bufWriter.Flush()
		return
	}

	// Handle state persistence: merge incoming state with existing state for this thread
	mergedState := h.stateMgr.Merge(threadID, input.State)

	// If no messages, send current state snapshot according to AG-UI protocol
	// This allows the frontend to synchronize state on initial connection
	if len(input.Messages) == 0 {
		// Send STATE_SNAPSHOT event with current state (official AG-UI protocol event)
		stateSnapshot := events.NewStateSnapshotEvent(mergedState)
		bufWriter := bufio.NewWriter(w)
		if err := h.writeSSEEvent(bufWriter, stateSnapshot); err != nil {
			log.Printf("Error writing state snapshot event: %v", err)
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
	if err := h.writeSSEEvent(bufWriter, runStarted); err != nil {
		log.Printf("Error writing RUN_STARTED event: %v", err)
		return
	}

	// Generate message ID for this response
	messageID := events.GenerateMessageID()
	messageStarted := false

	// Send TEXT_MESSAGE_START event
	textStart := events.NewTextMessageStartEvent(messageID, events.WithRole("assistant"))
	if err := h.writeSSEEvent(bufWriter, textStart); err != nil {
		log.Printf("Error writing TEXT_MESSAGE_START event: %v", err)
		return
	}
	messageStarted = true

	// Run the agent and stream responses using the adapter
	eventChan, err := h.adapter.RunAgent(ctx, &input, threadID, runID, messageID, "demo_user")
	if err != nil {
		log.Printf("Error running agent: %v", err)

		// If message was started, we must send TEXT_MESSAGE_END before RUN_ERROR
		if messageStarted {
			textEnd := events.NewTextMessageEndEvent(messageID)
			if err := h.writeSSEEvent(bufWriter, textEnd); err != nil {
				log.Printf("Error writing TEXT_MESSAGE_END event after error: %v", err)
			}
			bufWriter.Flush()
		}

		// Send error event using RUN_ERROR
		errorEvent := events.NewRunErrorEvent(err.Error(), events.WithRunID(runID))
		if err := h.writeSSEEvent(bufWriter, errorEvent); err != nil {
			log.Printf("Error writing RUN_ERROR event: %v", err)
		}
		bufWriter.Flush()
		return
	}

	// Stream events from the adapter
	for event := range eventChan {
		if err := h.writeSSEEvent(bufWriter, event); err != nil {
			log.Printf("Error encoding event: %v", err)
			break
		}
		bufWriter.Flush()
	}

	// Send TEXT_MESSAGE_END event
	textEnd := events.NewTextMessageEndEvent(messageID)
	if err := h.writeSSEEvent(bufWriter, textEnd); err != nil {
		log.Printf("Error writing TEXT_MESSAGE_END event: %v", err)
		return
	}

	// Send RUN_FINISHED event
	runFinished := events.NewRunFinishedEvent(threadID, runID)
	if err := h.writeSSEEvent(bufWriter, runFinished); err != nil {
		log.Printf("Error writing RUN_FINISHED event: %v", err)
		return
	}

	bufWriter.Flush()
}

// ValidateMessages validates that messages have the required structure
func (h *Handler) ValidateMessages(messages []map[string]interface{}) error {
	for i, msg := range messages {
		if msg == nil {
			return fmt.Errorf("message at index %d is nil", i)
		}

		// Check for required fields
		id, hasID := msg["id"]
		if !hasID || id == nil || id == "" {
			return fmt.Errorf("message at index %d missing required field 'id'", i)
		}

		role, hasRole := msg["role"]
		if !hasRole || role == nil {
			return fmt.Errorf("message at index %d missing required field 'role'", i)
		}

		roleStr, ok := role.(string)
		if !ok {
			return fmt.Errorf("message at index %d has invalid 'role' type (expected string)", i)
		}

		// Validate role value
		validRoles := map[string]bool{
			"user":      true,
			"assistant": true,
			"system":    true,
			"developer": true,
			"tool":      true,
		}
		if !validRoles[roleStr] {
			return fmt.Errorf("message at index %d has invalid 'role' value: %s", i, roleStr)
		}

		// Check for content field (required for user and assistant messages)
		if roleStr == "user" || roleStr == "assistant" {
			content, hasContent := msg["content"]
			if !hasContent || content == nil {
				return fmt.Errorf("message at index %d missing required field 'content' for role '%s'", i, roleStr)
			}

			// Content should be a string or array
			if _, ok := content.(string); !ok {
				if _, ok := content.([]interface{}); !ok {
					return fmt.Errorf("message at index %d has invalid 'content' type (expected string or array)", i)
				}
			}
		}
	}

	return nil
}
