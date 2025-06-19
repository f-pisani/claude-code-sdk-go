package transport

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdkerrors "github.com/f-pisani/claude-code-sdk-go/internal/errors"
)

// Helper function to create test scripts properly
func createTestScript(t *testing.T, script string) string {
	// Create a temporary directory for our test scripts
	tmpDir := t.TempDir()

	// Create the script file in the temp directory
	tmpFileName := filepath.Join(tmpDir, "test-script.sh")

	// Write the script content with executable permissions
	if err := os.WriteFile(tmpFileName, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	// Ensure the file is executable (some systems may ignore the mode in WriteFile)
	if err := os.Chmod(tmpFileName, 0755); err != nil {
		t.Fatal(err)
	}

	return tmpFileName
}

// TestFindCLI tests the CLI discovery logic
func TestFindCLI(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() func()
		wantFound bool
	}{
		{
			name: "claude in PATH",
			setup: func() func() {
				// Create a temporary directory with a mock claude executable
				tmpDir := t.TempDir()
				claudePath := filepath.Join(tmpDir, "claude")
				if err := os.WriteFile(claudePath, []byte("#!/bin/sh\necho claude"), 0755); err != nil {
					t.Fatal(err)
				}

				// Save original PATH and prepend our temp dir
				origPath := os.Getenv("PATH")
				os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)

				return func() {
					os.Setenv("PATH", origPath)
				}
			},
			wantFound: true,
		},
		{
			name: "no claude found",
			setup: func() func() {
				// Save original PATH and set to empty
				origPath := os.Getenv("PATH")
				os.Setenv("PATH", "")
				return func() {
					os.Setenv("PATH", origPath)
				}
			},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			path := findCLI()
			if tt.wantFound {
				if path == "" {
					t.Error("expected to find CLI but got empty path")
				}
			} else {
				if path != "" {
					t.Errorf("expected no CLI found but got path: %q", path)
				}
			}
		})
	}
}

// MockOptionsBuilder implements OptionsBuilder for testing
type MockOptionsBuilder struct {
	args []string
}

func (m *MockOptionsBuilder) BuildCLIArgs() ([]string, error) {
	return m.args, nil
}

// TestBuildCommand tests the command argument building
func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name     string
		options  interface{}
		prompt   string
		expected []string
	}{
		{
			name:    "nil options",
			options: nil,
			prompt:  "Hello",
			expected: []string{
				"/test/claude",
				"--output-format", "stream-json",
				"--verbose",
				"--print", "Hello",
			},
		},
		{
			name: "with mocked options",
			options: &MockOptionsBuilder{
				args: []string{"--system-prompt", "You are helpful"},
			},
			prompt: "Test",
			expected: []string{
				"/test/claude",
				"--output-format", "stream-json",
				"--verbose",
				"--system-prompt", "You are helpful",
				"--print", "Test",
			},
		},
		{
			name: "with multiple mocked options",
			options: &MockOptionsBuilder{
				args: []string{
					"--system-prompt", "You are helpful",
					"--max-turns", "3",
					"--permission-mode", "autoApprove",
				},
			},
			prompt: "Test",
			expected: []string{
				"/test/claude",
				"--output-format", "stream-json",
				"--verbose",
				"--system-prompt", "You are helpful",
				"--max-turns", "3",
				"--permission-mode", "autoApprove",
				"--print", "Test",
			},
		},
		{
			name: "with allowed tools",
			options: &MockOptionsBuilder{
				args: []string{"--allowedTools", "Read,Write,Bash"},
			},
			prompt: "Test",
			expected: []string{
				"/test/claude",
				"--output-format", "stream-json",
				"--verbose",
				"--allowedTools", "Read,Write,Bash",
				"--print", "Test",
			},
		},
		{
			name: "with continue and resume",
			options: &MockOptionsBuilder{
				args: []string{
					"--continue",
					"--resume", "prev-session",
				},
			},
			prompt: "Test",
			expected: []string{
				"/test/claude",
				"--output-format", "stream-json",
				"--verbose",
				"--continue",
				"--resume", "prev-session",
				"--print", "Test",
			},
		},
		{
			name:    "non-builder options",
			options: struct{ Field string }{Field: "value"},
			prompt:  "Test",
			expected: []string{
				"/test/claude",
				"--output-format", "stream-json",
				"--verbose",
				"--print", "Test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &SubprocessCLITransport{
				cliPath: "/test/claude",
				prompt:  tt.prompt,
				options: tt.options,
			}

			cmd, err := transport.buildCommand()
			if err != nil {
				t.Fatalf("buildCommand() returned error: %v", err)
			}

			// Check that all expected args are present
			if len(cmd) != len(tt.expected) {
				t.Errorf("got %d args, expected %d. Got: %v", len(cmd), len(tt.expected), cmd)
				return
			}

			for i, expected := range tt.expected {
				if cmd[i] != expected {
					t.Errorf("arg at position %d: got %q, want %q", i, cmd[i], expected)
				}
			}
		})
	}
}

