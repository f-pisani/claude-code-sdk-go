package claudecode

import (
	"encoding/json"
	"testing"
)

func TestMessageTypes(t *testing.T) {
	t.Run("UserMessage creation", func(t *testing.T) {
		msg := UserMessage{Content: "Hello, Claude!"}
		if msg.Content != "Hello, Claude!" {
			t.Errorf("Expected content 'Hello, Claude!', got %s", msg.Content)
		}
	})

	t.Run("AssistantMessage with text", func(t *testing.T) {
		textBlock := TextBlock{Text: "Hello, human!"}
		msg := AssistantMessage{Content: []ContentBlock{textBlock}}
		if len(msg.Content) != 1 {
			t.Errorf("Expected 1 content block, got %d", len(msg.Content))
		}
		if tb, ok := msg.Content[0].(TextBlock); ok {
			if tb.Text != "Hello, human!" {
				t.Errorf("Expected text 'Hello, human!', got %s", tb.Text)
			}
		} else {
			t.Error("Expected TextBlock type")
		}
	})

	t.Run("ToolUseBlock", func(t *testing.T) {
		block := ToolUseBlock{
			ID:    "tool-123",
			Name:  "Read",
			Input: map[string]interface{}{"file_path": "/test.txt"},
		}
		if block.ID != "tool-123" {
			t.Errorf("Expected ID 'tool-123', got %s", block.ID)
		}
		if block.Name != "Read" {
			t.Errorf("Expected name 'Read', got %s", block.Name)
		}
		if block.Input["file_path"] != "/test.txt" {
			t.Errorf("Expected file_path '/test.txt', got %v", block.Input["file_path"])
		}
	})

	t.Run("ToolResultBlock", func(t *testing.T) {
		isError := false
		block := ToolResultBlock{
			ToolUseID: "tool-123",
			Content:   "File contents here",
			IsError:   &isError,
		}
		if block.ToolUseID != "tool-123" {
			t.Errorf("Expected tool_use_id 'tool-123', got %s", block.ToolUseID)
		}
		if block.Content != "File contents here" {
			t.Errorf("Expected content 'File contents here', got %v", block.Content)
		}
		if *block.IsError != false {
			t.Error("Expected is_error to be false")
		}
	})

	t.Run("ResultMessage", func(t *testing.T) {
		cost := 0.01
		msg := ResultMessage{
			Subtype:       "success",
			DurationMs:    1500,
			DurationAPIMs: 1200,
			IsError:       false,
			NumTurns:      1,
			SessionID:     "session-123",
			TotalCostUSD:  &cost,
		}
		if msg.Subtype != "success" {
			t.Errorf("Expected subtype 'success', got %s", msg.Subtype)
		}
		if *msg.TotalCostUSD != 0.01 {
			t.Errorf("Expected cost 0.01, got %f", *msg.TotalCostUSD)
		}
		if msg.SessionID != "session-123" {
			t.Errorf("Expected session_id 'session-123', got %s", msg.SessionID)
		}
	})
}

func TestOptions(t *testing.T) {
	t.Run("Default options", func(t *testing.T) {
		options := NewOptions()
		if options.MaxThinkingTokens != 8000 {
			t.Errorf("Expected MaxThinkingTokens 8000, got %d", options.MaxThinkingTokens)
		}
		if len(options.AllowedTools) != 0 {
			t.Errorf("Expected empty AllowedTools, got %v", options.AllowedTools)
		}
		if options.SystemPrompt != "" {
			t.Errorf("Expected empty SystemPrompt, got %s", options.SystemPrompt)
		}
		if options.PermissionMode != nil {
			t.Errorf("Expected nil PermissionMode, got %v", options.PermissionMode)
		}
		if options.ContinueConversation {
			t.Error("Expected ContinueConversation to be false")
		}
	})

	t.Run("Options with tools", func(t *testing.T) {
		options := NewOptions()
		options.AllowedTools = []string{"Read", "Write", "Edit"}
		options.DisallowedTools = []string{"Bash"}

		if len(options.AllowedTools) != 3 {
			t.Errorf("Expected 3 allowed tools, got %d", len(options.AllowedTools))
		}
		if len(options.DisallowedTools) != 1 {
			t.Errorf("Expected 1 disallowed tool, got %d", len(options.DisallowedTools))
		}
	})

	t.Run("Options with permission mode", func(t *testing.T) {
		options := NewOptions()
		mode := PermissionModeBypassPermissions
		options.PermissionMode = &mode

		if *options.PermissionMode != PermissionModeBypassPermissions {
			t.Errorf("Expected PermissionModeBypassPermissions, got %v", *options.PermissionMode)
		}
	})
}

