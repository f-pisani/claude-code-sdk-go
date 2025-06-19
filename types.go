package claudecode

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/f-pisani/claude-code-sdk-go/internal/validation"
)

// PermissionMode represents the permission mode for tool execution
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// McpServerConfig represents MCP server configuration
type McpServerConfig struct {
	Transport []string               `json:"transport"`
	Env       map[string]interface{} `json:"env,omitempty"`
}

// ContentBlock represents different types of content blocks
type ContentBlock interface {
	isContentBlock()
}

// TextBlock represents text content
type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) isContentBlock() {}

// ToolUseBlock represents tool usage
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (ToolUseBlock) isContentBlock() {}

// ToolResultBlock represents tool execution result
type ToolResultBlock struct {
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content,omitempty"` // string or []map[string]interface{}
	IsError   *bool       `json:"is_error,omitempty"`
}

func (ToolResultBlock) isContentBlock() {}

// Message represents different types of messages
type Message interface {
	isMessage()
}

// UserMessage represents a message from the user
type UserMessage struct {
	Content string `json:"content"`
}

func (UserMessage) isMessage() {}

// AssistantMessage represents a message from the assistant
type AssistantMessage struct {
	Content []ContentBlock `json:"content"`
}

func (AssistantMessage) isMessage() {}

// SystemMessage represents a system message with metadata
type SystemMessage struct {
	Subtype string                 `json:"subtype"`
	Data    map[string]interface{} `json:"data"`
}

func (SystemMessage) isMessage() {}

// ResultMessage represents the final result with cost and usage information
type ResultMessage struct {
	Subtype       string                 `json:"subtype"`
	DurationMs    int                    `json:"duration_ms"`
	DurationAPIMs int                    `json:"duration_api_ms"`
	IsError       bool                   `json:"is_error"`
	NumTurns      int                    `json:"num_turns"`
	SessionID     string                 `json:"session_id"`
	TotalCostUSD  *float64               `json:"total_cost_usd,omitempty"`
	Usage         map[string]interface{} `json:"usage,omitempty"`
	Result        *string                `json:"result,omitempty"`
}

func (ResultMessage) isMessage() {}

// Options represents configuration options for Claude Code
type Options struct {
	AllowedTools             []string                   `json:"allowed_tools,omitempty"`
	MaxThinkingTokens        int                        `json:"max_thinking_tokens"`
	SystemPrompt             string                     `json:"system_prompt,omitempty"`
	AppendSystemPrompt       string                     `json:"append_system_prompt,omitempty"`
	McpTools                 []string                   `json:"mcp_tools,omitempty"`
	McpServers               map[string]McpServerConfig `json:"mcp_servers,omitempty"`
	PermissionMode           *PermissionMode            `json:"permission_mode,omitempty"`
	ContinueConversation     bool                       `json:"continue_conversation,omitempty"`
	Resume                   string                     `json:"resume,omitempty"`
	MaxTurns                 *int                       `json:"max_turns,omitempty"`
	DisallowedTools          []string                   `json:"disallowed_tools,omitempty"`
	Model                    string                     `json:"model,omitempty"`
	PermissionPromptToolName string                     `json:"permission_prompt_tool_name,omitempty"`
	Cwd                      string                     `json:"cwd,omitempty"`
	MessageBufferSize        int                        `json:"message_buffer_size,omitempty"`
	ErrorBufferSize          int                        `json:"error_buffer_size,omitempty"`
	QueryTimeout             int                        `json:"query_timeout,omitempty"` // Timeout in seconds for the entire query
}

// NewOptions creates a new Options instance with default values
func NewOptions() *Options {
	return &Options{
		MaxThinkingTokens: 8000,
		AllowedTools:      []string{},
		DisallowedTools:   []string{},
		McpTools:          []string{},
		McpServers:        make(map[string]McpServerConfig),
		MessageBufferSize: 10,
		ErrorBufferSize:   1,
	}
}

// BuildCLIArgs builds command line arguments from options with validation
func (o *Options) BuildCLIArgs() ([]string, error) {
	if o == nil {
		return []string{}, nil
	}

	args := []string{}

	// Add prompt-related arguments
	if err := o.addPromptArgs(&args); err != nil {
		return nil, err
	}

	// Add tool-related arguments
	if err := o.addToolArgs(&args); err != nil {
		return nil, err
	}

	// Add configuration arguments
	if err := o.addConfigArgs(&args); err != nil {
		return nil, err
	}

	// Add permission-related arguments
	if err := o.addPermissionArgs(&args); err != nil {
		return nil, err
	}

	// Add session-related arguments
	if err := o.addSessionArgs(&args); err != nil {
		return nil, err
	}

	// Add MCP-related arguments
	if err := o.addMCPArgs(&args); err != nil {
		return nil, err
	}

	return args, nil
}

