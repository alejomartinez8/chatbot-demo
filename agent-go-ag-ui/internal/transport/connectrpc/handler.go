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
	"agent-go-ag-ui/internal/transport"
)

// Handler handles Connect RPC requests for the AG-UI protocol
// Only responsible for Protobuf serialization - protocol logic is in agui_adapter
type Handler struct {
	adapter  *agui_adapter.AGUIAdapter
	stateMgr *transport.StateManager
}

// NewHandler creates a new Connect RPC handler
func NewHandler(adapter *agui_adapter.AGUIAdapter, stateMgr *transport.StateManager) *Handler {
	return &Handler{
		adapter:  adapter,
		stateMgr: stateMgr,
	}
}

// connectEventSender implements agui_adapter.EventSender for Connect RPC transport
type connectEventSender struct {
	stream *connect.ServerStream[aguiv1.AGUIEvent]
}

func (c *connectEventSender) SendEvent(event events.Event) error {
	aguiEvent, err := convertAGUIEvent(event)
	if err != nil {
		return fmt.Errorf("failed to convert event: %w", err)
	}
	return c.stream.Send(aguiEvent)
}

func (c *connectEventSender) SendRunError(runID string, err error) error {
	errorEvent := events.NewRunErrorEvent(err.Error(), events.WithRunID(runID))
	return c.SendEvent(errorEvent)
}

// RunAgent implements the AGUIService.RunAgent RPC method
func (h *Handler) RunAgent(
	ctx context.Context,
	req *aguiv1.RunAgentInput,
	stream *connect.ServerStream[aguiv1.AGUIEvent],
) error {
	// Convert protobuf RunAgentInput to agui_adapter.RunAgentInput
	runInput, err := h.convertRunAgentInput(req)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to convert request: %w", err))
	}

	// Validate input early (fail fast)
	if err := runInput.Validate(); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("validation failed: %w", err))
	}

	// Create Connect RPC event sender
	sender := &connectEventSender{stream: stream}

	// Delegate protocol logic to adapter
	if err := h.adapter.RunAgentProtocol(ctx, runInput, h.stateMgr, sender); err != nil {
		log.Printf("Error running agent protocol: %v", err)
		// Error already sent via sender.SendRunError, but we need to return a Connect error
		return connect.NewError(connect.CodeInternal, err)
	}

	return nil
}

// convertRunAgentInput converts a protobuf RunAgentInput to agui_adapter.RunAgentInput
func (h *Handler) convertRunAgentInput(req *aguiv1.RunAgentInput) (*agui_adapter.RunAgentInput, error) {
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

	return &agui_adapter.RunAgentInput{
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
func convertAGUIEvent(event events.Event) (*aguiv1.AGUIEvent, error) {
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