func TestContentBlockJSONMarshaling(t *testing.T) {
	t.Run("AssistantMessage JSON unmarshaling", func(t *testing.T) {
		jsonData := `{
			"content": [
				{"type": "text", "text": "Hello"},
				{"type": "tool_use", "id": "123", "name": "Read", "input": {"file": "test.txt"}},
				{"type": "tool_result", "tool_use_id": "123", "content": "File contents", "is_error": false}
			]
		}`

		var msg AssistantMessage
		err := json.Unmarshal([]byte(jsonData), &msg)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(msg.Content) != 3 {
			t.Fatalf("Expected 3 content blocks, got %d", len(msg.Content))
		}

		// Check text block
		if tb, ok := msg.Content[0].(TextBlock); ok {
			if tb.Text != "Hello" {
				t.Errorf("Expected text 'Hello', got %s", tb.Text)
			}
		} else {
			t.Error("Expected first block to be TextBlock")
		}

		// Check tool use block
		if tub, ok := msg.Content[1].(ToolUseBlock); ok {
			if tub.ID != "123" {
				t.Errorf("Expected ID '123', got %s", tub.ID)
			}
			if tub.Name != "Read" {
				t.Errorf("Expected name 'Read', got %s", tub.Name)
			}
		} else {
			t.Error("Expected second block to be ToolUseBlock")
		}

		// Check tool result block
		if trb, ok := msg.Content[2].(ToolResultBlock); ok {
			if trb.ToolUseID != "123" {
				t.Errorf("Expected tool_use_id '123', got %s", trb.ToolUseID)
			}
			if trb.Content != "File contents" {
				t.Errorf("Expected content 'File contents', got %v", trb.Content)
			}
		} else {
			t.Error("Expected third block to be ToolResultBlock")
		}
	})
}

