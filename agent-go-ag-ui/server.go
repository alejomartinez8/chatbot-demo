package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const defaultPort = "8000"

// RunAgentInput represents the AG-UI protocol input format
type RunAgentInput struct {
	ThreadID       string                   `json:"threadId"`
	RunID          string                   `json:"runId"`
	State          map[string]interface{}   `json:"state"`
	Messages       []map[string]interface{} `json:"messages"`
	Tools          []interface{}            `json:"tools"`
	Context        []interface{}            `json:"context"`
	ForwardedProps map[string]interface{}   `json:"forwardedProps"`
}

// sessionCache stores sessions by thread ID
var (
	sessionCache   = make(map[string]session.Session)
	sessionCacheMu sync.RWMutex
	sessionService = session.InMemoryService()
)

// handleAgentRequest handles AG-UI protocol requests
func handleAgentRequest(adkAgent agent.Agent) http.HandlerFunc {
	sseWriter := sse.NewSSEWriter()

	return func(w http.ResponseWriter, r *http.Request) {
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
		var lastUserMessage string
		for i := len(input.Messages) - 1; i >= 0; i-- {
			msg := input.Messages[i]
			if role, ok := msg["role"].(string); ok && role == "user" {
				if content, ok := msg["content"].(string); ok {
					lastUserMessage = content
					break
				}
			}
		}

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

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

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
		if err := streamAgentResponse(ctx, bufWriter, sseWriter, adkAgent, lastUserMessage, threadID, messageID); err != nil {
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
}

// streamAgentResponse executes the ADK agent and streams the response as AG-UI events
func streamAgentResponse(ctx context.Context, w *bufio.Writer, sseWriter *sse.SSEWriter, adkAgent agent.Agent, userMessage string, threadID string, messageID string) error {
	// Create a runner for executing the agent
	r, err := runner.New(runner.Config{
		AppName:        "agent-go-ag-ui",
		Agent:          adkAgent,
		SessionService: sessionService,
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Get or create a session for this thread
	userID := "demo_user"
	var sess session.Session
	var exists bool

	if threadID != "" {
		sessionCacheMu.RLock()
		sess, exists = sessionCache[threadID]
		sessionCacheMu.RUnlock()
	}

	if !exists {
		// Create a new session
		sessResp, err := sessionService.Create(ctx, &session.CreateRequest{
			AppName: "agent-go-ag-ui",
			UserID:  userID,
		})
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		sess = sessResp.Session
		if threadID != "" {
			sessionCacheMu.Lock()
			sessionCache[threadID] = sess
			sessionCacheMu.Unlock()
		}
	}

	// Create user content from the message
	userContent := genai.NewContentFromText(userMessage, genai.RoleUser)

	// Run the agent using the runner
	runConfig := agent.RunConfig{}
	adkEvents := r.Run(ctx, userID, sess.ID(), userContent, runConfig)

	// Stream events as they come from the agent
	var responseBuilder strings.Builder
	for adkEvent, err := range adkEvents {
		if err != nil {
			return fmt.Errorf("agent execution error: %w", err)
		}
		if adkEvent == nil {
			break
		}

		// Extract text from the event's LLMResponse Content
		if adkEvent.Content != nil {
			for _, part := range adkEvent.Content.Parts {
				if part.Text != "" {
					responseBuilder.WriteString(part.Text)

					// Stream the text chunk as TEXT_MESSAGE_CONTENT event
					contentEvent := events.NewTextMessageContentEvent(messageID, part.Text)
					if err := sseWriter.WriteEvent(ctx, w, contentEvent); err != nil {
						return fmt.Errorf("failed to write content event: %w", err)
					}
					w.Flush()
				}
			}
		}

		// Check if this is the final response
		if adkEvent.IsFinalResponse() {
			break
		}
	}

	// If no content was streamed, send a default message
	if responseBuilder.Len() == 0 {
		defaultMsg := "I received your message, but couldn't generate a response."
		contentEvent := events.NewTextMessageContentEvent(messageID, defaultMsg)
		if err := sseWriter.WriteEvent(ctx, w, contentEvent); err != nil {
			return fmt.Errorf("failed to write default content event: %w", err)
		}
		w.Flush()
	}

	return nil
}

// startServer starts the HTTP server
func startServer(adkAgent agent.Agent) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	http.HandleFunc("/", handleAgentRequest(adkAgent))

	log.Printf("Starting AG-UI server on port %s", port)
	log.Printf("Agent will be accessible at http://localhost:%s", port)
	return http.ListenAndServe(":"+port, nil)
}
