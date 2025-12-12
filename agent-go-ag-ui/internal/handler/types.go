package handler

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
