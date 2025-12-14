package agui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	aguiv1 "agent-go-ag-ui/gen/proto/agui/v1"

	"connectrpc.com/connect"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/structpb"
)

// ConnectHandler handles Connect RPC requests for the AG-UI protocol
type ConnectHandler struct {
	agent      agent.Agent
	streamer   *Streamer
	stateMgr   *StateManager
	appName    string
	defaultUID string
}

// NewConnectHandler creates a new Connect RPC handler
func NewConnectHandler(agent agent.Agent, streamer *Streamer, stateMgr *StateManager, appName string) *ConnectHandler {
	return &ConnectHandler{
		agent:      agent,
		streamer:   streamer,
		stateMgr:   stateMgr,
		appName:    appName,
		defaultUID: "demo_user",
	}
}

// RunAgent implements the AGUIService.RunAgent RPC method
func (h *ConnectHandler) RunAgent(
	ctx context.Context,
	req *aguiv1.RunAgentRequest,
	stream *connect.ServerStream[aguiv1.AGUIEvent],
) error {
	// Convert protobuf request to internal RunAgentInput
	runInput, err := h.convertRunAgentRequest(req)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to convert request: %w", err))
	}

	// Use IDs from input or generate new ones
	threadID := runInput.ThreadID
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}
	runID := runInput.RunID
	if runID == "" {
		runID = events.GenerateRunID()
	}

	// Validate messages (reuse validation from Handler)
	handler := NewHandler(h.agent, h.streamer, h.stateMgr, h.appName)
	if err := handler.ValidateMessages(runInput.Messages); err != nil {
		errorEvent := events.NewRunErrorEvent("Invalid messages: "+err.Error(), events.WithRunID(runID))
		aguiEvent, err := h.convertAGUIEvent(errorEvent)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert error event: %w", err))
		}
		if err := stream.Send(aguiEvent); err != nil {
			return fmt.Errorf("failed to send error event: %w", err)
		}
		return nil
	}

	// Handle state persistence: merge incoming state with existing state for this thread
	mergedState := h.stateMgr.Merge(threadID, runInput.State)

	// If no messages, send current state snapshot according to AG-UI protocol
	if len(runInput.Messages) == 0 {
		stateSnapshot := events.NewStateSnapshotEvent(mergedState)
		aguiEvent, err := h.convertAGUIEvent(stateSnapshot)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert state snapshot: %w", err))
		}
		if err := stream.Send(aguiEvent); err != nil {
			return fmt.Errorf("failed to send state snapshot: %w", err)
		}
		return nil
	}

	// Send RUN_STARTED event
	runStarted := events.NewRunStartedEvent(threadID, runID)
	aguiEvent, err := h.convertAGUIEvent(runStarted)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert run started event: %w", err))
	}
	if err := stream.Send(aguiEvent); err != nil {
		return fmt.Errorf("failed to send run started event: %w", err)
	}

	// Generate message ID for this response
	messageID := events.GenerateMessageID()

	// Send TEXT_MESSAGE_START event
	textStart := events.NewTextMessageStartEvent(messageID, events.WithRole("assistant"))
	aguiEvent, err = h.convertAGUIEvent(textStart)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert text message start event: %w", err))
	}
	if err := stream.Send(aguiEvent); err != nil {
		return fmt.Errorf("failed to send text message start event: %w", err)
	}

	// Create a channel to receive events from the streamer
	eventChan := make(chan events.Event, 100)
	errorChan := make(chan error, 1)

	// Run the agent in a goroutine and collect events
	go func() {
		defer close(eventChan)
		defer close(errorChan)

		// We need to adapt the streamer to send events to our channel
		// For now, we'll use a wrapper that collects events
		err := h.streamAgentResponse(ctx, runInput.Messages, threadID, messageID, h.defaultUID, eventChan)
		if err != nil {
			errorChan <- err
		}
	}()

	// Stream events as they come
	messageStarted := true
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errorChan:
			if err != nil {
				// Send TEXT_MESSAGE_END before RUN_ERROR if message was started
				if messageStarted {
					textEnd := events.NewTextMessageEndEvent(messageID)
					aguiEvent, err := h.convertAGUIEvent(textEnd)
					if err == nil {
						stream.Send(aguiEvent)
					}
				}

				// Send error event
				errorEvent := events.NewRunErrorEvent(err.Error(), events.WithRunID(runID))
				aguiEvent, err := h.convertAGUIEvent(errorEvent)
				if err != nil {
					return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert error event: %w", err))
				}
				if err := stream.Send(aguiEvent); err != nil {
					return fmt.Errorf("failed to send error event: %w", err)
				}
				return nil
			}
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed, send final events
				textEnd := events.NewTextMessageEndEvent(messageID)
				aguiEvent, err := h.convertAGUIEvent(textEnd)
				if err != nil {
					return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert text message end event: %w", err))
				}
				if err := stream.Send(aguiEvent); err != nil {
					return fmt.Errorf("failed to send text message end event: %w", err)
				}

				runFinished := events.NewRunFinishedEvent(threadID, runID)
				aguiEvent, err = h.convertAGUIEvent(runFinished)
				if err != nil {
					return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert run finished event: %w", err))
				}
				if err := stream.Send(aguiEvent); err != nil {
					return fmt.Errorf("failed to send run finished event: %w", err)
				}
				return nil
			}

			// Convert and send event
			aguiEvent, err := h.convertAGUIEvent(event)
			if err != nil {
				log.Printf("Failed to convert event: %v", err)
				continue
			}
			if err := stream.Send(aguiEvent); err != nil {
				return fmt.Errorf("failed to send event: %w", err)
			}
		}
	}
}

