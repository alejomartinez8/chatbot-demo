package agui_adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	adksession "google.golang.org/adk/session"
	"google.golang.org/genai"

	"agent-go-ag-ui/internal/session"
	"agent-go-ag-ui/internal/transport"
)

// AGUIAdapter is the SINGLE source of truth for ADK → AG-UI event conversion
type AGUIAdapter struct {
	agent      agent.Agent
	sessionMgr *session.Manager
	appName    string
	timeout    time.Duration
}

// NewAGUIAdapter creates a new AG-UI adapter
func NewAGUIAdapter(agent agent.Agent, sessionMgr *session.Manager, appName string) *AGUIAdapter {
	return &AGUIAdapter{
		agent:      agent,
		sessionMgr: sessionMgr,
		appName:    appName,
		timeout:    60 * time.Second,
	}
}

// RunAgent executes the agent and streams AG-UI events
// This is the SINGLE source of truth for ADK → AG-UI conversion
func (a *AGUIAdapter) RunAgent(
	ctx context.Context,
	input *RunAgentInput,
	threadID, runID, messageID, userID string,
) (<-chan events.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	eventChan := make(chan events.Event, 100)

	go func() {
		defer cancel()
		defer close(eventChan)

		// Create runner
		r, err := runner.New(runner.Config{
			AppName:        a.appName,
			Agent:          a.agent,
			SessionService: a.sessionMgr.Service(),
		})
		if err != nil {
			eventChan <- events.NewRunErrorEvent(fmt.Sprintf("failed to create runner: %v", err), events.WithRunID(runID))
			return
		}

		// Get or create session
		sess, err := a.sessionMgr.GetOrCreate(ctx, a.appName, userID, threadID)
		if err != nil {
			eventChan <- events.NewRunErrorEvent(fmt.Sprintf("failed to get session: %v", err), events.WithRunID(runID))
			return
		}

		// Find last user message
		var lastUserContent *genai.Content
		for i := len(input.Messages) - 1; i >= 0; i-- {
			msg := input.Messages[i]
			role, ok := msg["role"].(string)
			if !ok || role != "user" {
				continue
			}
			content, ok := msg["content"].(string)
			if ok && content != "" {
				lastUserContent = genai.NewContentFromText(content, genai.RoleUser)
				break
			}
		}

		if lastUserContent == nil {
			eventChan <- events.NewRunErrorEvent("no valid user message found", events.WithRunID(runID))
			return
		}

		// Run agent
		runConfig := agent.RunConfig{}
		adkEvents := r.Run(ctx, userID, sess.ID(), lastUserContent, runConfig)

		// Convert ADK events to AG-UI events
		var responseBuilder strings.Builder
		toolCallMap := make(map[string]string)
		startedToolCalls := make(map[string]bool)

		for adkEvent := range adkEvents {
			if adkEvent == nil {
				continue
			}

			// Translate ADK event to AG-UI events
			a.translateADKEvent(adkEvent, messageID, eventChan, &responseBuilder, toolCallMap, startedToolCalls)

			if adkEvent.IsFinalResponse() {
				break
			}
		}

		// Default message if no content
		if responseBuilder.Len() == 0 {
			defaultMsg := "I received your message, but couldn't generate a response."
			eventChan <- events.NewTextMessageContentEvent(messageID, defaultMsg)
		}
	}()

	return eventChan, nil
}