// addPromptArgs adds system prompt related arguments
func (o *Options) addPromptArgs(args *[]string) error {
	if o.SystemPrompt != "" {
		sanitized, err := validation.SanitizeString(o.SystemPrompt, validation.MaxStringLength)
		if err != nil {
			return fmt.Errorf("invalid system prompt: %w", err)
		}
		*args = append(*args, "--system-prompt", sanitized)
	}

	if o.AppendSystemPrompt != "" {
		sanitized, err := validation.SanitizeString(o.AppendSystemPrompt, validation.MaxStringLength)
		if err != nil {
			return fmt.Errorf("invalid append system prompt: %w", err)
		}
		*args = append(*args, "--append-system-prompt", sanitized)
	}

	return nil
}

// addToolArgs adds tool-related arguments
func (o *Options) addToolArgs(args *[]string) error {
	// Allowed tools
	if len(o.AllowedTools) > 0 {
		tools, err := o.validateToolList(o.AllowedTools, "allowed")
		if err != nil {
			return err
		}
		*args = append(*args, "--allowedTools", strings.Join(tools, ","))
	}

	// Disallowed tools
	if len(o.DisallowedTools) > 0 {
		tools, err := o.validateToolList(o.DisallowedTools, "disallowed")
		if err != nil {
			return err
		}
		*args = append(*args, "--disallowedTools", strings.Join(tools, ","))
	}

	return nil
}

// addPermissionArgs adds permission-related arguments
func (o *Options) addPermissionArgs(args *[]string) error {
	// Permission prompt tool
	if o.PermissionPromptToolName != "" {
		sanitized, err := validation.SanitizeCommandArg(o.PermissionPromptToolName)
		if err != nil {
			return fmt.Errorf("invalid permission prompt tool name: %w", err)
		}
		*args = append(*args, "--permission-prompt-tool", sanitized)
	}

	// Permission mode
	if o.PermissionMode != nil {
		mode := string(*o.PermissionMode)
		if mode != "default" && mode != "acceptEdits" && mode != "bypassPermissions" {
			return fmt.Errorf("invalid permission mode: %s", mode)
		}
		*args = append(*args, "--permission-mode", mode)
	}

	return nil
}

// addConfigArgs adds configuration arguments
func (o *Options) addConfigArgs(args *[]string) error {
	// Max turns
	if o.MaxTurns != nil {
		if *o.MaxTurns < 0 || *o.MaxTurns > 1000 {
			return fmt.Errorf("max turns must be between 0 and 1000")
		}
		*args = append(*args, "--max-turns", fmt.Sprintf("%d", *o.MaxTurns))
	}

	// Model
	if o.Model != "" {
		if err := validation.ValidateModel(o.Model); err != nil {
			return err
		}
		*args = append(*args, "--model", o.Model)
	}

	// Max thinking tokens
	if o.MaxThinkingTokens != 8000 {
		if o.MaxThinkingTokens < 0 || o.MaxThinkingTokens > 100000 {
			return fmt.Errorf("max thinking tokens must be between 0 and 100000")
		}
		*args = append(*args, "--max-thinking-tokens", fmt.Sprintf("%d", o.MaxThinkingTokens))
	}

	return nil
}

// addSessionArgs adds session-related arguments
func (o *Options) addSessionArgs(args *[]string) error {
	if o.ContinueConversation {
		*args = append(*args, "--continue")
	}

	if o.Resume != "" {
		sanitized, err := validation.SanitizeCommandArg(o.Resume)
		if err != nil {
			return fmt.Errorf("invalid resume ID: %w", err)
		}
		*args = append(*args, "--resume", sanitized)
	}

	return nil
}

// addMCPArgs adds MCP-related arguments
func (o *Options) addMCPArgs(args *[]string) error {
	// MCP tools
	if len(o.McpTools) > 0 {
		tools, err := o.validateToolList(o.McpTools, "MCP")
		if err != nil {
			return err
		}
		*args = append(*args, "--mcp-tools", strings.Join(tools, ","))
	}

	// MCP servers
	if len(o.McpServers) > 0 {
		mcpConfig := map[string]interface{}{
			"mcpServers": o.McpServers,
		}
		configJSON, err := json.Marshal(mcpConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal MCP config: %w", err)
		}
		// Validate JSON size
		if len(configJSON) > validation.MaxJSONSize {
			return fmt.Errorf("MCP config exceeds maximum size")
		}
		*args = append(*args, "--mcp-config", string(configJSON))
	}

	return nil
}

// validateToolList validates a list of tool names
func (o *Options) validateToolList(tools []string, toolType string) ([]string, error) {
	validatedTools := make([]string, 0, len(tools))
	for _, tool := range tools {
		sanitized, err := validation.SanitizeCommandArg(tool)
		if err != nil {
			return nil, fmt.Errorf("invalid %s tool name %q: %w", toolType, tool, err)
		}
		validatedTools = append(validatedTools, sanitized)
	}
	return validatedTools, nil
}

