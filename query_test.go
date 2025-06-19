package claudecode

import (
	"context"
	"testing"
	"time"
)

func TestQuery(t *testing.T) {
	t.Run("Uses default options when nil", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// This test mainly ensures no panic when options is nil
		msgCh, errCh := Query(ctx, "test", nil)

		// Cancel immediately
		cancel()

		// Drain channels
		for {
			select {
			case _, ok := <-msgCh:
				if !ok {
					return
				}
			case <-errCh:
				return
			case <-time.After(100 * time.Millisecond):
				return
			}
		}
	})

	t.Run("Handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		msgCh, errCh := Query(ctx, "test", nil)

		// Cancel immediately
		cancel()

		// Both channels should close
		timeout := time.After(1 * time.Second)
		msgClosed := false
		errClosed := false

		for !msgClosed || !errClosed {
			select {
			case _, ok := <-msgCh:
				if !ok {
					msgClosed = true
				}
			case _, ok := <-errCh:
				if !ok {
					errClosed = true
				}
			case <-timeout:
				t.Fatal("Timeout waiting for channels to close after context cancellation")
			}
		}
	})

	t.Run("Passes options correctly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		opts := &Options{
			SystemPrompt: "Test prompt",
			MaxTurns:     intPtr(5),
			Model:        "test-model",
		}

		// Start query with options
		msgCh, errCh := Query(ctx, "test with options", opts)

		// Cancel immediately to clean up
		cancel()

		// Drain channels
		for {
			select {
			case _, ok := <-msgCh:
				if !ok {
					return
				}
			case _, ok := <-errCh:
				if !ok {
					return
				}
			case <-time.After(100 * time.Millisecond):
				return
			}
		}
	})

	t.Run("Returns error channel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, errCh := Query(ctx, "test", nil)

		// Error channel should be created
		if errCh == nil {
			t.Fatal("Error channel should not be nil")
		}

		cancel()
	})

	t.Run("Returns message channel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		msgCh, _ := Query(ctx, "test", nil)

		// Message channel should be created
		if msgCh == nil {
			t.Fatal("Message channel should not be nil")
		}

		cancel()
	})
}

// TestConvertMessage tests the message conversion logic
func TestConvertMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType string
		wantErr  bool
	}{
		{
			name: "assistant message",
			input: map[string]interface{}{
				"_type": "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"_blockType": "text",
						"text":       "Hello",
					},
				},
			},
			wantType: "AssistantMessage",
		},
		{
			name: "user message",
			input: map[string]interface{}{
				"_type":   "user",
				"content": "Hi there",
			},
			wantType: "UserMessage",
		},
		{
			name: "system message",
			input: map[string]interface{}{
				"_type":   "system",
				"subtype": "info",
				"data":    map[string]interface{}{"key": "value"},
			},
			wantType: "SystemMessage",
		},
		{
			name: "result message",
			input: map[string]interface{}{
				"_type":          "result",
				"total_cost_usd": 0.01,
				"duration_ms":    1000,
			},
			wantType: "ResultMessage",
		},
		{
			name:     "non-map message",
			input:    "not a map",
			wantType: "unknown",
		},
		{
			name: "map without _type",
			input: map[string]interface{}{
				"content": "test",
			},
			wantType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMessage(tt.input)

			switch msg := result.(type) {
			case AssistantMessage:
				if tt.wantType != "AssistantMessage" {
					t.Errorf("got AssistantMessage, want %s", tt.wantType)
				}
			case UserMessage:
				if tt.wantType != "UserMessage" {
					t.Errorf("got UserMessage, want %s", tt.wantType)
				}
			case SystemMessage:
				if tt.wantType != "SystemMessage" {
					t.Errorf("got SystemMessage, want %s", tt.wantType)
				}
			case ResultMessage:
				if tt.wantType != "ResultMessage" {
					t.Errorf("got ResultMessage, want %s", tt.wantType)
				}
			default:
				if tt.wantType != "unknown" {
					t.Errorf("got unknown type %T, want %s", msg, tt.wantType)
				}
			}
		})
	}
}

