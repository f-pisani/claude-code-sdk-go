package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/f-pisani/claude-code-sdk-go/internal/errors"
	"github.com/f-pisani/claude-code-sdk-go/internal/validation"
)

// SubprocessCLITransport implements Transport using the Claude Code CLI
type SubprocessCLITransport struct {
	prompt  string
	options interface{}
	cliPath string
	cwd     string

	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu        sync.Mutex
	connected bool
}

// CwdProvider interface for options that provide a working directory
type CwdProvider interface {
	GetCwd() string
}

// NewSubprocessCLITransport creates a new subprocess transport
func NewSubprocessCLITransport(prompt string, options interface{}, cliPath string) *SubprocessCLITransport {
	if cliPath == "" {
		cliPath = findCLI()
	}

	// Extract cwd from options if available
	cwd := ""
	if options != nil {
		if provider, ok := options.(CwdProvider); ok {
			cwd = provider.GetCwd()
		}
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	return &SubprocessCLITransport{
		prompt:  prompt,
		options: options,
		cliPath: cliPath,
		cwd:     cwd,
	}
}

// findCLI attempts to find the Claude CLI binary
func findCLI() string {
	// Check if claude is in PATH
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}

	// Build locations based on OS
	var locations []string

	switch runtime.GOOS {
	case "windows":
		// Windows-specific locations
		locations = []string{
			filepath.Join(os.Getenv("APPDATA"), "npm", "claude.cmd"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "claude.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "npm", "claude.cmd"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "npm", "claude.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "nodejs", "claude.cmd"),
			filepath.Join(os.Getenv("ProgramFiles"), "nodejs", "claude.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "nodejs", "claude.cmd"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "nodejs", "claude.exe"),
		}

		// Add home directory locations if HOME is set
		if home := os.Getenv("HOME"); home != "" {
			locations = append(locations,
				filepath.Join(home, "node_modules", ".bin", "claude.cmd"),
				filepath.Join(home, "node_modules", ".bin", "claude.exe"),
			)
		}

		// Add USERPROFILE locations
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			locations = append(locations,
				filepath.Join(userProfile, "AppData", "Roaming", "npm", "claude.cmd"),
				filepath.Join(userProfile, "AppData", "Roaming", "npm", "claude.exe"),
				filepath.Join(userProfile, "node_modules", ".bin", "claude.cmd"),
				filepath.Join(userProfile, "node_modules", ".bin", "claude.exe"),
			)
		}

	default:
		// Unix-like systems (Linux, macOS, etc.)
		home := os.Getenv("HOME")
		locations = []string{
			filepath.Join(home, ".npm-global", "bin", "claude"),
			"/usr/local/bin/claude",
			filepath.Join(home, ".local", "bin", "claude"),
			filepath.Join(home, "node_modules", ".bin", "claude"),
			filepath.Join(home, ".yarn", "bin", "claude"),
		}

		// macOS-specific locations
		if runtime.GOOS == "darwin" {
			locations = append(locations,
				"/opt/homebrew/bin/claude",
				"/usr/local/opt/claude/bin/claude",
			)
		}
	}

	// Check each location
	for _, path := range locations {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// On Windows, check if it's executable
			if runtime.GOOS == "windows" {
				// Windows executables should end with .exe or .cmd
				if strings.HasSuffix(path, ".exe") || strings.HasSuffix(path, ".cmd") {
					return path
				}
			} else {
				// On Unix, check if it's executable
				if info.Mode()&0111 != 0 {
					return path
				}
			}
		}
	}

	return ""
}

// buildCommand constructs the CLI command with arguments
func (t *SubprocessCLITransport) buildCommand() ([]string, error) {
	cmd := []string{t.cliPath, "--output-format", "stream-json", "--verbose"}

	// Use the OptionsBuilder interface if available
	if t.options != nil {
		if builder, ok := t.options.(OptionsBuilder); ok {
			args, err := builder.BuildCLIArgs()
			if err != nil {
				return nil, fmt.Errorf("failed to build CLI args: %w", err)
			}
			cmd = append(cmd, args...)
		}
	}

	cmd = append(cmd, "--print", t.prompt)
	return cmd, nil
}