// convertRunAgentRequest converts a protobuf RunAgentRequest to internal RunAgentInput
func (h *ConnectHandler) convertRunAgentRequest(req *aguiv1.RunAgentRequest) (*RunAgentInput, error) {
	// Convert state
	state := make(map[string]interface{})
	if req.State != nil {
		state = req.State.AsMap()
	}

	// Convert messages
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		msgMap := make(map[string]interface{})
		msgMap["id"] = msg.Id
		msgMap["role"] = msg.Role
		if msg.Content != nil {
			// Convert protobuf Value to interface{}
			var content interface{}
			if err := json.Unmarshal([]byte(msg.Content.String()), &content); err != nil {
				// Fallback: use the value directly
				content = msg.Content.AsInterface()
			}
			msgMap["content"] = content
		}
		if msg.Name != "" {
			msgMap["name"] = msg.Name
		}
		if msg.ToolCalls != nil {
			msgMap["tool_calls"] = msg.ToolCalls.AsInterface()
		}
		messages = append(messages, msgMap)
	}

	// Convert tools
	tools := make([]interface{}, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, tool.AsInterface())
	}

	// Convert context
	context := make([]interface{}, 0, len(req.Context))
	for _, ctxItem := range req.Context {
		context = append(context, ctxItem.AsInterface())
	}

	// Convert forwarded props
	forwardedProps := make(map[string]interface{})
	if req.ForwardedProps != nil {
		forwardedProps = req.ForwardedProps.AsMap()
	}

	return &RunAgentInput{
		ThreadID:       req.ThreadId,
		RunID:          req.RunId,
		State:          state,
		Messages:       messages,
		Tools:          tools,
		Context:        context,
		ForwardedProps: forwardedProps,
	}, nil
}

// convertAGUIEvent converts an AG-UI event to protobuf AGUIEvent
func (h *ConnectHandler) convertAGUIEvent(event events.Event) (*aguiv1.AGUIEvent, error) {
	// Serialize event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	// Parse JSON into a map
	var eventMap map[string]interface{}
	if err := json.Unmarshal(eventJSON, &eventMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event JSON: %w", err)
	}

	// Convert map to protobuf Struct
	eventStruct, err := structpb.NewStruct(eventMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create struct: %w", err)
	}

	// Extract event type
	eventType := ""
	if t, ok := eventMap["type"].(string); ok {
		eventType = t
	}

	return &aguiv1.AGUIEvent{
		Type: eventType,
		Data: eventStruct,
	}, nil
}

// streamAgentResponse runs the agent and sends events to the channel
// This reuses the core logic from Streamer but outputs to a channel instead of SSE
func (h *ConnectHandler) streamAgentResponse(
	ctx context.Context,
	messages []map[string]interface{},
	threadID, messageID, userID string,
	eventChan chan<- events.Event,
) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, h.streamer.timeout)
	defer cancel()

	// Create a runner for executing the agent
	r, err := runner.New(runner.Config{
		AppName:        h.appName,
		Agent:          h.agent,
		SessionService: h.streamer.sessionMgr.Service(),
	})
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Get or create a session for this thread
	sess, err := h.streamer.sessionMgr.GetOrCreate(ctx, h.appName, userID, threadID)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}

	// Find the last user message
	var lastUserContent *genai.Content
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		role, ok := msg["role"].(string)
		if !ok {
			continue
		}

		if role == "user" {
			content, ok := msg["content"].(string)
			if ok && content != "" {
				lastUserContent = genai.NewContentFromText(content, genai.RoleUser)
				break
			}
		}
	}

	if lastUserContent == nil {
		return fmt.Errorf("no valid user message found in messages")
	}

	// Run the agent
	runConfig := agent.RunConfig{}
	adkEvents := r.Run(ctx, userID, sess.ID(), lastUserContent, runConfig)

	// Process events and send to channel
	var responseBuilder strings.Builder
	toolCallMap := make(map[string]string)
	startedToolCalls := make(map[string]bool)

	for adkEvent := range adkEvents {
		if adkEvent == nil {
			continue
		}

		if adkEvent.Content != nil {
			for _, part := range adkEvent.Content.Parts {
				if part.Text != "" {
					responseBuilder.WriteString(part.Text)
					contentEvent := events.NewTextMessageContentEvent(messageID, part.Text)
					eventChan <- contentEvent
				}

				if part.FunctionCall != nil {
					fc := part.FunctionCall
					agUIToolCallID := fc.ID
					if agUIToolCallID == "" {
						agUIToolCallID = events.GenerateToolCallID()
					}
					toolCallMap[fc.ID] = agUIToolCallID

					toolCallStart := events.NewToolCallStartEvent(agUIToolCallID, fc.Name)
					eventChan <- toolCallStart
					startedToolCalls[agUIToolCallID] = true

					if fc.Args != nil {
						argsJSON, err := json.Marshal(fc.Args)
						if err != nil {
							return fmt.Errorf("failed to marshal tool args: %w", err)
						}
						toolCallArgsEvent := events.NewToolCallArgsEvent(agUIToolCallID, string(argsJSON))
						eventChan <- toolCallArgsEvent
					}
				}

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

					toolCallResult := events.NewToolCallResultEvent(messageID, agUIToolCallID, resultStr)
					eventChan <- toolCallResult

					toolCallEnd := events.NewToolCallEndEvent(agUIToolCallID)
					eventChan <- toolCallEnd
					delete(startedToolCalls, agUIToolCallID)
				}
			}
		}

		if adkEvent.IsFinalResponse() {
			break
		}
	}

	// If no content was streamed, send a default message
	if responseBuilder.Len() == 0 {
		defaultMsg := "I received your message, but couldn't generate a response."
		contentEvent := events.NewTextMessageContentEvent(messageID, defaultMsg)
		eventChan <- contentEvent
	}

	return nil
}
