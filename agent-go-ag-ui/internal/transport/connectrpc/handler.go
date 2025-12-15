package connectrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	aguiv1 "agent-go-ag-ui/gen/proto/agui/v1"

	"connectrpc.com/connect"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"google.golang.org/protobuf/types/known/structpb"

	"agent-go-ag-ui/internal/agui_adapter"
	"agent-go-ag-ui/internal/domain"
	"agent-go-ag-ui/internal/transport"
)

// Handler handles Connect RPC requests for the AG-UI protocol
type Handler struct {
	adapter  *agui_adapter.AGUIAdapter
	stateMgr *transport.StateManager
	appName  string
}

// NewHandler creates a new Connect RPC handler
func NewHandler(adapter *agui_adapter.AGUIAdapter, stateMgr *transport.StateManager, appName string) *Handler {
	return &Handler{
		adapter:  adapter,
		stateMgr: stateMgr,
		appName:  appName,
	}
}

// RunAgent implements the AGUIService.RunAgent RPC method
func (h *Handler) RunAgent(
	ctx context.Context,
	req *aguiv1.RunAgentInput,
	stream *connect.ServerStream[aguiv1.AGUIEvent],
) error {
	// Convert protobuf RunAgentInput to domain.RunAgentInput
	runInput, err := h.convertRunAgentInput(req)
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

	// Validate messages
	if err := h.validateMessages(runInput.Messages); err != nil {
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

	// Run the agent using the shared adapter
	eventChan, err := h.adapter.RunAgent(ctx, runInput, threadID, runID, messageID, "demo_user")
	if err != nil {
		// Send TEXT_MESSAGE_END before RUN_ERROR if message was started
		textEnd := events.NewTextMessageEndEvent(messageID)
		aguiEvent, err := h.convertAGUIEvent(textEnd)
		if err == nil {
			stream.Send(aguiEvent)
		}

		// Send error event
		errorEvent := events.NewRunErrorEvent(err.Error(), events.WithRunID(runID))
		aguiEvent, err = h.convertAGUIEvent(errorEvent)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert error event: %w", err))
		}
		if err := stream.Send(aguiEvent); err != nil {
			return fmt.Errorf("failed to send error event: %w", err)
		}
		return nil
	}

	// Stream events as they come from the adapter
	messageStarted := true
	for event := range eventChan {
		// Convert and send event
		aguiEvent, err := h.convertAGUIEvent(event)
		if err != nil {
			log.Printf("Failed to convert event: %v", err)
			continue
		}
		if err := stream.Send(aguiEvent); err != nil {
			return fmt.Errorf("failed to send event: %w", err)
		}
		messageStarted = true
	}

	// Send TEXT_MESSAGE_END event
	if messageStarted {
		textEnd := events.NewTextMessageEndEvent(messageID)
		aguiEvent, err := h.convertAGUIEvent(textEnd)
		if err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to convert text message end event: %w", err))
		}
		if err := stream.Send(aguiEvent); err != nil {
			return fmt.Errorf("failed to send text message end event: %w", err)
		}
	}

	// Send RUN_FINISHED event
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

// convertRunAgentInput converts a protobuf RunAgentInput to domain.RunAgentInput
func (h *Handler) convertRunAgentInput(req *aguiv1.RunAgentInput) (*domain.RunAgentInput, error) {
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
		toolMap := make(map[string]interface{})
		toolMap["name"] = tool.Name
		toolMap["description"] = tool.Description
		if tool.Parameters != nil {
			toolMap["parameters"] = tool.Parameters.AsMap()
		}
		tools = append(tools, toolMap)
	}

	// Convert context
	context := make([]interface{}, 0, len(req.Context))
	for _, ctxItem := range req.Context {
		ctxMap := make(map[string]interface{})
		ctxMap["description"] = ctxItem.Description
		ctxMap["value"] = ctxItem.Value
		context = append(context, ctxMap)
	}

	// Convert forwarded props
	forwardedProps := make(map[string]interface{})
	if req.ForwardedProps != nil {
		forwardedProps = req.ForwardedProps.AsMap()
	}

	return &domain.RunAgentInput{
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
func (h *Handler) convertAGUIEvent(event events.Event) (*aguiv1.AGUIEvent, error) {
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

// validateMessages validates that messages have the required structure
func (h *Handler) validateMessages(messages []map[string]interface{}) error {
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