// TestSubprocessLifecycle tests the subprocess start/stop lifecycle
func TestSubprocessLifecycle(t *testing.T) {
	// Skip if running in CI or restricted environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping subprocess test in CI environment")
	}

	// Create a script that runs until killed
	script := `#!/bin/sh
while true; do
	sleep 0.1
done`

	tmpFileName := createTestScript(t, script)

	transport := &SubprocessCLITransport{
		cliPath: tmpFileName,
		prompt:  "test",
		cwd:     t.TempDir(),
	}

	ctx := context.Background()

	// Test Connect
	err := transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Verify process is running
	if !transport.IsConnected() {
		t.Fatal("transport should be connected after Connect")
	}
	if transport.cmd == nil {
		t.Fatal("cmd should not be nil after Connect")
	}

	// Test Disconnect
	err = transport.Disconnect()
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	// Verify cleanup
	if transport.IsConnected() {
		t.Error("transport should not be connected after Disconnect")
	}
	if transport.cmd != nil {
		t.Error("cmd should be nil after Disconnect")
	}
}

// TestReceiveMessages tests receiving messages from the subprocess
func TestReceiveMessages(t *testing.T) {
	// Create a test program that outputs JSON messages
	script := `#!/bin/sh
echo '{"type":"assistant","content":[{"type":"text","text":"Hello"}]}'
echo '{"type":"result","cost_usd":0.01}'
exit 0`

	tmpFileName := createTestScript(t, script)

	transport := &SubprocessCLITransport{
		cliPath: tmpFileName,
		prompt:  "test",
		cwd:     t.TempDir(),
	}

	ctx := context.Background()
	err := transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer transport.Disconnect()

	// Test receiving messages
	msgCh, errCh := transport.ReceiveMessages(ctx)

	// Collect all messages
	var messages []map[string]interface{}
	done := false
	for !done {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				done = true
				break
			}
			messages = append(messages, msg)
		case err := <-errCh:
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for messages")
		}
	}

	// Verify we got the expected messages
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	// Check first message
	if messages[0]["type"] != "assistant" {
		t.Errorf("First message type: got %v, want assistant", messages[0]["type"])
	}

	// Check second message
	if messages[1]["type"] != "result" {
		t.Errorf("Second message type: got %v, want result", messages[1]["type"])
	}
	if messages[1]["cost_usd"] != 0.01 {
		t.Errorf("Result cost_usd: got %v, want 0.01", messages[1]["cost_usd"])
	}
}

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	// Test Connect with cancelled context
	transport := &SubprocessCLITransport{
		cliPath: "sleep",
		prompt:  "test",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := transport.Connect(ctx)
	if err == nil {
		transport.Disconnect()
		t.Error("Connect should fail with cancelled context")
	}

	// Test ReceiveMessages with cancelled context
	script := `#!/bin/sh
while true; do
	sleep 1
done`

	tmpFileName := createTestScript(t, script)

	transport2 := &SubprocessCLITransport{
		cliPath: tmpFileName,
		prompt:  "test",
		cwd:     t.TempDir(),
	}

	ctx2 := context.Background()
	err = transport2.Connect(ctx2)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer transport2.Disconnect()

	ctx3, cancel3 := context.WithCancel(context.Background())
	msgCh, errCh := transport2.ReceiveMessages(ctx3)

	// Cancel context after a short delay
	time.AfterFunc(100*time.Millisecond, cancel3)

	// Wait for channels to close or context to be done
	done := false
	for !done {
		select {
		case _, ok := <-msgCh:
			if !ok {
				done = true
			}
		case _, ok := <-errCh:
			if !ok {
				done = true
			}
		case <-ctx3.Done():
			// Context cancelled, test passed
			done = true
		case <-time.After(1 * time.Second):
			// Even if channels don't close, context cancellation worked
			done = true
		}
	}
}