// Connect starts the subprocess
func (t *SubprocessCLITransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	if t.cliPath == "" {
		// Check if Node.js is installed
		if _, err := exec.LookPath("node"); err != nil {
			errorMsg := "Claude Code requires Node.js, which is not installed.\n\n" +
				"Install Node.js from: https://nodejs.org/\n" +
				"\nAfter installing Node.js, install Claude Code:\n" +
				"  npm install -g @anthropic-ai/claude-code"
			return errors.NewCLINotFoundError(errorMsg, "")
		}

		return errors.NewCLINotFoundError(
			"Claude Code not found. Install with:\n"+
				"  npm install -g @anthropic-ai/claude-code\n"+
				"\nIf already installed locally, try:\n"+
				"  export PATH=\"$HOME/node_modules/.bin:$PATH\"\n"+
				"\nOr specify the path when creating transport:\n"+
				"  NewSubprocessCLITransport(..., \"/path/to/claude\")",
			"",
		)
	}

	cmdArgs, err := t.buildCommand()
	if err != nil {
		return err
	}

	t.cmd = exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)

	// Validate and set working directory
	if t.cwd != "" {
		validatedCwd, err := validation.ValidateWorkingDirectory(t.cwd)
		if err != nil {
			return fmt.Errorf("invalid working directory: %w", err)
		}
		t.cmd.Dir = validatedCwd
	}

	// Set environment with filtering
	filteredEnv := validation.FilterEnvironment(os.Environ())
	t.cmd.Env = append(filteredEnv, "CLAUDE_CODE_ENTRYPOINT=sdk-go")

	// Setup pipes
	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return &errors.CLIConnectionError{
			SDKError: errors.SDKError{Message: "Failed to create stdout pipe"},
		}
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		// Clean up stdout pipe on error
		if t.stdout != nil {
			t.stdout.Close()
			t.stdout = nil
		}
		return &errors.CLIConnectionError{
			SDKError: errors.SDKError{Message: "Failed to create stderr pipe"},
		}
	}

	// Start the process
	if err := t.cmd.Start(); err != nil {
		// Clean up pipes on start failure
		if t.stdout != nil {
			t.stdout.Close()
			t.stdout = nil
		}
		if t.stderr != nil {
			t.stderr.Close()
			t.stderr = nil
		}
		if strings.Contains(err.Error(), "executable file not found") {
			return errors.NewCLINotFoundError(fmt.Sprintf("Claude Code not found at: %s", t.cliPath), t.cliPath)
		}
		return &errors.CLIConnectionError{
			SDKError: errors.SDKError{Message: "Failed to start Claude Code"},
		}
	}

	t.connected = true
	return nil
}

// Disconnect terminates the subprocess
func (t *SubprocessCLITransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected || t.cmd == nil {
		return nil
	}

	if t.cmd.Process != nil {
		// Try graceful termination first
		if err := t.cmd.Process.Signal(os.Interrupt); err == nil {
			// Wait a bit for graceful shutdown
			// Make channel buffered to prevent goroutine leak
			done := make(chan error, 1)
			go func() {
				done <- t.cmd.Wait()
			}()

			select {
			case <-done:
				// Process exited gracefully
			case <-time.After(5 * time.Second):
				// Force kill after timeout
				t.cmd.Process.Kill()
				<-done
			}
		} else {
			// If we can't send interrupt, just kill it
			t.cmd.Process.Kill()
			t.cmd.Wait()
		}
	}

	if t.stdout != nil {
		t.stdout.Close()
	}
	if t.stderr != nil {
		t.stderr.Close()
	}

	t.connected = false
	t.cmd = nil
	t.stdout = nil
	t.stderr = nil

	return nil
}

// ReceiveMessages returns channels for receiving messages and errors
func (t *SubprocessCLITransport) ReceiveMessages(ctx context.Context) (<-chan map[string]interface{}, <-chan error) {
	// Get buffer sizes from options if available
	msgBufSize := 10
	errBufSize := 1

	// Check if options has buffer size methods
	if opt, ok := t.options.(interface {
		GetMessageBufferSize() int
		GetErrorBufferSize() int
	}); ok {
		msgBufSize = opt.GetMessageBufferSize()
		errBufSize = opt.GetErrorBufferSize()
	}

	// Create channels with configurable buffer sizes
	msgCh := make(chan map[string]interface{}, msgBufSize)
	errCh := make(chan error, errBufSize)

	if !t.IsConnected() {
		t.handleNotConnected(msgCh, errCh)
		return msgCh, errCh
	}

	go func() {
		// Ensure channels are always closed, even on panic
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic in ReceiveMessages: %v", r)
			}
			close(msgCh)
			close(errCh)
		}()

		// Collect stderr in background
		stderrLines, stderrDone := t.collectStderr()

		// Process stdout messages
		if err := t.processStdout(ctx, msgCh, errCh); err != nil {
			return
		}

		// Wait for process completion and handle any errors
		<-stderrDone
		t.handleProcessExit(stderrLines, errCh)
	}()

	return msgCh, errCh
}

