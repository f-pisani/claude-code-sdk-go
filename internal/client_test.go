package internal

import (
	"testing"
)

// TestClientProcessQuery tests the ProcessQuery method
func TestClientProcessQuery(t *testing.T) {
	// This test would require mocking the transport layer
	// Since the client creates its own transport internally,
	// we'll focus on testing the parsing methods instead
	t.Skip("Skipping ProcessQuery test - requires transport mocking")
}

// TestParseMessage tests message parsing
func TestParseMessage(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		input    map[string]interface{}
		wantType string
		wantNil  bool
	}{
		{
			name: "user message",
			input: map[string]interface{}{
				"type": "user",
				"message": map[string]interface{}{
					"content": "Hello",
				},
			},
			wantType: "user",
		},
		{
			name: "assistant message",
			input: map[string]interface{}{
				"type": "assistant",
				"message": map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Hello!",
						},
					},
				},
			},
			wantType: "assistant",
		},
		{
			name: "system message",
			input: map[string]interface{}{
				"type":    "system",
				"subtype": "info",
			},
			wantType: "system",
		},
		{
			name: "result message",
			input: map[string]interface{}{
				"type":           "result",
				"subtype":        "completion",
				"duration_ms":    1000.0,
				"duration_api_ms": 800.0,
				"is_error":       false,
				"num_turns":      1.0,
				"session_id":     "test-session",
				"total_cost_usd": 0.01,
				"usage": map[string]interface{}{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
			wantType: "result",
		},
		{
			name: "unknown type",
			input: map[string]interface{}{
				"type": "unknown",
			},
			wantNil: true,
		},
		{
			name: "missing type",
			input: map[string]interface{}{
				"content": "test",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.parseMessage(tt.input)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Error("expected non-nil result")
				return
			}

			// Check the type
			if msg, ok := result.(map[string]interface{}); ok {
				if msgType, ok := msg["_type"].(string); ok {
					if msgType != tt.wantType {
						t.Errorf("got type %q, want %q", msgType, tt.wantType)
					}
				} else {
					t.Error("missing _type field")
				}
			} else {
				t.Errorf("result is not a map: %T", result)
			}
		})
	}
}

// TestParseContentBlock tests content block parsing
func TestParseContentBlock(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantBlock string
		wantNil   bool
	}{
		{
			name: "text block",
			input: map[string]interface{}{
				"type": "text",
				"text": "Hello, world!",
			},
			wantBlock: "text",
		},
		{
			name: "tool use block",
			input: map[string]interface{}{
				"type": "tool_use",
				"id":   "tool_123",
				"name": "Read",
				"input": map[string]interface{}{
					"path": "/test.txt",
				},
			},
			wantBlock: "tool_use",
		},
		{
			name: "tool result block",
			input: map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": "tool_123",
				"content":     "File contents",
				"is_error":    false,
			},
			wantBlock: "tool_result",
		},
		{
			name: "unknown block type",
			input: map[string]interface{}{
				"type": "unknown",
			},
			wantNil: true,
		},
		{
			name: "missing type",
			input: map[string]interface{}{
				"text": "test",
			},
			wantNil: true,
		},
		{
			name: "text block missing text",
			input: map[string]interface{}{
				"type": "text",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.parseContentBlock(tt.input)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Error("expected non-nil result")
				return
			}

			// Check the block type
			if block, ok := result.(map[string]interface{}); ok {
				if blockType, ok := block["_blockType"].(string); ok {
					if blockType != tt.wantBlock {
						t.Errorf("got block type %q, want %q", blockType, tt.wantBlock)
					}
				} else {
					t.Error("missing _blockType field")
				}
			} else {
				t.Errorf("result is not a map: %T", result)
			}
		})
	}
}

// TestParseAssistantMessage tests parsing of assistant messages with multiple content blocks
func TestParseAssistantMessage(t *testing.T) {
	client := NewClient()

	input := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "I'll help you read that file.",
				},
				map[string]interface{}{
					"type": "tool_use",
					"id":   "tool_456",
					"name": "Read",
					"input": map[string]interface{}{
						"path": "/example.txt",
					},
				},
			},
		},
	}

	result := client.parseMessage(input)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	msg, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %T", result)
	}

	// Check type
	if msgType := msg["_type"]; msgType != "assistant" {
		t.Errorf("got type %v, want assistant", msgType)
	}

	// Check content blocks
	content, ok := msg["content"].([]interface{})
	if !ok {
		t.Fatal("content is not a slice")
	}

	if len(content) != 2 {
		t.Errorf("got %d content blocks, want 2", len(content))
	}

	// Check first block (text)
	if block1, ok := content[0].(map[string]interface{}); ok {
		if block1["_blockType"] != "text" {
			t.Errorf("first block type: got %v, want text", block1["_blockType"])
		}
		if block1["text"] != "I'll help you read that file." {
			t.Errorf("text content mismatch")
		}
	}

	// Check second block (tool_use)
	if len(content) > 1 {
		if block2, ok := content[1].(map[string]interface{}); ok {
			if block2["_blockType"] != "tool_use" {
				t.Errorf("second block type: got %v, want tool_use", block2["_blockType"])
			}
			if block2["name"] != "Read" {
				t.Errorf("tool name: got %v, want Read", block2["name"])
			}
		}
	}
}

// TestParseResultMessage tests parsing of result messages
func TestParseResultMessage(t *testing.T) {
	client := NewClient()

	input := map[string]interface{}{
		"type":           "result",
		"subtype":        "completion",
		"duration_ms":    1500.0,
		"duration_api_ms": 1200.0,
		"is_error":       false,
		"num_turns":      3.0,
		"session_id":     "session-123",
		"total_cost_usd": 0.025,
		"usage": map[string]interface{}{
			"input_tokens":  250,
			"output_tokens": 150,
		},
		"result": "Task completed successfully",
	}

	result := client.parseMessage(input)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	msg, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not a map: %T", result)
	}

	// Check all fields
	checks := map[string]interface{}{
		"_type":          "result",
		"subtype":        "completion",
		"duration_ms":    1500,
		"duration_api_ms": 1200,
		"is_error":       false,
		"num_turns":      3,
		"session_id":     "session-123",
		"total_cost_usd": 0.025,
		"result":         "Task completed successfully",
	}

	for key, expected := range checks {
		if actual := msg[key]; actual != expected {
			t.Errorf("%s: got %v (%T), want %v (%T)", key, actual, actual, expected, expected)
		}
	}

	// Check usage
	if usage, ok := msg["usage"].(map[string]interface{}); ok {
		if usage["input_tokens"] != 250 {
			t.Errorf("input_tokens: got %v, want 250", usage["input_tokens"])
		}
		if usage["output_tokens"] != 150 {
			t.Errorf("output_tokens: got %v, want 150", usage["output_tokens"])
		}
	} else {
		t.Error("usage field missing or not a map")
	}
}

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("NewClient returned nil")
	}
}

// TestContextCancellation tests that ProcessQuery respects context cancellation
func TestContextCancellation(t *testing.T) {
	// This test is skipped because ProcessQuery creates its own transport
	// and we can't easily mock it
	t.Skip("Skipping context cancellation test - requires transport mocking")
}