// TestJSONMarshaling tests JSON marshaling and unmarshaling for all types
func TestJSONMarshaling(t *testing.T) {
	t.Run("UserMessage marshaling", func(t *testing.T) {
		msg := UserMessage{Content: "Test message"}
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal UserMessage: %v", err)
		}

		var decoded UserMessage
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal UserMessage: %v", err)
		}

		if decoded.Content != msg.Content {
			t.Errorf("Content mismatch: got %q, want %q", decoded.Content, msg.Content)
		}
	})

	t.Run("SystemMessage marshaling", func(t *testing.T) {
		msg := SystemMessage{
			Subtype: "info",
			Data: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal SystemMessage: %v", err)
		}

		var decoded SystemMessage
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal SystemMessage: %v", err)
		}

		if decoded.Subtype != msg.Subtype {
			t.Errorf("Subtype mismatch: got %q, want %q", decoded.Subtype, msg.Subtype)
		}

		if decoded.Data["key1"] != "value1" {
			t.Errorf("Data[key1] mismatch: got %v, want %v", decoded.Data["key1"], "value1")
		}

		// JSON numbers are unmarshaled as float64
		if decoded.Data["key2"] != float64(42) {
			t.Errorf("Data[key2] mismatch: got %v, want %v", decoded.Data["key2"], 42)
		}
	})

	t.Run("ResultMessage marshaling", func(t *testing.T) {
		cost := 0.025
		result := "Success"
		msg := ResultMessage{
			Subtype:       "completion",
			DurationMs:    2000,
			DurationAPIMs: 1800,
			IsError:       false,
			NumTurns:      5,
			SessionID:     "test-session",
			TotalCostUSD:  &cost,
			Usage: map[string]interface{}{
				"input_tokens":  500,
				"output_tokens": 250,
			},
			Result: &result,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal ResultMessage: %v", err)
		}

		var decoded ResultMessage
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ResultMessage: %v", err)
		}

		if decoded.Subtype != msg.Subtype {
			t.Errorf("Subtype mismatch: got %q, want %q", decoded.Subtype, msg.Subtype)
		}

		if decoded.DurationMs != msg.DurationMs {
			t.Errorf("DurationMs mismatch: got %d, want %d", decoded.DurationMs, msg.DurationMs)
		}

		if *decoded.TotalCostUSD != *msg.TotalCostUSD {
			t.Errorf("TotalCostUSD mismatch: got %f, want %f", *decoded.TotalCostUSD, *msg.TotalCostUSD)
		}

		if *decoded.Result != *msg.Result {
			t.Errorf("Result mismatch: got %q, want %q", *decoded.Result, *msg.Result)
		}
	})

	t.Run("TextBlock marshaling", func(t *testing.T) {
		block := TextBlock{Text: "Hello, world!"}
		data, err := json.Marshal(block)
		if err != nil {
			t.Fatalf("Failed to marshal TextBlock: %v", err)
		}

		var decoded TextBlock
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal TextBlock: %v", err)
		}

		if decoded.Text != block.Text {
			t.Errorf("Text mismatch: got %q, want %q", decoded.Text, block.Text)
		}
	})

	t.Run("ToolUseBlock marshaling", func(t *testing.T) {
		block := ToolUseBlock{
			ID:   "tool-456",
			Name: "Write",
			Input: map[string]interface{}{
				"path":    "/output.txt",
				"content": "Test content",
			},
		}

		data, err := json.Marshal(block)
		if err != nil {
			t.Fatalf("Failed to marshal ToolUseBlock: %v", err)
		}

		var decoded ToolUseBlock
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ToolUseBlock: %v", err)
		}

		if decoded.ID != block.ID {
			t.Errorf("ID mismatch: got %q, want %q", decoded.ID, block.ID)
		}

		if decoded.Name != block.Name {
			t.Errorf("Name mismatch: got %q, want %q", decoded.Name, block.Name)
		}

		if decoded.Input["path"] != block.Input["path"] {
			t.Errorf("Input[path] mismatch: got %v, want %v", decoded.Input["path"], block.Input["path"])
		}
	})

	t.Run("ToolResultBlock marshaling", func(t *testing.T) {
		isError := true
		block := ToolResultBlock{
			ToolUseID: "tool-456",
			Content:   "Error: File not found",
			IsError:   &isError,
		}

		data, err := json.Marshal(block)
		if err != nil {
			t.Fatalf("Failed to marshal ToolResultBlock: %v", err)
		}

		var decoded ToolResultBlock
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ToolResultBlock: %v", err)
		}

		if decoded.ToolUseID != block.ToolUseID {
			t.Errorf("ToolUseID mismatch: got %q, want %q", decoded.ToolUseID, block.ToolUseID)
		}

		if *decoded.IsError != *block.IsError {
			t.Errorf("IsError mismatch: got %v, want %v", *decoded.IsError, *block.IsError)
		}
	})

	t.Run("Options marshaling", func(t *testing.T) {
		maxTurns := 10
		permMode := PermissionModeAcceptEdits
		opts := Options{
			AllowedTools:      []string{"Read", "Write"},
			MaxThinkingTokens: 5000,
			SystemPrompt:      "You are helpful",
			McpServers: map[string]McpServerConfig{
				"test": {
					Transport: []string{"stdio", "test-server"},
					Env: map[string]interface{}{
						"PORT": 8080,
					},
				},
			},
			PermissionMode: &permMode,
			MaxTurns:       &maxTurns,
			Model:          "claude-3",
			Cwd:            "/workspace",
		}

		data, err := json.Marshal(opts)
		if err != nil {
			t.Fatalf("Failed to marshal Options: %v", err)
		}

		var decoded Options
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal Options: %v", err)
		}

		if decoded.SystemPrompt != opts.SystemPrompt {
			t.Errorf("SystemPrompt mismatch: got %q, want %q", decoded.SystemPrompt, opts.SystemPrompt)
		}

		if decoded.MaxThinkingTokens != opts.MaxThinkingTokens {
			t.Errorf("MaxThinkingTokens mismatch: got %d, want %d", decoded.MaxThinkingTokens, opts.MaxThinkingTokens)
		}

		if *decoded.PermissionMode != *opts.PermissionMode {
			t.Errorf("PermissionMode mismatch: got %v, want %v", *decoded.PermissionMode, *opts.PermissionMode)
		}

		if len(decoded.AllowedTools) != len(opts.AllowedTools) {
			t.Errorf("AllowedTools length mismatch: got %d, want %d", len(decoded.AllowedTools), len(opts.AllowedTools))
		}
	})

	t.Run("McpServerConfig marshaling", func(t *testing.T) {
		config := McpServerConfig{
			Transport: []string{"stdio", "mcp-server", "--debug"},
			Env: map[string]interface{}{
				"DEBUG":   "true",
				"PORT":    8080,
				"TIMEOUT": 30.5,
			},
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal McpServerConfig: %v", err)
		}

		var decoded McpServerConfig
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal McpServerConfig: %v", err)
		}

		if len(decoded.Transport) != len(config.Transport) {
			t.Errorf("Transport length mismatch: got %d, want %d", len(decoded.Transport), len(config.Transport))
		}

		for i, v := range config.Transport {
			if decoded.Transport[i] != v {
				t.Errorf("Transport[%d] mismatch: got %q, want %q", i, decoded.Transport[i], v)
			}
		}

		if decoded.Env["DEBUG"] != config.Env["DEBUG"] {
			t.Errorf("Env[DEBUG] mismatch: got %v, want %v", decoded.Env["DEBUG"], config.Env["DEBUG"])
		}

		// JSON numbers are unmarshaled as float64
		if decoded.Env["PORT"] != float64(8080) {
			t.Errorf("Env[PORT] mismatch: got %v, want %v", decoded.Env["PORT"], 8080)
		}
	})

	t.Run("Complex AssistantMessage marshaling", func(t *testing.T) {
		// This tests the custom MarshalJSON method for AssistantMessage
		msg := AssistantMessage{
			Content: []ContentBlock{
				TextBlock{Text: "I'll help you with that."},
				ToolUseBlock{
					ID:   "tool-789",
					Name: "Edit",
					Input: map[string]interface{}{
						"file":     "main.go",
						"line":     42,
						"new_text": "fmt.Println(\"Hello\")",
					},
				},
				ToolResultBlock{
					ToolUseID: "tool-789",
					Content:   "Edit completed successfully",
				},
			},
		}

		// First, let's manually marshal to see the expected structure
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal AssistantMessage: %v", err)
		}

		// Now unmarshal it back
		var decoded AssistantMessage
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal AssistantMessage: %v", err)
		}

		if len(decoded.Content) != 3 {
			t.Fatalf("Content length mismatch: got %d, want %d", len(decoded.Content), 3)
		}

		// Verify each block
		if tb, ok := decoded.Content[0].(TextBlock); ok {
			if tb.Text != "I'll help you with that." {
				t.Errorf("TextBlock text mismatch: got %q, want %q", tb.Text, "I'll help you with that.")
			}
		} else {
			t.Errorf("Content[0] is not TextBlock: %T", decoded.Content[0])
		}

		if tub, ok := decoded.Content[1].(ToolUseBlock); ok {
			if tub.Name != "Edit" {
				t.Errorf("ToolUseBlock name mismatch: got %q, want %q", tub.Name, "Edit")
			}
			if tub.Input["line"] != float64(42) { // JSON numbers unmarshal as float64
				t.Errorf("ToolUseBlock input[line] mismatch: got %v, want %v", tub.Input["line"], 42)
			}
		} else {
			t.Errorf("Content[1] is not ToolUseBlock: %T", decoded.Content[1])
		}

		if trb, ok := decoded.Content[2].(ToolResultBlock); ok {
			if trb.Content != "Edit completed successfully" {
				t.Errorf("ToolResultBlock content mismatch: got %v, want %v", trb.Content, "Edit completed successfully")
			}
		} else {
			t.Errorf("Content[2] is not ToolResultBlock: %T", decoded.Content[2])
		}
	})

	t.Run("Edge cases", func(t *testing.T) {
		// Test nil pointers
		t.Run("ResultMessage with nil fields", func(t *testing.T) {
			msg := ResultMessage{
				Subtype:    "partial",
				DurationMs: 100,
				IsError:    true,
				// Leave TotalCostUSD, Usage, and Result as nil
			}

			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("Failed to marshal ResultMessage with nil fields: %v", err)
			}

			var decoded ResultMessage
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal ResultMessage with nil fields: %v", err)
			}

			if decoded.TotalCostUSD != nil {
				t.Errorf("Expected TotalCostUSD to be nil, got %v", *decoded.TotalCostUSD)
			}

			if decoded.Result != nil {
				t.Errorf("Expected Result to be nil, got %v", *decoded.Result)
			}
		})

		t.Run("Empty AssistantMessage", func(t *testing.T) {
			msg := AssistantMessage{
				Content: []ContentBlock{},
			}

			data, err := json.Marshal(msg)
			if err != nil {
				t.Fatalf("Failed to marshal empty AssistantMessage: %v", err)
			}

			var decoded AssistantMessage
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal empty AssistantMessage: %v", err)
			}

			if len(decoded.Content) != 0 {
				t.Errorf("Expected empty content, got %d blocks", len(decoded.Content))
			}
		})

		t.Run("ToolResultBlock with complex content", func(t *testing.T) {
			// ToolResultBlock.Content can be string or []map[string]interface{}
			block := ToolResultBlock{
				ToolUseID: "complex-tool",
				Content: []map[string]interface{}{
					{"type": "text", "text": "Line 1"},
					{"type": "text", "text": "Line 2"},
				},
			}

			data, err := json.Marshal(block)
			if err != nil {
				t.Fatalf("Failed to marshal ToolResultBlock with complex content: %v", err)
			}

			var decoded ToolResultBlock
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("Failed to unmarshal ToolResultBlock with complex content: %v", err)
			}

			if decoded.ToolUseID != block.ToolUseID {
				t.Errorf("ToolUseID mismatch: got %q, want %q", decoded.ToolUseID, block.ToolUseID)
			}

			// Content should be unmarshaled as []interface{}
			if content, ok := decoded.Content.([]interface{}); ok {
				if len(content) != 2 {
					t.Errorf("Content length mismatch: got %d, want 2", len(content))
				}
			} else {
				t.Errorf("Content is not []interface{}: %T", decoded.Content)
			}
		})
	})
}