// IsConnected checks if the subprocess is running
func (t *SubprocessCLITransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.connected && t.cmd != nil && t.cmd.Process != nil
}

// handleNotConnected handles the case when transport is not connected
func (t *SubprocessCLITransport) handleNotConnected(msgCh chan map[string]interface{}, errCh chan error) {
	go func() {
		errCh <- &errors.CLIConnectionError{
			SDKError: errors.SDKError{Message: "Not connected"},
		}
		close(msgCh)
		close(errCh)
	}()
}

// collectStderr collects stderr output in the background with resource limits
func (t *SubprocessCLITransport) collectStderr() ([]string, <-chan struct{}) {
	var stderrLines []string
	stderrDone := make(chan struct{})

	go func() {
		defer close(stderrDone)
		scanner := bufio.NewScanner(t.stderr)
		// Set max scan buffer to prevent OOM
		scanner.Buffer(make([]byte, 0, 64*1024), validation.MaxJSONSize)

		for scanner.Scan() {
			line := scanner.Text()
			// Truncate long lines
			if len(line) > validation.MaxStderrLineLength {
				line = line[:validation.MaxStderrLineLength] + "..."
			}

			// Limit number of stderr lines collected
			if len(stderrLines) < validation.MaxStderrLines {
				stderrLines = append(stderrLines, line)
			} else if len(stderrLines) == validation.MaxStderrLines {
				stderrLines = append(stderrLines, "[stderr truncated - too many lines]")
			}
		}
	}()

	return stderrLines, stderrDone
}

// processStdout reads and processes stdout messages
func (t *SubprocessCLITransport) processStdout(ctx context.Context, msgCh chan<- map[string]interface{}, errCh chan<- error) error {
	scanner := bufio.NewScanner(t.stdout)
	// Set max scan buffer to prevent OOM
	scanner.Buffer(make([]byte, 0, 64*1024), validation.MaxJSONSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if err := t.processLine(ctx, line, msgCh, errCh); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		errCh <- &errors.CLIConnectionError{
			SDKError: errors.SDKError{Message: "Error reading stdout"},
		}
		return err
	}

	return nil
}

// processLine processes a single line of JSON output
func (t *SubprocessCLITransport) processLine(ctx context.Context, line string, msgCh chan<- map[string]interface{}, errCh chan<- error) error {
	// Check JSON size before parsing
	if len(line) > validation.MaxJSONSize {
		errCh <- errors.NewCLIJSONDecodeError("[JSON too large]", fmt.Errorf("JSON exceeds maximum size of %d bytes", validation.MaxJSONSize))
		return fmt.Errorf("JSON too large")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		// Only treat as error if it looks like JSON
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			// Truncate line for error message to prevent excessive memory use
			truncatedLine := line
			if len(truncatedLine) > 200 {
				truncatedLine = truncatedLine[:200] + "..."
			}
			errCh <- errors.NewCLIJSONDecodeError(truncatedLine, err)
			return err
		}
		return nil // Skip non-JSON lines
	}

	select {
	case msgCh <- data:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// handleProcessExit handles process exit and any associated errors
func (t *SubprocessCLITransport) handleProcessExit(stderrLines []string, errCh chan<- error) {
	if err := t.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			stderrOutput := strings.Join(stderrLines, "\n")
			if stderrOutput != "" && strings.Contains(strings.ToLower(stderrOutput), "error") {
				// Sanitize stderr output to prevent information disclosure
				sanitizedStderr := validation.TruncateError(fmt.Errorf("%s", stderrOutput), 1000)
				errCh <- errors.NewProcessError("CLI process failed", &exitCode, sanitizedStderr)
			}
		}
	}
}