// translateADKEvent converts ADK events to AG-UI events
// This is the core conversion logic, shared by all transports
func (a *AGUIAdapter) translateADKEvent(
	adkEvent *adksession.Event,
	messageID string,
	eventChan chan<- events.Event,
	responseBuilder *strings.Builder,
	toolCallMap map[string]string,
	startedToolCalls map[string]bool,
) {
	if adkEvent == nil {
		return
	}

	if adkEvent.Content == nil {
		return
	}

	for _, part := range adkEvent.Content.Parts {
		// Text content
		if part.Text != "" {
			responseBuilder.WriteString(part.Text)
			eventChan <- events.NewTextMessageContentEvent(messageID, part.Text)
		}

		// Function call (tool call start)
		if part.FunctionCall != nil {
			fc := part.FunctionCall
			agUIToolCallID := fc.ID
			if agUIToolCallID == "" {
				agUIToolCallID = events.GenerateToolCallID()
			}
			toolCallMap[fc.ID] = agUIToolCallID

			eventChan <- events.NewToolCallStartEvent(agUIToolCallID, fc.Name)
			startedToolCalls[agUIToolCallID] = true

			if fc.Args != nil {
				argsJSON, err := json.Marshal(fc.Args)
				if err == nil {
					eventChan <- events.NewToolCallArgsEvent(agUIToolCallID, string(argsJSON))
				}
			}
		}

		// Function response (tool call result)
		if part.FunctionResponse != nil {
			fr := part.FunctionResponse
			agUIToolCallID, exists := toolCallMap[fr.ID]
			if !exists {
				agUIToolCallID = events.GenerateToolCallID()
			}

			resultStr := ""
			if fr.Response != nil {
				if resultBytes, err := json.Marshal(fr.Response); err == nil {
					resultStr = string(resultBytes)
				} else {
					resultStr = fmt.Sprintf("%v", fr.Response)
				}
			}

			eventChan <- events.NewToolCallResultEvent(messageID, agUIToolCallID, resultStr)
			eventChan <- events.NewToolCallEndEvent(agUIToolCallID)
			delete(startedToolCalls, agUIToolCallID)
		}
	}
}

// EventSender defines the interface for sending events (SSE or Connect RPC)
// This allows the adapter to be transport-agnostic
type EventSender interface {
	SendEvent(event events.Event) error
	SendRunError(runID string, err error) error
}

// RunAgentProtocol orchestrates the complete AG-UI protocol flow
// This is the single source of truth for AG-UI protocol logic
// Transport handlers only need to parse requests and serialize events
func (a *AGUIAdapter) RunAgentProtocol(
	ctx context.Context,
	input *RunAgentInput,
	stateMgr *transport.StateManager,
	sender EventSender,
) error {
	// Generate IDs if not provided
	threadID := input.ThreadID
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}
	runID := input.RunID
	if runID == "" {
		runID = events.GenerateRunID()
	}

	// Note: Validation is done in handlers before calling RunAgentProtocol
	// This ensures fail-fast behavior and proper HTTP error codes

	// Handle state persistence: merge incoming state with existing state for this thread
	mergedState := stateMgr.Merge(threadID, input.State)

	// If no messages, send current state snapshot according to AG-UI protocol
	if len(input.Messages) == 0 {
		stateSnapshot := events.NewStateSnapshotEvent(mergedState)
		return sender.SendEvent(stateSnapshot)
	}

	// Send RUN_STARTED event
	runStarted := events.NewRunStartedEvent(threadID, runID)
	if err := sender.SendEvent(runStarted); err != nil {
		return fmt.Errorf("failed to send RUN_STARTED: %w", err)
	}

	// Generate message ID for this response
	messageID := events.GenerateMessageID()

	// Send TEXT_MESSAGE_START event
	textStart := events.NewTextMessageStartEvent(messageID, events.WithRole("assistant"))
	if err := sender.SendEvent(textStart); err != nil {
		return fmt.Errorf("failed to send TEXT_MESSAGE_START: %w", err)
	}

	// Run the agent and stream responses
	eventChan, err := a.RunAgent(ctx, input, threadID, runID, messageID, "demo_user")
	if err != nil {
		// If message was started, we must send TEXT_MESSAGE_END before RUN_ERROR
		textEnd := events.NewTextMessageEndEvent(messageID)
		sender.SendEvent(textEnd) // Best effort, ignore error

		// Send error event
		return sender.SendRunError(runID, fmt.Errorf("agent execution failed: %w", err))
	}

	// Stream events from the adapter
	for event := range eventChan {
		if err := sender.SendEvent(event); err != nil {
			return fmt.Errorf("failed to send event: %w", err)
		}
	}

	// Send TEXT_MESSAGE_END event
	textEnd := events.NewTextMessageEndEvent(messageID)
	if err := sender.SendEvent(textEnd); err != nil {
		return fmt.Errorf("failed to send TEXT_MESSAGE_END: %w", err)
	}

	// Send RUN_FINISHED event
	runFinished := events.NewRunFinishedEvent(threadID, runID)
	if err := sender.SendEvent(runFinished); err != nil {
		return fmt.Errorf("failed to send RUN_FINISHED: %w", err)
	}

	return nil
}
