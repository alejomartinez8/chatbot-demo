package agui

import (
	"bufio"
	"context"
	"encoding/json"
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
// It processes all messages from the conversation history, not just the last one
func (s *Streamer) StreamResponse(ctx context.Context, w *bufio.Writer, sseWriter *sse.SSEWriter, messages []map[string]interface{}, threadID, messageID, userID string) error {
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

	// Get or create a session for this thread
	// Use threadID as the session ID to reuse sessions for the same thread
	sess, err := s.sessionMgr.GetOrCreate(ctx, s.appName, userID, threadID)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}

	// Convert all messages from AG-UI format to ADK genai.Content format
	// We need to extract the last user message for the current run, but the session
	// will maintain the conversation history
	var lastUserContent *genai.Content

	// Process messages in order and find the last user message
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		role, ok := msg["role"].(string)
		if !ok {
			continue
		}

		if role == "user" {
			content, ok := msg["content"].(string)
			if ok && content != "" {
				// Found the last user message - this is what we'll send to the agent
				lastUserContent = genai.NewContentFromText(content, genai.RoleUser)
				break
			}
		}
	}

	// If no user message found, return an error
	if lastUserContent == nil {
		return fmt.Errorf("no valid user message found in messages")
	}

	// Run the agent using the runner
	// The session maintains conversation history, so we only need to send the latest user message
	// The ADK will handle the conversation context through the session
	runConfig := agent.RunConfig{}
	adkEvents := r.Run(ctx, userID, sess.ID(), lastUserContent, runConfig)

	// Stream events as they come from the agent
	var responseBuilder strings.Builder
	// Map to track tool calls by their ID (from FunctionCall.ID)
	toolCallMap := make(map[string]string) // ADK function call ID -> AG-UI tool call ID
	// Track started tool calls that need to be closed on error
	startedToolCalls := make(map[string]bool) // AG-UI tool call ID -> started

	// Cleanup function to close all started tool calls on error
	closeStartedToolCalls := func() {
		for toolCallID := range startedToolCalls {
			toolCallEnd := events.NewToolCallEndEvent(toolCallID)
			sseWriter.WriteEvent(ctx, w, toolCallEnd)
		}
		w.Flush()
	}

	// Process events from the ADK runner
	// The runner returns a channel of *session.Event
	for adkEvent := range adkEvents {
		if adkEvent == nil {
			continue
		}

		// Extract text from the event's LLMResponse Content
		if adkEvent.Content != nil {
			for _, part := range adkEvent.Content.Parts {
				if part.Text != "" {
					responseBuilder.WriteString(part.Text)

					// Stream the text chunk as TEXT_MESSAGE_CONTENT event
					contentEvent := events.NewTextMessageContentEvent(messageID, part.Text)
					if err := sseWriter.WriteEvent(ctx, w, contentEvent); err != nil {
						closeStartedToolCalls()
						return fmt.Errorf("failed to write content event: %w", err)
					}
					w.Flush()
				}

				// Handle function calls (tool calls)
				if part.FunctionCall != nil {
					fc := part.FunctionCall
					// Use ADK's function call ID if available, otherwise generate one
					agUIToolCallID := fc.ID
					if agUIToolCallID == "" {
						agUIToolCallID = events.GenerateToolCallID()
					}
					// Store mapping for later when we get the response
					toolCallMap[fc.ID] = agUIToolCallID

					// Send TOOL_CALL_START event
					toolCallStart := events.NewToolCallStartEvent(
						agUIToolCallID,
						fc.Name,
					)
					if err := sseWriter.WriteEvent(ctx, w, toolCallStart); err != nil {
						closeStartedToolCalls()
						return fmt.Errorf("failed to write tool call start event: %w", err)
					}
					startedToolCalls[agUIToolCallID] = true
					w.Flush()

					// Convert tool arguments to JSON and send TOOL_CALL_ARGS event
					if fc.Args != nil {
						argsJSON, err := json.Marshal(fc.Args)
						if err != nil {
							closeStartedToolCalls()
							return fmt.Errorf("failed to marshal tool args: %w", err)
						}

						toolCallArgsEvent := events.NewToolCallArgsEvent(
							agUIToolCallID,
							string(argsJSON),
						)
						if err := sseWriter.WriteEvent(ctx, w, toolCallArgsEvent); err != nil {
							closeStartedToolCalls()
							return fmt.Errorf("failed to write tool call args event: %w", err)
						}
						w.Flush()
					}
				}

				// Handle function responses (tool results)
				if part.FunctionResponse != nil {
					fr := part.FunctionResponse
					// Look up the corresponding AG-UI tool call ID
					agUIToolCallID, exists := toolCallMap[fr.ID]
					if !exists {
						// If we don't have a mapping, generate a new ID
						agUIToolCallID = events.GenerateToolCallID()
					}

					// Convert response to string
					resultStr := ""
					if fr.Response != nil {
						if resultBytes, err := json.Marshal(fr.Response); err == nil {
							resultStr = string(resultBytes)
						} else {
							resultStr = fmt.Sprintf("%v", fr.Response)
						}
					}

					// Send TOOL_CALL_RESULT event (requires messageID, toolCallID, content)
					toolCallResult := events.NewToolCallResultEvent(
						messageID,
						agUIToolCallID,
						resultStr,
					)
					if err := sseWriter.WriteEvent(ctx, w, toolCallResult); err != nil {
						closeStartedToolCalls()
						return fmt.Errorf("failed to write tool call result event: %w", err)
					}
					w.Flush()

					// Send TOOL_CALL_END event
					toolCallEnd := events.NewToolCallEndEvent(agUIToolCallID)
					if err := sseWriter.WriteEvent(ctx, w, toolCallEnd); err != nil {
						return fmt.Errorf("failed to write tool call end event: %w", err)
					}
					delete(startedToolCalls, agUIToolCallID) // Mark as closed
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
