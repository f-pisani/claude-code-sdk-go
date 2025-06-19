package claudecode

import (
	"strings"
	"testing"
)

func TestSecurityValidation(t *testing.T) {
	tests := []struct {
		name        string
		options     *Options
		shouldError bool
		errorMsg    string
	}{
		{
			name: "command injection in system prompt",
			options: &Options{
				SystemPrompt:      "Hello; rm -rf /",
				MaxThinkingTokens: 8000,
			},
			shouldError: false, // System prompts are sanitized but not rejected
		},
		{
			name: "command injection in tool name",
			options: &Options{
				AllowedTools:      []string{"Read", "Write && malicious-command"},
				MaxThinkingTokens: 8000,
			},
			shouldError: true,
			errorMsg:    "shell metacharacters",
		},
		{
			name: "path traversal attempt in resume",
			options: &Options{
				Resume:            "../../../etc/passwd",
				MaxThinkingTokens: 8000,
			},
			shouldError: true,
			errorMsg:    "shell metacharacters", // dots are caught as metacharacters
		},
		{
			name: "SQL injection attempt in model",
			options: &Options{
				Model:             "'; DROP TABLE users; --",
				MaxThinkingTokens: 8000,
			},
			shouldError: true,
			errorMsg:    "invalid model",
		},
		{
			name: "valid claude model",
			options: &Options{
				Model:             "claude-3-opus-20240229",
				MaxThinkingTokens: 8000,
			},
			shouldError: false,
		},
		{
			name: "future claude model",
			options: &Options{
				Model:             "claude-4-future-model",
				MaxThinkingTokens: 8000,
			},
			shouldError: false, // Should allow future models starting with claude-
		},
		{
			name: "XSS attempt in permission tool name",
			options: &Options{
				PermissionPromptToolName: "<script>alert('xss')</script>",
				MaxThinkingTokens:        8000,
			},
			shouldError: true,
			errorMsg:    "shell metacharacters",
		},
		{
			name: "buffer overflow attempt with very long string",
			options: &Options{
				SystemPrompt:      strings.Repeat("A", 15000),
				MaxThinkingTokens: 8000,
			},
			shouldError: true,
			errorMsg:    "exceeds maximum length",
		},
		{
			name: "null byte injection",
			options: &Options{
				Resume:            "session\x00malicious",
				MaxThinkingTokens: 8000,
			},
			shouldError: false, // Null bytes are sanitized out
		},
		{
			name: "negative max thinking tokens",
			options: &Options{
				MaxThinkingTokens: -1000,
			},
			shouldError: true,
			errorMsg:    "must be between 0 and 100000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.options.BuildCLIArgs()

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("SafeBoolPtr", func(t *testing.T) {
		// Test nil pointer
		if SafeBoolPtr(nil) != false {
			t.Error("SafeBoolPtr(nil) should return false")
		}

		// Test true pointer
		trueVal := true
		if SafeBoolPtr(&trueVal) != true {
			t.Error("SafeBoolPtr(&true) should return true")
		}

		// Test false pointer
		falseVal := false
		if SafeBoolPtr(&falseVal) != false {
			t.Error("SafeBoolPtr(&false) should return false")
		}
	})

	t.Run("SafeFloat64Ptr", func(t *testing.T) {
		// Test nil pointer
		if SafeFloat64Ptr(nil) != 0.0 {
			t.Error("SafeFloat64Ptr(nil) should return 0.0")
		}

		// Test non-zero pointer
		val := 3.14
		if SafeFloat64Ptr(&val) != 3.14 {
			t.Error("SafeFloat64Ptr(&3.14) should return 3.14")
		}
	})

	t.Run("SafeIntPtr", func(t *testing.T) {
		// Test nil pointer
		if SafeIntPtr(nil) != 0 {
			t.Error("SafeIntPtr(nil) should return 0")
		}

		// Test non-zero pointer
		val := 42
		if SafeIntPtr(&val) != 42 {
			t.Error("SafeIntPtr(&42) should return 42")
		}
	})

	t.Run("SafeStringPtr", func(t *testing.T) {
		// Test nil pointer
		if SafeStringPtr(nil) != "" {
			t.Error("SafeStringPtr(nil) should return empty string")
		}

		// Test non-empty pointer
		val := "hello"
		if SafeStringPtr(&val) != "hello" {
			t.Error("SafeStringPtr(&\"hello\") should return \"hello\"")
		}
	})
}