// TestConvertContentBlock tests content block conversion
func TestConvertContentBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType string
		wantNil  bool
	}{
		{
			name: "text block",
			input: map[string]interface{}{
				"_blockType": "text",
				"text":       "Hello, world!",
			},
			wantType: "TextBlock",
		},
		{
			name: "tool use block",
			input: map[string]interface{}{
				"_blockType": "tool_use",
				"id":         "tool_123",
				"name":       "Read",
				"input":      map[string]interface{}{"path": "/test.txt"},
			},
			wantType: "ToolUseBlock",
		},
		{
			name: "tool result block",
			input: map[string]interface{}{
				"_blockType":  "tool_result",
				"tool_use_id": "tool_123",
				"content":     "File contents",
				"is_error":    false,
			},
			wantType: "ToolResultBlock",
		},
		{
			name:    "non-map input",
			input:   "not a map",
			wantNil: true,
		},
		{
			name: "unknown block type",
			input: map[string]interface{}{
				"_blockType": "unknown",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertContentBlock(tt.input)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil but got %v", result)
				}
				return
			}

			if result == nil {
				t.Error("expected non-nil result")
				return
			}

			switch block := result.(type) {
			case TextBlock:
				if tt.wantType != "TextBlock" {
					t.Errorf("got TextBlock, want %s", tt.wantType)
				}
			case ToolUseBlock:
				if tt.wantType != "ToolUseBlock" {
					t.Errorf("got ToolUseBlock, want %s", tt.wantType)
				}
			case ToolResultBlock:
				if tt.wantType != "ToolResultBlock" {
					t.Errorf("got ToolResultBlock, want %s", tt.wantType)
				}
			default:
				t.Errorf("unexpected block type: %T", block)
			}
		})
	}
}

// TestMessageConversionIntegration tests the full message conversion flow
func TestMessageConversionIntegration(t *testing.T) {
	// Test converting a complex assistant message
	input := map[string]interface{}{
		"_type": "assistant",
		"content": []interface{}{
			map[string]interface{}{
				"_blockType": "text",
				"text":       "I'll help you with that.",
			},
			map[string]interface{}{
				"_blockType": "tool_use",
				"id":         "tool_456",
				"name":       "Write",
				"input": map[string]interface{}{
					"path":    "/output.txt",
					"content": "Hello, world!",
				},
			},
		},
	}

	msg := convertMessage(input)
	assistantMsg, ok := msg.(AssistantMessage)
	if !ok {
		t.Fatalf("expected AssistantMessage, got %T", msg)
	}

	if len(assistantMsg.Content) != 2 {
		t.Errorf("expected 2 content blocks, got %d", len(assistantMsg.Content))
	}

	// Check first block
	if _, ok := assistantMsg.Content[0].(TextBlock); !ok {
		t.Errorf("expected first block to be TextBlock, got %T", assistantMsg.Content[0])
	}

	// Check second block
	if toolBlock, ok := assistantMsg.Content[1].(ToolUseBlock); ok {
		if toolBlock.Name != "Write" {
			t.Errorf("expected tool name 'Write', got %s", toolBlock.Name)
		}
	} else {
		t.Errorf("expected second block to be ToolUseBlock, got %T", assistantMsg.Content[1])
	}
}

// TestErrorHandling tests error propagation
func TestErrorHandling(t *testing.T) {
	// This test would require mocking the internal client
	// Since Query creates its own client, we can only test basic behavior
	t.Run("Error channel receives errors", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Use invalid options that might cause an error
		opts := &Options{
			SystemPrompt: "",
			MaxTurns:     intPtr(-1), // Invalid value
		}

		_, errCh := Query(ctx, "test", opts)

		// Cancel after a short time
		time.AfterFunc(50*time.Millisecond, cancel)

		// Wait for potential error or timeout
		select {
		case err := <-errCh:
			// If we get an error, that's fine for this test
			if err != nil {
				t.Logf("Received error (expected): %v", err)
			}
		case <-time.After(100 * time.Millisecond):
			// Timeout is also acceptable
		}
	})
}

// TestQueryOptions tests that options are properly handled
func TestQueryOptions(t *testing.T) {
	tests := []struct {
		name string
		opts *Options
	}{
		{
			name: "nil options",
			opts: nil,
		},
		{
			name: "empty options",
			opts: &Options{},
		},
		{
			name: "full options",
			opts: &Options{
				SystemPrompt:       "You are helpful",
				AppendSystemPrompt: "Be concise",
				Model:              "claude-3",
				MaxTurns:           intPtr(10),
				PermissionMode:     (*PermissionMode)(stringPtr("default")),
				AllowedTools:       []string{"Read", "Write"},
				DisallowedTools:    []string{"Bash"},
				Cwd:                "/tmp",
			},
		},
		{
			name: "options with MCP servers",
			opts: &Options{
				McpServers: map[string]McpServerConfig{
					"test-server": {
						Transport: []string{"stdio", "test-mcp-server", "--port", "8080"},
						Env:       map[string]interface{}{"DEBUG": "true"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Query should not panic with any options
			msgCh, errCh := Query(ctx, "test prompt", tt.opts)

			// Cancel immediately
			cancel()

			// Drain channels
			done := false
			for !done {
				select {
				case _, ok := <-msgCh:
					if !ok {
						done = true
					}
				case _, ok := <-errCh:
					if !ok && msgCh == nil {
						done = true
					}
				case <-time.After(100 * time.Millisecond):
					done = true
				}
			}
		})
	}
}

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
