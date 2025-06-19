package validation

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		want      string
		wantErr   bool
	}{
		{
			name:      "normal string",
			input:     "Hello, World!",
			maxLength: 100,
			want:      "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "string with leading/trailing whitespace",
			input:     "  Hello, World!  ",
			maxLength: 100,
			want:      "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "string with null bytes",
			input:     "Hello\x00World",
			maxLength: 100,
			want:      "HelloWorld",
			wantErr:   false,
		},
		{
			name:      "empty string",
			input:     "",
			maxLength: 100,
			want:      "",
			wantErr:   false,
		},
		{
			name:      "string exceeds max length",
			input:     strings.Repeat("a", 101),
			maxLength: 100,
			want:      "",
			wantErr:   true,
		},
		{
			name:      "use default max length",
			input:     "Hello",
			maxLength: 0,
			want:      "Hello",
			wantErr:   false,
		},
		{
			name:      "string with multiple null bytes",
			input:     "\x00Hello\x00World\x00",
			maxLength: 100,
			want:      "HelloWorld",
			wantErr:   false,
		},
		{
			name:      "string with tabs and newlines",
			input:     "\tHello\nWorld\r",
			maxLength: 100,
			want:      "Hello\nWorld",
			wantErr:   false,
		},
		{
			name:      "unicode string",
			input:     "Hello 世界 🌍",
			maxLength: 100,
			want:      "Hello 世界 🌍",
			wantErr:   false,
		},
		{
			name:      "string at exact max length",
			input:     strings.Repeat("a", 100),
			maxLength: 100,
			want:      strings.Repeat("a", 100),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeString(tt.input, tt.maxLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeCommandArg(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "normal command arg",
			input:   "mycommand",
			want:    "mycommand",
			wantErr: false,
		},
		{
			name:    "command with hyphen",
			input:   "my-command",
			want:    "my-command",
			wantErr: false,
		},
		{
			name:    "command with underscore",
			input:   "my_command",
			want:    "my_command",
			wantErr: false,
		},
		{
			name:    "command with semicolon",
			input:   "cmd;rm -rf /",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with pipe",
			input:   "cmd|cat",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with ampersand",
			input:   "cmd&",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with dollar sign",
			input:   "cmd$VAR",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with backtick",
			input:   "cmd`echo test`",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with single quote",
			input:   "cmd'test'",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with double quote",
			input:   `cmd"test"`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with parentheses",
			input:   "cmd(test)",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with brackets",
			input:   "cmd[test]",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with braces",
			input:   "cmd{test}",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with asterisk",
			input:   "cmd*",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with question mark",
			input:   "cmd?",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with exclamation",
			input:   "cmd!",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with tilde",
			input:   "~cmd",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with space",
			input:   "cmd test",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with dot",
			input:   "cmd.exe",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with slash",
			input:   "cmd/test",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with backslash",
			input:   "cmd\\test",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with less than",
			input:   "cmd<input",
			want:    "",
			wantErr: true,
		},
		{
			name:    "command with greater than",
			input:   "cmd>output",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "string with whitespace",
			input:   "  cmd  ",
			want:    "cmd",
			wantErr: false,
		},
		{
			name:    "very long command",
			input:   strings.Repeat("a", MaxStringLength+1),
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeCommandArg(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeCommandArg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeCommandArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{
			name:    "empty model (allowed)",
			model:   "",
			wantErr: false,
		},
		{
			name:    "claude-3-opus-20240229",
			model:   "claude-3-opus-20240229",
			wantErr: false,
		},
		{
			name:    "claude-3-sonnet-20240229",
			model:   "claude-3-sonnet-20240229",
			wantErr: false,
		},
		{
			name:    "claude-3-haiku-20240307",
			model:   "claude-3-haiku-20240307",
			wantErr: false,
		},
		{
			name:    "claude-3-5-sonnet-20241022",
			model:   "claude-3-5-sonnet-20241022",
			wantErr: false,
		},
		{
			name:    "claude-3-5-haiku-20241022",
			model:   "claude-3-5-haiku-20241022",
			wantErr: false,
		},
		{
			name:    "future claude model",
			model:   "claude-4-opus-20250101",
			wantErr: false,
		},
		{
			name:    "gpt-4 (invalid)",
			model:   "gpt-4",
			wantErr: true,
		},
		{
			name:    "gemini-pro (invalid)",
			model:   "gemini-pro",
			wantErr: true,
		},
		{
			name:    "claude (without version)",
			model:   "claude",
			wantErr: true,
		},
		{
			name:    "claude- (edge case)",
			model:   "claude-",
			wantErr: false,
		},
		{
			name:    "CLAUDE-3-opus (case sensitive)",
			model:   "CLAUDE-3-opus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModel(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		checkPath func(string) bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "absolute path",
			path:    "/home/user/file.txt",
			wantErr: false,
			checkPath: func(cleaned string) bool {
				return filepath.IsAbs(cleaned)
			},
		},
		{
			name:    "relative path",
			path:    "file.txt",
			wantErr: false,
			checkPath: func(cleaned string) bool {
				return filepath.IsAbs(cleaned)
			},
		},
		{
			name:    "path with ..",
			path:    "/home/user/../admin/secret.txt",
			wantErr: false,
			checkPath: func(cleaned string) bool {
				// After cleaning, .. should be resolved
				return !strings.Contains(cleaned, "..")
			},
		},
		{
			name:    "path traversal attempt",
			path:    "/home/user/../../etc/passwd",
			wantErr: false,
			checkPath: func(cleaned string) bool {
				// After cleaning, path should not contain ..
				return !strings.Contains(cleaned, "..")
			},
		},
		{
			name:    "path with single dot",
			path:    "./file.txt",
			wantErr: false,
			checkPath: func(cleaned string) bool {
				return filepath.IsAbs(cleaned)
			},
		},
		{
			name:    "complex path",
			path:    "/home/user/./docs/../files/./document.txt",
			wantErr: false,
			checkPath: func(cleaned string) bool {
				return filepath.IsAbs(cleaned) && !strings.Contains(cleaned, "..")
			},
		},
		{
			name:    "Windows-style path",
			path:    `C:\Users\test\file.txt`,
			wantErr: false,
			checkPath: func(cleaned string) bool {
				if runtime.GOOS == "windows" {
					return filepath.IsAbs(cleaned)
				}
				return true // Skip check on non-Windows
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tt.checkPath != nil {
				if !tt.checkPath(got) {
					t.Errorf("ValidatePath() = %v, failed check function", got)
				}
			}
		})
	}
}

func TestValidateWorkingDirectory(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{
			name:    "empty directory (allowed)",
			dir:     "",
			wantErr: false,
		},
		{
			name:    "valid absolute directory",
			dir:     "/home/user",
			wantErr: false,
		},
		{
			name:    "relative directory",
			dir:     "mydir",
			wantErr: false,
		},
		{
			name:    "directory with ..",
			dir:     "/home/user/../admin",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateWorkingDirectory(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkingDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.dir == "" && got != "" {
				t.Errorf("ValidateWorkingDirectory() = %v, want empty string for empty input", got)
			}
		})
	}
}

func TestTruncateError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		maxLength int
		want      string
	}{
		{
			name:      "nil error",
			err:       nil,
			maxLength: 100,
			want:      "",
		},
		{
			name:      "short error",
			err:       errors.New("short error"),
			maxLength: 100,
			want:      "short error",
		},
		{
			name:      "long error",
			err:       errors.New(strings.Repeat("a", 110)),
			maxLength: 100,
			want:      strings.Repeat("a", 100) + "...",
		},
		{
			name:      "error with file path (Unix)",
			err:       errors.New("failed to open /etc/passwd"),
			maxLength: 100,
			want:      "failed to open [path]",
		},
		{
			name:      "error with Windows path",
			err:       errors.New("failed to open C:\\Windows\\System32\\config"),
			maxLength: 100,
			want:      "failed to open [path]",
		},
		{
			name:      "error with multiple paths",
			err:       errors.New("copy /src/file.txt to /dst/file.txt failed"),
			maxLength: 100,
			want:      "copy [path] to [path] failed",
		},
		{
			name:      "error with home directory",
			err:       errors.New("cannot access /home/user/.ssh/id_rsa"),
			maxLength: 100,
			want:      "cannot access [path]",
		},
		{
			name:      "complex error with paths and truncation",
			err:       fmt.Errorf("failed to process %s: %s", "/very/long/path/to/some/file/that/exceeds/the/maximum/length/allowed/for/error/messages.txt", strings.Repeat("x", 50)),
			maxLength: 50,
			want:      "failed to process [path] " + strings.Repeat("x", 25) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateError(tt.err, tt.maxLength)
			if got != tt.want {
				t.Errorf("TruncateError() = %q (len=%d), want %q (len=%d)", got, len(got), tt.want, len(tt.want))
			}
		})
	}
}

func TestFilterEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		env      []string
		expected []string
	}{
		{
			name:     "empty environment",
			env:      []string{},
			expected: []string{},
		},
		{
			name: "safe environment variables",
			env: []string{
				"CLAUDE_API_KEY=test",
				"LANG=en_US.UTF-8",
				"LC_ALL=C",
				"TZ=UTC",
				"TERM=xterm",
				"USER=testuser",
				"HOME=/home/testuser",
				"PATH=/usr/bin:/bin",
				"TMPDIR=/tmp",
				"TEMP=/tmp",
				"TMP=/tmp",
			},
			expected: []string{
				"CLAUDE_API_KEY=test",
				"LANG=en_US.UTF-8",
				"LC_ALL=C",
				"TZ=UTC",
				"TERM=xterm",
				"USER=testuser",
				"HOME=/home/testuser",
				"PATH=/usr/bin:/bin",
				"TMPDIR=/tmp",
				"TEMP=/tmp",
				"TMP=/tmp",
			},
		},
		{
			name: "blocked environment variables",
			env: []string{
				"AWS_SECRET_ACCESS_KEY=secret",
				"AWS_SESSION_TOKEN=token",
				"GITHUB_TOKEN=ghp_xxx",
				"NPM_TOKEN=npm_xxx",
				"ANTHROPIC_API_KEY=sk-xxx",
				"CLAUDE_API_KEY=allowed",
			},
			expected: []string{
				"CLAUDE_API_KEY=allowed",
			},
		},
		{
			name: "mixed safe and unsafe variables",
			env: []string{
				"PATH=/usr/bin",
				"SECRET_KEY=secret",
				"HOME=/home/user",
				"DATABASE_PASSWORD=pass",
				"LANG=en_US",
				"API_TOKEN=token",
				"CLAUDE_CONFIG=test",
			},
			expected: []string{
				"PATH=/usr/bin",
				"HOME=/home/user",
				"LANG=en_US",
				"CLAUDE_CONFIG=test",
			},
		},
		{
			name: "malformed environment entries",
			env: []string{
				"VALID=value",
				"INVALID_NO_EQUALS",
				"=NO_KEY",
				"MULTIPLE=EQUALS=SIGNS",
				"PATH=/usr/bin",
			},
			expected: []string{
				"PATH=/usr/bin",
			},
		},
		{
			name: "environment variables with special characters",
			env: []string{
				"PATH=/usr/bin:/usr/local/bin",
				"CLAUDE_OPTIONS=--verbose --debug",
				"LC_NAME=en_US.UTF-8",
				"HOME=/home/user with spaces",
			},
			expected: []string{
				"PATH=/usr/bin:/usr/local/bin",
				"CLAUDE_OPTIONS=--verbose --debug",
				"LC_NAME=en_US.UTF-8",
				"HOME=/home/user with spaces",
			},
		},
		{
			name: "case sensitivity check",
			env: []string{
				"path=/usr/bin",      // lowercase should not match
				"PATH=/usr/bin",      // uppercase should match
				"Claude_API=test",    // should not match (wrong case)
				"CLAUDE_API=test",    // should match
			},
			expected: []string{
				"PATH=/usr/bin",
				"CLAUDE_API=test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterEnvironment(tt.env)
			
			// Check length
			if len(got) != len(tt.expected) {
				t.Errorf("FilterEnvironment() returned %d items, want %d", len(got), len(tt.expected))
				t.Errorf("Got: %v", got)
				t.Errorf("Expected: %v", tt.expected)
				return
			}
			
			// Create maps for easy comparison
			gotMap := make(map[string]bool)
			for _, e := range got {
				gotMap[e] = true
			}
			
			// Check each expected value exists
			for _, e := range tt.expected {
				if !gotMap[e] {
					t.Errorf("FilterEnvironment() missing expected value: %s", e)
				}
			}
			
			// Check no unexpected values
			expectedMap := make(map[string]bool)
			for _, e := range tt.expected {
				expectedMap[e] = true
			}
			
			for _, e := range got {
				if !expectedMap[e] {
					t.Errorf("FilterEnvironment() included unexpected value: %s", e)
				}
			}
		})
	}
}

