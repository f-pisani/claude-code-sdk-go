package claudecode

import (
	"github.com/f-pisani/claude-code-sdk-go/internal/errors"
)

// Re-export error types from internal package

// SDKError is the base error type for all Claude SDK errors
type SDKError = errors.SDKError

// CLIConnectionError is raised when unable to connect to Claude Code
type CLIConnectionError = errors.CLIConnectionError

// CLINotFoundError is raised when Claude Code is not found or not installed
type CLINotFoundError = errors.CLINotFoundError

// NewCLINotFoundError creates a new CLINotFoundError
var NewCLINotFoundError = errors.NewCLINotFoundError

// ProcessError is raised when the CLI process fails
type ProcessError = errors.ProcessError

// NewProcessError creates a new ProcessError
var NewProcessError = errors.NewProcessError

// CLIJSONDecodeError is raised when unable to decode JSON from CLI output
type CLIJSONDecodeError = errors.CLIJSONDecodeError

// NewCLIJSONDecodeError creates a new CLIJSONDecodeError
var NewCLIJSONDecodeError = errors.NewCLIJSONDecodeError
