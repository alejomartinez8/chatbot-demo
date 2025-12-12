package stream

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/encoding/sse"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/genai"

	"agent-go-ag-ui/internal/session"
)

// Streamer handles streaming agent responses
type Streamer struct {
	agent      agent.Agent
	sessionMgr *session.Manager
	appName    string
	timeout    time.Duration
}

// NewStreamer creates a new streamer
func NewStreamer(agent agent.Agent, sessionMgr *session.Manager, appName string) *Streamer {
	return &Streamer{
		agent:      agent,
		sessionMgr: sessionMgr,
		appName:    appName,
		timeout:    60 * time.Second,
	}
}

// StreamResponse executes the ADK agent and streams the response as AG-UI events
func (s *Streamer) StreamResponse(ctx context.Context, w *bufio.Writer, sseWriter *sse.SSEWriter, userMessage, threadID, messageID, userID string) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Create a runner for executing the agent
	r, err := runner.New(runner.Config{
		AppName:        s.appName,
		Agent:          s.agent,
		SessionService: s.sessionMgr.Service(),
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Create a new session for this request
	sess, err := s.sessionMgr.Create(ctx, s.appName, userID)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
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
