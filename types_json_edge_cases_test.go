package claudecode

import (
	"encoding/json"
	"testing"
)

func TestJSONMarshalingEdgeCases(t *testing.T) {
	t.Run("ContentBlock with nil fields", func(t *testing.T) {
		// Test ToolResultBlock with nil Content and IsError
		block := ToolResultBlock{
			ToolUseID: "test-id",
			Content:   nil,
			IsError:   nil,
		}

		data, err := json.Marshal(block)
		if err != nil {
			t.Fatalf("Failed to marshal ToolResultBlock with nil fields: %v", err)
		}

		var decoded ToolResultBlock
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ToolResultBlock: %v", err)
		}

		if decoded.ToolUseID != block.ToolUseID {
			t.Errorf("ToolUseID mismatch: got %q, want %q", decoded.ToolUseID, block.ToolUseID)
		}
		if decoded.Content != nil {
			t.Errorf("Expected nil Content, got %v", decoded.Content)
		}
		if decoded.IsError != nil {
			t.Errorf("Expected nil IsError, got %v", decoded.IsError)
		}
	})

	t.Run("ToolResultBlock with complex content", func(t *testing.T) {
		// Test with string content
		stringContent := ToolResultBlock{
			ToolUseID: "test-1",
			Content:   "This is a string result",
			IsError:   boolPtr(false),
		}

		data, err := json.Marshal(stringContent)
		if err != nil {
			t.Fatalf("Failed to marshal ToolResultBlock with string content: %v", err)
		}

		var decoded1 ToolResultBlock
		err = json.Unmarshal(data, &decoded1)
		if err != nil {
			t.Fatalf("Failed to unmarshal ToolResultBlock: %v", err)
		}

		if decoded1.Content != "This is a string result" {
			t.Errorf("String content mismatch: got %v, want %v", decoded1.Content, "This is a string result")
		}

		// Test with array content
		arrayContent := ToolResultBlock{
			ToolUseID: "test-2",
			Content: []map[string]interface{}{
				{"type": "text", "text": "First item"},
				{"type": "image", "url": "http://example.com/image.png"},
			},
			IsError: boolPtr(true),
		}

		data, err = json.Marshal(arrayContent)
		if err != nil {
			t.Fatalf("Failed to marshal ToolResultBlock with array content: %v", err)
		}

		var decoded2 ToolResultBlock
		err = json.Unmarshal(data, &decoded2)
		if err != nil {
			t.Fatalf("Failed to unmarshal ToolResultBlock: %v", err)
		}

		if contentArray, ok := decoded2.Content.([]interface{}); ok {
			if len(contentArray) != 2 {
				t.Errorf("Array content length mismatch: got %d, want 2", len(contentArray))
			}
		} else {
			t.Errorf("Expected array content, got %T", decoded2.Content)
		}
	})

	t.Run("AssistantMessage empty content array", func(t *testing.T) {
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
			t.Fatalf("Failed to unmarshal AssistantMessage: %v", err)
		}

		if len(decoded.Content) != 0 {
			t.Errorf("Expected empty content array, got %d items", len(decoded.Content))
		}
	})

	t.Run("Message type without _type field", func(t *testing.T) {
		// Test unmarshaling JSON without _type field
		invalidJSON := `{"content": "test message"}`

		var msg interface{}
		err := json.Unmarshal([]byte(invalidJSON), &msg)
		if err != nil {
			t.Fatalf("Failed to unmarshal invalid JSON: %v", err)
		}

		// convertMessage should return nil for invalid message
		result := convertMessage(msg)
		if result != nil {
			t.Errorf("Expected nil for message without _type, got %T", result)
		}
	})

	t.Run("ContentBlock without _blockType field", func(t *testing.T) {
		// Test unmarshaling content block without _blockType
		invalidBlock := map[string]interface{}{
			"text": "some text",
		}

		result := convertContentBlock(invalidBlock)
		if result != nil {
			t.Errorf("Expected nil for block without _blockType, got %T", result)
		}
	})

	t.Run("ResultMessage with all nil optional fields", func(t *testing.T) {
		msg := ResultMessage{
			Subtype:       "completion",
			DurationMs:    100,
			DurationAPIMs: 50,
			IsError:       false,
			NumTurns:      1,
			SessionID:     "test-session",
			TotalCostUSD:  nil,
			Usage:         nil,
			Result:        nil,
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

		if decoded.TotalCostUSD != nil {
			t.Errorf("Expected nil TotalCostUSD, got %v", decoded.TotalCostUSD)
		}
		if decoded.Usage != nil {
			t.Errorf("Expected nil Usage, got %v", decoded.Usage)
		}
		if decoded.Result != nil {
			t.Errorf("Expected nil Result, got %v", decoded.Result)
		}
	})

	t.Run("Options with empty maps and slices", func(t *testing.T) {
		opts := Options{
			AllowedTools:    []string{},
			DisallowedTools: []string{},
			McpTools:        []string{},
			McpServers:      map[string]McpServerConfig{},
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

		// JSON unmarshaling may convert empty slices to nil
		if decoded.AllowedTools != nil && len(decoded.AllowedTools) != 0 {
			t.Errorf("Expected empty or nil AllowedTools, got %v", decoded.AllowedTools)
		}
	})

	t.Run("McpServerConfig with nil Env", func(t *testing.T) {
		config := McpServerConfig{
			Transport: []string{"stdio"},
			Env:       nil,
		}

		data, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("Failed to marshal McpServerConfig: %v", err)
		}

		// Check that Env is omitted when nil
		if string(data) == `{"transport":["stdio"]}` || string(data) == `{"transport":["stdio"],"env":null}` {
			// Both are acceptable
		} else {
			t.Errorf("Unexpected JSON output: %s", string(data))
		}
	})

	t.Run("ToolUseBlock with nested input", func(t *testing.T) {
		block := ToolUseBlock{
			ID:   "test-tool",
			Name: "ComplexTool",
			Input: map[string]interface{}{
				"nested": map[string]interface{}{
					"deep": map[string]interface{}{
						"value": "deeply nested",
						"array": []interface{}{1, 2, 3},
					},
				},
				"number": 42.5,
				"bool":   true,
				"null":   nil,
			},
		}

		data, err := json.Marshal(block)
		if err != nil {
			t.Fatalf("Failed to marshal ToolUseBlock with nested input: %v", err)
		}

		var decoded ToolUseBlock
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal ToolUseBlock: %v", err)
		}

		// Verify nested structure is preserved
		if nested, ok := decoded.Input["nested"].(map[string]interface{}); ok {
			if deep, ok := nested["deep"].(map[string]interface{}); ok {
				if deep["value"] != "deeply nested" {
					t.Errorf("Nested value mismatch: got %v, want %v", deep["value"], "deeply nested")
				}
			} else {
				t.Errorf("Expected deep to be a map, got %T", nested["deep"])
			}
		} else {
			t.Errorf("Expected nested to be a map, got %T", decoded.Input["nested"])
		}
	})
}

func boolPtr(b bool) *bool {
	return &b
}