// GetCwd returns the working directory
func (o *Options) GetCwd() string {
	if o == nil {
		return ""
	}
	return o.Cwd
}

// GetMessageBufferSize returns the message buffer size with default
func (o *Options) GetMessageBufferSize() int {
	if o == nil || o.MessageBufferSize <= 0 {
		return 10
	}
	return o.MessageBufferSize
}

// GetErrorBufferSize returns the error buffer size with default
func (o *Options) GetErrorBufferSize() int {
	if o == nil || o.ErrorBufferSize <= 0 {
		return 1
	}
	return o.ErrorBufferSize
}

// GetQueryTimeout returns the query timeout duration
// Returns 0 if no timeout is set (meaning use context timeout)
func (o *Options) GetQueryTimeout() time.Duration {
	if o == nil || o.QueryTimeout <= 0 {
		return 0
	}
	return time.Duration(o.QueryTimeout) * time.Second
}

// Custom JSON marshaling/unmarshaling for ContentBlock to handle polymorphism

type contentBlockJSON struct {
	Type string `json:"type"`
	*TextBlock
	*ToolUseBlock
	*ToolResultBlock
}

func (cb *contentBlockJSON) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	typ, ok := raw["type"].(string)
	if !ok {
		return nil
	}

	switch typ {
	case "text":
		cb.Type = "text"
		cb.TextBlock = &TextBlock{}
		if text, ok := raw["text"].(string); ok {
			cb.TextBlock.Text = text
		}
	case "tool_use":
		cb.Type = "tool_use"
		cb.ToolUseBlock = &ToolUseBlock{}
		if id, ok := raw["id"].(string); ok {
			cb.ToolUseBlock.ID = id
		}
		if name, ok := raw["name"].(string); ok {
			cb.ToolUseBlock.Name = name
		}
		if input, ok := raw["input"].(map[string]interface{}); ok {
			cb.ToolUseBlock.Input = input
		}
	case "tool_result":
		cb.Type = "tool_result"
		cb.ToolResultBlock = &ToolResultBlock{}
		if toolUseID, ok := raw["tool_use_id"].(string); ok {
			cb.ToolResultBlock.ToolUseID = toolUseID
		}
		if content, ok := raw["content"]; ok {
			cb.ToolResultBlock.Content = content
		}
		if isError, ok := raw["is_error"].(bool); ok {
			cb.ToolResultBlock.IsError = &isError
		}
	}

	return nil
}

func (cb contentBlockJSON) MarshalJSON() ([]byte, error) {
	switch cb.Type {
	case "text":
		return json.Marshal(struct {
			Type string `json:"type"`
			*TextBlock
		}{
			Type:      "text",
			TextBlock: cb.TextBlock,
		})
	case "tool_use":
		return json.Marshal(struct {
			Type string `json:"type"`
			*ToolUseBlock
		}{
			Type:         "tool_use",
			ToolUseBlock: cb.ToolUseBlock,
		})
	case "tool_result":
		return json.Marshal(struct {
			Type string `json:"type"`
			*ToolResultBlock
		}{
			Type:            "tool_result",
			ToolResultBlock: cb.ToolResultBlock,
		})
	}
	return nil, nil
}

// MarshalJSON for AssistantMessage to handle ContentBlock polymorphism
func (am AssistantMessage) MarshalJSON() ([]byte, error) {
	temp := struct {
		Content []json.RawMessage `json:"content"`
	}{
		Content: make([]json.RawMessage, 0, len(am.Content)),
	}

	for _, block := range am.Content {
		var data []byte
		var err error

		switch b := block.(type) {
		case TextBlock:
			data, err = json.Marshal(struct {
				Type string `json:"type"`
				TextBlock
			}{
				Type:      "text",
				TextBlock: b,
			})
		case ToolUseBlock:
			data, err = json.Marshal(struct {
				Type string `json:"type"`
				ToolUseBlock
			}{
				Type:         "tool_use",
				ToolUseBlock: b,
			})
		case ToolResultBlock:
			data, err = json.Marshal(struct {
				Type string `json:"type"`
				ToolResultBlock
			}{
				Type:            "tool_result",
				ToolResultBlock: b,
			})
		default:
			continue
		}

		if err != nil {
			return nil, err
		}
		temp.Content = append(temp.Content, data)
	}

	return json.Marshal(temp)
}

// UnmarshalJSON for AssistantMessage to handle ContentBlock polymorphism
func (am *AssistantMessage) UnmarshalJSON(data []byte) error {
	var temp struct {
		Content []contentBlockJSON `json:"content"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	am.Content = make([]ContentBlock, 0, len(temp.Content))
	for _, cb := range temp.Content {
		switch cb.Type {
		case "text":
			am.Content = append(am.Content, *cb.TextBlock)
		case "tool_use":
			am.Content = append(am.Content, *cb.ToolUseBlock)
		case "tool_result":
			am.Content = append(am.Content, *cb.ToolResultBlock)
		}
	}

	return nil
}
