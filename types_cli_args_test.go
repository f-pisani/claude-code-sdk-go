package claudecode

import (
	"strings"
	"testing"
)

func TestBuildCLIArgs_AllOptions(t *testing.T) {
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
			name:     "empty options with defaults",
			options:  NewOptions(),
			expected: []string{},
		},
		{
			name: "system prompt",
			options: &Options{
				SystemPrompt:      "You are helpful",
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--system-prompt", "You are helpful"},
		},
		{
			name: "append system prompt",
			options: &Options{
				AppendSystemPrompt: "Additional context",
				MaxThinkingTokens:  8000,
			},
			expected: []string{"--append-system-prompt", "Additional context"},
		},
		{
			name: "allowed tools",
			options: &Options{
				AllowedTools:      []string{"Read", "Write", "Edit"},
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--allowedTools", "Read,Write,Edit"},
		},
		{
			name: "max turns",
			options: &Options{
				MaxTurns:          intPtr(5),
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--max-turns", "5"},
		},
		{
			name: "disallowed tools",
			options: &Options{
				DisallowedTools:   []string{"Delete", "Execute"},
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--disallowedTools", "Delete,Execute"},
		},
		{
			name: "model",
			options: &Options{
				Model:             "claude-3-opus",
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--model", "claude-3-opus"},
		},
		{
			name: "permission prompt tool name",
			options: &Options{
				PermissionPromptToolName: "custom-tool",
				MaxThinkingTokens:        8000,
			},
			expected: []string{"--permission-prompt-tool", "custom-tool"},
		},
		{
			name: "permission mode",
			options: &Options{
				PermissionMode:    permissionModePtr(PermissionModeAcceptEdits),
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--permission-mode", "acceptEdits"},
		},
		{
			name: "continue conversation",
			options: &Options{
				ContinueConversation: true,
				MaxThinkingTokens:    8000,
			},
			expected: []string{"--continue"},
		},
		{
			name: "resume",
			options: &Options{
				Resume:            "session-123",
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--resume", "session-123"},
		},
		{
			name: "max thinking tokens non-default",
			options: &Options{
				MaxThinkingTokens: 10000,
			},
			expected: []string{"--max-thinking-tokens", "10000"},
		},
		{
			name: "max thinking tokens default (8000)",
			options: &Options{
				MaxThinkingTokens: 8000,
			},
			expected: []string{},
		},
		{
			name: "mcp tools",
			options: &Options{
				McpTools:          []string{"tool1", "tool2"},
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--mcp-tools", "tool1,tool2"},
		},
		{
			name: "mcp servers",
			options: &Options{
				McpServers: map[string]McpServerConfig{
					"server1": {
						Transport: []string{"stdio"},
						Env:       map[string]interface{}{"KEY": "value"},
					},
				},
				MaxThinkingTokens: 8000,
			},
			expected: []string{"--mcp-config"},
		},
		{
			name: "all options combined",
			options: &Options{
				SystemPrompt:             "System",
				AppendSystemPrompt:       "Append",
				AllowedTools:             []string{"Read", "Write"},
				MaxTurns:                 intPtr(10),
				DisallowedTools:          []string{"Delete"},
				Model:                    "claude-3",
				PermissionPromptToolName: "tool",
				PermissionMode:           permissionModePtr(PermissionModeBypassPermissions),
				ContinueConversation:     true,
				Resume:                   "session",
				MaxThinkingTokens:        15000,
				McpTools:                 []string{"mcp1"},
				McpServers: map[string]McpServerConfig{
					"srv": {Transport: []string{"stdio"}},
				},
			},
			expected: []string{
				"--system-prompt", "System",
				"--append-system-prompt", "Append",
				"--allowedTools", "Read,Write",
				"--max-turns", "10",
				"--disallowedTools", "Delete",
				"--model", "claude-3",
				"--permission-prompt-tool", "tool",
				"--permission-mode", "bypassPermissions",
				"--continue",
				"--resume", "session",
				"--max-thinking-tokens", "15000",
				"--mcp-tools", "mcp1",
				"--mcp-config",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.options.BuildCLIArgs()
			if err != nil {
				t.Fatalf("BuildCLIArgs() returned error: %v", err)
			}

			// For mcp-config, we just check if the flag is present
			// since the JSON content can vary in formatting
			resultStr := strings.Join(result, " ")
			expectedStr := strings.Join(tt.expected, " ")

			if strings.Contains(expectedStr, "--mcp-config") {
				if !strings.Contains(resultStr, "--mcp-config") {
					t.Errorf("Expected --mcp-config flag to be present")
				}
				// Check that mcpServers is in the JSON
				mcpIndex := -1
				for i, arg := range result {
					if arg == "--mcp-config" && i+1 < len(result) {
						mcpIndex = i + 1
						break
					}
				}
				if mcpIndex != -1 && !strings.Contains(result[mcpIndex], "mcpServers") {
					t.Errorf("Expected mcpServers in mcp-config JSON")
				}
			} else {
				// For non-mcp-config tests, do exact comparison
				if len(result) != len(tt.expected) {
					t.Errorf("BuildCLIArgs() returned %d args, expected %d\nGot: %v\nExpected: %v",
						len(result), len(tt.expected), result, tt.expected)
					return
				}

				for i, expected := range tt.expected {
					if result[i] != expected {
						t.Errorf("BuildCLIArgs()[%d] = %q, expected %q", i, result[i], expected)
					}
				}
			}
		})
	}
}

func TestBuildCLIArgs_EdgeCases(t *testing.T) {
	t.Run("empty strings are not included", func(t *testing.T) {
		options := &Options{
			SystemPrompt:             "",
			AppendSystemPrompt:       "",
			Model:                    "",
			PermissionPromptToolName: "",
			Resume:                   "",
			Cwd:                      "",
			MaxThinkingTokens:        8000,
		}
		result, err := options.BuildCLIArgs()
		if err != nil {
			t.Fatalf("BuildCLIArgs() returned error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected no args for empty strings, got: %v", result)
		}
	})

	t.Run("empty slices are not included", func(t *testing.T) {
		options := &Options{
			AllowedTools:      []string{},
			DisallowedTools:   []string{},
			McpTools:          []string{},
			MaxThinkingTokens: 8000,
		}
		result, err := options.BuildCLIArgs()
		if err != nil {
			t.Fatalf("BuildCLIArgs() returned error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected no args for empty slices, got: %v", result)
		}
	})

	t.Run("nil MaxTurns is not included", func(t *testing.T) {
		options := &Options{
			MaxTurns:          nil,
			MaxThinkingTokens: 8000,
		}
		result, err := options.BuildCLIArgs()
		if err != nil {
			t.Fatalf("BuildCLIArgs() returned error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected no args for nil MaxTurns, got: %v", result)
		}
	})

	t.Run("nil PermissionMode is not included", func(t *testing.T) {
		options := &Options{
			PermissionMode:    nil,
			MaxThinkingTokens: 8000,
		}
		result, err := options.BuildCLIArgs()
		if err != nil {
			t.Fatalf("BuildCLIArgs() returned error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected no args for nil PermissionMode, got: %v", result)
		}
	})
}

func TestBuildCLIArgs_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		options     *Options
		expectedErr string
	}{
		{
			name: "invalid permission mode",
			options: &Options{
				PermissionMode:    permissionModePtr("invalid-mode"),
				MaxThinkingTokens: 8000,
			},
			expectedErr: "invalid permission mode",
		},
		{
			name: "negative max turns",
			options: &Options{
				MaxTurns:          intPtr(-1),
				MaxThinkingTokens: 8000,
			},
			expectedErr: "max turns must be between 0 and 1000",
		},
		{
			name: "max turns too high",
			options: &Options{
				MaxTurns:          intPtr(1001),
				MaxThinkingTokens: 8000,
			},
			expectedErr: "max turns must be between 0 and 1000",
		},
		{
			name: "negative max thinking tokens",
			options: &Options{
				MaxThinkingTokens: -1,
			},
			expectedErr: "max thinking tokens must be between 0 and 100000",
		},
		{
			name: "max thinking tokens too high",
			options: &Options{
				MaxThinkingTokens: 100001,
			},
			expectedErr: "max thinking tokens must be between 0 and 100000",
		},
		{
			name: "invalid model name",
			options: &Options{
				Model:             "gpt-4",
				MaxThinkingTokens: 8000,
			},
			expectedErr: "invalid model",
		},
		{
			name: "tool name with shell metacharacters",
			options: &Options{
				AllowedTools:      []string{"Read", "Write; rm -rf /"},
				MaxThinkingTokens: 8000,
			},
			expectedErr: "shell metacharacters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.options.BuildCLIArgs()
			if err == nil {
				t.Errorf("BuildCLIArgs() expected error containing %q, but got nil", tt.expectedErr)
				return
			}
			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("BuildCLIArgs() error = %v, expected error containing %q", err, tt.expectedErr)
			}
		})
	}
}

// Helper function
func permissionModePtr(mode PermissionMode) *PermissionMode {
	return &mode
}
