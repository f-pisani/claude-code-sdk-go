package errors

import (
	"fmt"
)

// SDKError is the base error type for all Claude SDK errors
type SDKError struct {
	Message string
}

func (e SDKError) Error() string {
	return e.Message
}

// CLIConnectionError is raised when unable to connect to Claude Code
type CLIConnectionError struct {
	SDKError
}

// CLINotFoundError is raised when Claude Code is not found or not installed
type CLINotFoundError struct {
	CLIConnectionError
	CLIPath string
}

// NewCLINotFoundError creates a new CLINotFoundError
func NewCLINotFoundError(message string, cliPath string) *CLINotFoundError {
	if cliPath != "" {
		message = fmt.Sprintf("%s: %s", message, cliPath)
	}
	return &CLINotFoundError{
		CLIConnectionError: CLIConnectionError{
			SDKError: SDKError{Message: message},
		},
		CLIPath: cliPath,
	}
}

// ProcessError is raised when the CLI process fails
type ProcessError struct {
	SDKError
	ExitCode *int
	Stderr   string
}

// NewProcessError creates a new ProcessError
func NewProcessError(message string, exitCode *int, stderr string) *ProcessError {
	if exitCode != nil {
		message = fmt.Sprintf("%s (exit code: %d)", message, *exitCode)
	}
	if stderr != "" {
		message = fmt.Sprintf("%s\nError output: %s", message, stderr)
	}
	return &ProcessError{
		SDKError: SDKError{Message: message},
		ExitCode: exitCode,
		Stderr:   stderr,
	}
}

// CLIJSONDecodeError is raised when unable to decode JSON from CLI output
type CLIJSONDecodeError struct {
	SDKError
	Line          string
	OriginalError error
}

// NewCLIJSONDecodeError creates a new CLIJSONDecodeError
func NewCLIJSONDecodeError(line string, originalError error) *CLIJSONDecodeError {
	truncated := line
	if len(truncated) > 100 {
		truncated = truncated[:100] + "..."
	}
	return &CLIJSONDecodeError{
		SDKError:      SDKError{Message: fmt.Sprintf("Failed to decode JSON: %s", truncated)},
		Line:          line,
		OriginalError: originalError,
	}
}

func (e CLIJSONDecodeError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.OriginalError)
}

func (e CLIJSONDecodeError) Unwrap() error {
	return e.OriginalError
}