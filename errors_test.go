package claudecode

import (
	"errors"
	"strings"
	"testing"
)

func TestErrors(t *testing.T) {
	t.Run("SDKError", func(t *testing.T) {
		err := SDKError{Message: "test error"}
		if err.Error() != "test error" {
			t.Errorf("Expected 'test error', got %s", err.Error())
		}
	})

	t.Run("CLINotFoundError", func(t *testing.T) {
		err := NewCLINotFoundError("Claude Code not found", "/usr/bin/claude")
		if !strings.Contains(err.Error(), "Claude Code not found: /usr/bin/claude") {
			t.Errorf("Expected error to contain path, got %s", err.Error())
		}
		if err.CLIPath != "/usr/bin/claude" {
			t.Errorf("Expected CLIPath '/usr/bin/claude', got %s", err.CLIPath)
		}
	})

	t.Run("ProcessError with exit code", func(t *testing.T) {
		exitCode := 1
		err := NewProcessError("Process failed", &exitCode, "")
		if !strings.Contains(err.Error(), "exit code: 1") {
			t.Errorf("Expected error to contain exit code, got %s", err.Error())
		}
		if *err.ExitCode != 1 {
			t.Errorf("Expected exit code 1, got %d", *err.ExitCode)
		}
	})

	t.Run("ProcessError with stderr", func(t *testing.T) {
		err := NewProcessError("Process failed", nil, "stderr output")
		if !strings.Contains(err.Error(), "stderr output") {
			t.Errorf("Expected error to contain stderr, got %s", err.Error())
		}
		if err.Stderr != "stderr output" {
			t.Errorf("Expected stderr 'stderr output', got %s", err.Stderr)
		}
	})

	t.Run("CLIJSONDecodeError", func(t *testing.T) {
		originalErr := errors.New("invalid character")
		longLine := strings.Repeat("x", 150)
		err := NewCLIJSONDecodeError(longLine, originalErr)

		// Check truncation
		if !strings.Contains(err.Error(), "...") {
			t.Error("Expected truncated line to contain '...'")
		}

		// Check original error is included
		if !strings.Contains(err.Error(), "invalid character") {
			t.Error("Expected error to contain original error message")
		}

		// Check Unwrap
		if err.Unwrap() != originalErr {
			t.Error("Expected Unwrap to return original error")
		}
	})
}