// TestBuildCLIArgs tests the Options.BuildCLIArgs method
func TestBuildCLIArgs(t *testing.T) {
	tests := []struct {
		name     string
		options  *Options
		expected []string
	}{
		{
			name:     "nil options",
			options:  nil,
			expected: []string{},
		},
		{
			name:     "empty options",
			options:  &Options{MaxThinkingTokens: 8000},
			expected: []string{},
		},
		{
			name: "with system prompt",
			options: &Options{
				SystemPrompt:      "You are helpful",
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--system-prompt", "You are helpful"},
		},
		{
			name: "with multiple options",
			options: &Options{
				SystemPrompt:       "You are helpful",
				AppendSystemPrompt: "Be concise",
				MaxTurns:           intPtrHelper(3),
				Model:              "claude-3",
				MaxThinkingTokens:  8000,
			},
			expected: []string{
				"--system-prompt", "You are helpful",
				"--append-system-prompt", "Be concise",
				"--max-turns", "3",
				"--model", "claude-3",
			},
		},
		{
			name: "with tools",
			options: &Options{
				AllowedTools:      []string{"Read", "Write", "Edit"},
				DisallowedTools:   []string{"Bash"},
				MaxThinkingTokens: 8000,
			},
			expected: []string{
				"--allowedTools", "Read,Write,Edit",
				"--disallowedTools", "Bash",
			},
		},
		{
			name: "with permission mode",
			options: &Options{
				PermissionMode:           (*PermissionMode)(stringPtrHelper("acceptEdits")),
				PermissionPromptToolName: "custom-prompt",
				MaxThinkingTokens:        8000,
			},
			expected: []string{
				"--permission-prompt-tool", "custom-prompt",
				"--permission-mode", "acceptEdits",
			},
		},
		{
			name: "with continuation options",
			options: &Options{
				ContinueConversation: true,
				Resume:               "session-123",
				MaxThinkingTokens:    8000,
			},
			expected: []string{
				"--continue",
				"--resume", "session-123",
			},
		},
		{
			name: "with max thinking tokens",
			options: &Options{
				MaxThinkingTokens: 5000,
			},
			expected: []string{
				"--max-thinking-tokens", "5000",
			},
		},
		{
			name: "with default max thinking tokens",
			options: &Options{
				MaxThinkingTokens: 8000, // Default value
			},
			expected: []string{}, // Should not be included
		},
		{
			name: "with MCP servers",
			options: &Options{
				McpServers: map[string]McpServerConfig{
					"test-server": {
						Transport: []string{"stdio", "test-mcp-server"},
						Env: map[string]interface{}{
							"PORT": 8080,
						},
					},
				},
				MaxThinkingTokens: 8000,
			},
			expected: []string{
				"--mcp-config", `{"mcpServers":{"test-server":{"transport":["stdio","test-mcp-server"],"env":{"PORT":8080}}}}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := tt.options.BuildCLIArgs()
			if err != nil {
				t.Fatalf("BuildCLIArgs() returned error: %v", err)
			}

			if len(args) != len(tt.expected) {
				t.Errorf("got %d args, expected %d. Got: %v", len(args), len(tt.expected), args)
				return
			}

			for i := 0; i < len(args); i++ {
				if args[i] != tt.expected[i] {
					t.Errorf("arg at position %d: got %q, want %q", i, args[i], tt.expected[i])
				}
			}
		})
	}
}

// Helper functions for creating pointers (renamed to avoid conflicts)
func intPtrHelper(i int) *int {
	return &i
}

func stringPtrHelper(s string) *string {
	return &s
}

// TestPermissionModes tests the permission mode constants
func TestPermissionModes(t *testing.T) {
	modes := []PermissionMode{
		PermissionModeDefault,
		PermissionModeAcceptEdits,
		PermissionModeBypassPermissions,
	}

	expectedValues := []string{
		"default",
		"acceptEdits",
		"bypassPermissions",
	}

	for i, mode := range modes {
		if string(mode) != expectedValues[i] {
			t.Errorf("PermissionMode value mismatch: got %q, want %q", mode, expectedValues[i])
		}
	}

	// Test marshaling
	for _, mode := range modes {
		data, err := json.Marshal(mode)
		if err != nil {
			t.Fatalf("Failed to marshal PermissionMode %q: %v", mode, err)
		}

		var decoded PermissionMode
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal PermissionMode: %v", err)
		}

		if decoded != mode {
			t.Errorf("PermissionMode mismatch after marshal/unmarshal: got %q, want %q", decoded, mode)
		}
	}
}