func TestValidationConstants(t *testing.T) {
	// Test that constants have reasonable values
	if MaxStringLength <= 0 {
		t.Error("MaxStringLength should be positive")
	}
	
	if MaxStderrLines <= 0 {
		t.Error("MaxStderrLines should be positive")
	}
	
	if MaxStderrLineLength <= 0 {
		t.Error("MaxStderrLineLength should be positive")
	}
	
	if MaxJSONSize <= 0 {
		t.Error("MaxJSONSize should be positive")
	}
	
	// Test that MaxJSONSize is reasonable (10MB)
	if MaxJSONSize != 10*1024*1024 {
		t.Errorf("MaxJSONSize = %d, want %d (10MB)", MaxJSONSize, 10*1024*1024)
	}
}

func TestShellMetacharactersRegex(t *testing.T) {
	// Test that the regex correctly identifies shell metacharacters
	metacharacters := []string{
		";", "&", "|", "<", ">", "$", "`", "\\", "'", "\"", 
		"(", ")", "[", "]", "{", "}", "*", "?", "!", "~", 
		" ", ".", "/",
	}
	
	for _, char := range metacharacters {
		if !shellMetacharacters.MatchString(char) {
			t.Errorf("shellMetacharacters regex should match %q", char)
		}
	}
	
	// Test safe characters
	safeChars := []string{"a", "A", "0", "-", "_"}
	for _, char := range safeChars {
		if shellMetacharacters.MatchString(char) {
			t.Errorf("shellMetacharacters regex should not match %q", char)
		}
	}
}

// Benchmark tests
func BenchmarkSanitizeString(b *testing.B) {
	input := strings.Repeat("Hello World ", 100)
	for i := 0; i < b.N; i++ {
		_, _ = SanitizeString(input, MaxStringLength)
	}
}

func BenchmarkSanitizeCommandArg(b *testing.B) {
	input := "my-command-with-hyphens_and_underscores"
	for i := 0; i < b.N; i++ {
		_, _ = SanitizeCommandArg(input)
	}
}

func BenchmarkValidateModel(b *testing.B) {
	model := "claude-3-opus-20240229"
	for i := 0; i < b.N; i++ {
		_ = ValidateModel(model)
	}
}

func BenchmarkFilterEnvironment(b *testing.B) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
		"SECRET_KEY=secret",
		"CLAUDE_API_KEY=test",
		"AWS_SECRET_ACCESS_KEY=secret",
		"LANG=en_US.UTF-8",
		"UNKNOWN_VAR=value",
	}
	for i := 0; i < b.N; i++ {
		_ = FilterEnvironment(env)
	}
}