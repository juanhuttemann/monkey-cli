// Package api defines data structures for API communication
package api

// apiRequest represents the request body sent to the LLM API
type apiRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
}

// Message represents a single message in the conversation.
// Content may be a plain string or a []ContentBlock (for tool_use/tool_result).
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// apiResponse represents the response from the LLM API
type apiResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ContentBlock represents a single content block in a message or response.
// Type is one of "text", "tool_use", or "tool_result".
type ContentBlock struct {
	Type string `json:"type"`

	// text blocks
	Text string `json:"text,omitempty"`

	// tool_use blocks (assistant → us)
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`

	// tool_result blocks (us → assistant)
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// Tool describes a tool the model may call.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

// InputSchema is the JSON Schema for a tool's input object.
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]PropertyDef `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// PropertyDef describes a single property in an InputSchema.
type PropertyDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolExecutor executes a named tool with the given input and returns output.
type ToolExecutor interface {
	ExecuteTool(name string, input map[string]any) (string, error)
}

// ToolCallResult records a single tool execution that occurred during a request.
type ToolCallResult struct {
	Name   string
	Input  map[string]any
	Output string
	Err    error
}