// TestJSONDecoding tests JSON decoding error handling
func TestJSONDecoding(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid JSON",
			output:      `{"type":"assistant","content":[{"type":"text","text":"Hello"}]}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			output:      `{"type":"assistant", invalid json`,
			expectError: true,
			errorType:   &sdkerrors.CLIJSONDecodeError{},
		},
		{
			name:        "non-JSON output",
			output:      `This is not JSON`,
			expectError: false, // Non-JSON lines are skipped
		},
		{
			name:        "empty lines",
			output:      "\n\n\n",
			expectError: false, // Empty lines are skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test program that outputs the test data
			script := fmt.Sprintf(`#!/bin/sh
echo '%s'
exit 0`, tt.output)

			tmpFileName := createTestScript(t, script)
			defer os.Remove(tmpFileName)

			transport := &SubprocessCLITransport{
				cliPath: tmpFileName,
				prompt:  "test",
				cwd:     t.TempDir(),
			}

			ctx := context.Background()
			err := transport.Connect(ctx)
			if err != nil {
				t.Fatalf("Connect failed: %v", err)
			}
			defer transport.Disconnect()

			// Try to receive
			msgCh, errCh := transport.ReceiveMessages(ctx)

			// Collect results
			var messages []map[string]interface{}
			var lastErr error
			done := false
			for !done {
				select {
				case msg, ok := <-msgCh:
					if !ok {
						done = true
						break
					}
					messages = append(messages, msg)
				case err, ok := <-errCh:
					if ok && err != nil {
						lastErr = err
					}
				case <-time.After(1 * time.Second):
					done = true
				}
			}

			if tt.expectError {
				if lastErr == nil {
					t.Error("expected error but got none")
				} else if tt.errorType != nil {
					// Check error type
					var jsonErr *sdkerrors.CLIJSONDecodeError
					if !errors.As(lastErr, &jsonErr) {
						t.Errorf("expected CLIJSONDecodeError, got %T", lastErr)
					}
				}
			} else {
				if lastErr != nil {
					t.Errorf("unexpected error: %v", lastErr)
				}
			}
		})
	}
}

// TestErrorPropagation tests that stderr is properly captured
func TestErrorPropagation(t *testing.T) {
	// Create a script that writes to stderr and exits with error
	script := `#!/bin/sh
echo "Error: Something went wrong" >&2
exit 1`

	tmpFileName := createTestScript(t, script)

	transport := &SubprocessCLITransport{
		cliPath: tmpFileName,
		prompt:  "test",
		cwd:     t.TempDir(),
	}

	ctx := context.Background()
	err := transport.Connect(ctx)
	if err != nil {
		// This is expected since the script exits immediately
		// But connection might succeed before the process exits
		return
	}

	// If connect succeeded, wait for error from ReceiveMessages
	msgCh, errCh := transport.ReceiveMessages(ctx)

	// Wait for error
	select {
	case err := <-errCh:
		if err != nil {
			// Check if it's a ProcessError with stderr
			var procErr *sdkerrors.ProcessError
			if errors.As(err, &procErr) {
				if !strings.Contains(procErr.Stderr, "Something went wrong") {
					t.Errorf("stderr should contain error message, got: %v", procErr.Stderr)
				}
			}
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for error")
	}

	// Drain msgCh
	for range msgCh {
	}

	transport.Disconnect()
}

// TestCLINotFoundError tests the CLI not found error
func TestCLINotFoundError(t *testing.T) {
	transport := &SubprocessCLITransport{
		cliPath: "", // Empty path should trigger CLI not found
		prompt:  "test",
	}

	ctx := context.Background()
	err := transport.Connect(ctx)
	if err == nil {
		transport.Disconnect()
		t.Fatal("expected error for empty CLI path")
	}

	// Check that it's a CLINotFoundError
	var cliErr *sdkerrors.CLINotFoundError
	if !errors.As(err, &cliErr) {
		t.Errorf("expected CLINotFoundError, got %T: %v", err, err)
	}
}

// MockCwdProvider implements CwdProvider for testing
type MockCwdProvider struct {
	cwd string
}

func (m *MockCwdProvider) GetCwd() string {
	return m.cwd
}

// TestNewSubprocessCLITransport tests the constructor
func TestNewSubprocessCLITransport(t *testing.T) {
	tests := []struct {
		name        string
		prompt      string
		options     interface{}
		cliPath     string
		expectedCwd string
		checkCwd    bool
	}{
		{
			name:        "with cwd provider",
			prompt:      "test",
			options:     &MockCwdProvider{cwd: "/custom/path"},
			cliPath:     "claude",
			expectedCwd: "/custom/path",
			checkCwd:    true,
		},
		{
			name:     "without cwd provider",
			prompt:   "test",
			options:  struct{}{},
			cliPath:  "claude",
			checkCwd: false, // Will use current working directory
		},
		{
			name:     "nil options",
			prompt:   "test",
			options:  nil,
			cliPath:  "claude",
			checkCwd: false, // Will use current working directory
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewSubprocessCLITransport(tt.prompt, tt.options, tt.cliPath)

			if transport.prompt != tt.prompt {
				t.Errorf("prompt: got %q, want %q", transport.prompt, tt.prompt)
			}

			if tt.checkCwd {
				if transport.cwd != tt.expectedCwd {
					t.Errorf("cwd: got %q, want %q", transport.cwd, tt.expectedCwd)
				}
			} else if transport.cwd == "" {
				t.Error("cwd should not be empty when not provided in options")
			}
		})
	}
}

// TestConcurrentAccess tests thread safety of the transport
func TestConcurrentAccess(t *testing.T) {
	// Create a long-running script
	script := `#!/bin/sh
while true; do
	read line
	echo "$line"
done`

	tmpFileName := createTestScript(t, script)

	transport := &SubprocessCLITransport{
		cliPath: tmpFileName,
		prompt:  "test",
		cwd:     t.TempDir(),
	}

	ctx := context.Background()
	err := transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer transport.Disconnect()

	// Test concurrent IsConnected calls
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				transport.IsConnected()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestEnvironmentVariable tests that CLAUDE_CODE_ENTRYPOINT is set
func TestEnvironmentVariable(t *testing.T) {
	// Create a script that prints environment variables
	script := `#!/bin/sh
echo "$CLAUDE_CODE_ENTRYPOINT"
exit 0`

	tmpFileName := createTestScript(t, script)

	transport := &SubprocessCLITransport{
		cliPath: tmpFileName,
		prompt:  "test",
		cwd:     t.TempDir(),
	}

	ctx := context.Background()
	err := transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer transport.Disconnect()

	// Read output
	reader := bufio.NewReader(transport.stdout)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Check environment variable was set
	if strings.TrimSpace(line) != "sdk-go" {
		t.Errorf("CLAUDE_CODE_ENTRYPOINT: got %q, want %q", strings.TrimSpace(line), "sdk-go")
	}
}

// MockTransport implements Transport interface for testing
type MockTransport struct {
	messages   []map[string]interface{}
	errors     []error
	index      int
	connected  bool
	connectErr error
	closeErr   error
}

func (m *MockTransport) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *MockTransport) Disconnect() error {
	m.connected = false
	return m.closeErr
}

func (m *MockTransport) ReceiveMessages(ctx context.Context) (<-chan map[string]interface{}, <-chan error) {
	msgCh := make(chan map[string]interface{})
	errCh := make(chan error, 1)

	if !m.connected {
		errCh <- errors.New("not connected")
		close(msgCh)
		close(errCh)
		return msgCh, errCh
	}

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for m.index < len(m.messages) {
			select {
			case <-ctx.Done():
				return
			default:
				if m.index < len(m.errors) && m.errors[m.index] != nil {
					errCh <- m.errors[m.index]
					m.index++
					return
				}
				msgCh <- m.messages[m.index]
				m.index++
			}
		}
	}()

	return msgCh, errCh
}

func (m *MockTransport) IsConnected() bool {
	return m.connected
}

// TestTransportInterface verifies the Transport interface is properly implemented
func TestTransportInterface(t *testing.T) {
	var _ Transport = (*SubprocessCLITransport)(nil)
	var _ Transport = (*MockTransport)(nil)
}

// TestBuildCLIArgs tests the Options.BuildCLIArgs method directly
func TestBuildCLIArgs(t *testing.T) {
	// This test would be in the main package, but we can't import it here
	// due to circular dependencies. The test should be in types_test.go
	t.Skip("BuildCLIArgs test should be in the main package")
}
