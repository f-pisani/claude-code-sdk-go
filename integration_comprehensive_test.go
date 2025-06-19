//go:build integration
// +build integration

package claudecode

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestIntegrationBasicQuery tests a simple query
func TestIntegrationBasicQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgCh, errCh := Query(ctx, "What is 2 + 2?", nil)

	messages := collectMessages(t, ctx, msgCh, errCh)

	// Verify we got at least one assistant message
	foundAssistant := false
	for _, msg := range messages {
		if _, ok := msg.(AssistantMessage); ok {
			foundAssistant = true
			break
		}
	}

	if !foundAssistant {
		t.Error("Expected at least one assistant message")
	}
}

// TestIntegrationWithOptions tests queries with various options
func TestIntegrationWithOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name    string
		prompt  string
		options *Options
	}{
		{
			name:   "with system prompt",
			prompt: "Say hello",
			options: &Options{
				SystemPrompt:      "You are a friendly assistant who always responds with enthusiasm",
				MaxThinkingTokens: 8000,
			},
		},
		{
			name:   "with max turns limit",
			prompt: "Count from 1 to 10",
			options: &Options{
				MaxTurns:          IntPtr(1),
				MaxThinkingTokens: 8000,
			},
		},
		{
			name:   "with model selection",
			prompt: "What is the capital of France?",
			options: &Options{
				Model:             "claude-3-5-haiku-20241022",
				MaxThinkingTokens: 8000,
			},
		},
		{
			name:   "with permission mode",
			prompt: "Tell me about Go programming",
			options: &Options{
				PermissionMode:    (*PermissionMode)(StringPtr("bypassPermissions")),
				MaxThinkingTokens: 8000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			msgCh, errCh := Query(ctx, tt.prompt, tt.options)

			messages := collectMessages(t, ctx, msgCh, errCh)

			// Basic validation
			if len(messages) == 0 {
				t.Error("Expected at least one message")
			}

			// Log messages for debugging
			for _, msg := range messages {
				t.Logf("Message type: %T", msg)
			}
		})
	}
}

// TestIntegrationToolUse tests tool usage scenarios
func TestIntegrationToolUse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	options := &Options{
		AllowedTools:      []string{"Read", "Write", "Edit"},
		MaxThinkingTokens: 8000,
	}

	prompt := "Create a file called test.txt with the content 'Hello, World!'"
	msgCh, errCh := Query(ctx, prompt, options)

	messages := collectMessages(t, ctx, msgCh, errCh)

	// Look for tool use
	foundToolUse := false
	for _, msg := range messages {
		if am, ok := msg.(AssistantMessage); ok {
			for _, block := range am.Content {
				if _, ok := block.(ToolUseBlock); ok {
					foundToolUse = true
					break
				}
			}
		}
	}

	if !foundToolUse {
		t.Log("Warning: No tool use detected (this might be expected depending on the model's behavior)")
	}
}

// TestIntegrationErrorHandling tests error scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		prompt      string
		options     *Options
		expectError bool
	}{
		{
			name:   "invalid model",
			prompt: "Hello",
			options: &Options{
				Model:             "invalid-model-name",
				MaxThinkingTokens: 8000,
			},
			expectError: true,
		},
		{
			name:   "invalid max turns",
			prompt: "Hello",
			options: &Options{
				MaxTurns:          IntPtr(-1),
				MaxThinkingTokens: 8000,
			},
			expectError: true,
		},
		{
			name:   "invalid thinking tokens",
			prompt: "Hello",
			options: &Options{
				MaxThinkingTokens: -1000,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			msgCh, errCh := Query(ctx, tt.prompt, tt.options)

			gotError := false
			for {
				select {
				case _, ok := <-msgCh:
					if !ok {
						goto done
					}
				case err := <-errCh:
					if err != nil {
						gotError = true
						t.Logf("Got expected error: %v", err)
					}
				case <-ctx.Done():
					goto done
				}
			}
		done:

			if tt.expectError && !gotError {
				t.Error("Expected error but got none")
			}
		})
	}
}

// TestIntegrationCancellation tests context cancellation
func TestIntegrationCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())

	msgCh, errCh := Query(ctx, "Count from 1 to 1000000 slowly", nil)

	// Cancel after a short delay
	go func() {
		time.Sleep(2 * time.Second)
		cancel()
	}()

	// Collect messages until channels close
	messageCount := 0
	for {
		select {
		case _, ok := <-msgCh:
			if !ok {
				goto done
			}
			messageCount++
		case err := <-errCh:
			if err != nil {
				t.Logf("Error (might be expected): %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for channels to close after cancellation")
			goto done
		}
	}
done:

	t.Logf("Received %d messages before cancellation", messageCount)
}

// TestIntegrationMessageTypes tests different message types
func TestIntegrationMessageTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgCh, errCh := Query(ctx, "Hello, please respond with a simple greeting", nil)

	messageTypes := make(map[string]int)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				goto done
			}

			switch msg.(type) {
			case UserMessage:
				messageTypes["user"]++
			case AssistantMessage:
				messageTypes["assistant"]++
			case SystemMessage:
				messageTypes["system"]++
			case ResultMessage:
				messageTypes["result"]++
			default:
				messageTypes["unknown"]++
			}

		case err := <-errCh:
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}
done:

	// Log message type counts
	for msgType, count := range messageTypes {
		t.Logf("%s messages: %d", msgType, count)
	}

	// Verify we got expected message types
	if messageTypes["assistant"] == 0 {
		t.Error("Expected at least one assistant message")
	}
	if messageTypes["result"] == 0 {
		t.Error("Expected at least one result message")
	}
}

// TestIntegrationLongConversation tests a longer conversation
func TestIntegrationLongConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test would require conversation continuation support
	// For now, we'll test a single longer prompt

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	prompt := `Please do the following:
1. Explain what a fibonacci sequence is
2. Write a function to calculate the nth fibonacci number
3. Calculate the 10th fibonacci number`

	msgCh, errCh := Query(ctx, prompt, nil)

	messages := collectMessages(t, ctx, msgCh, errCh)

	// Look for code blocks or explanations
	foundExplanation := false
	foundCode := false

	for _, msg := range messages {
		if am, ok := msg.(AssistantMessage); ok {
			for _, block := range am.Content {
				if tb, ok := block.(TextBlock); ok {
					text := strings.ToLower(tb.Text)
					if strings.Contains(text, "fibonacci") {
						foundExplanation = true
					}
					if strings.Contains(text, "func") || strings.Contains(text, "def") {
						foundCode = true
					}
				}
			}
		}
	}

	if !foundExplanation {
		t.Error("Expected explanation of fibonacci sequence")
	}

	t.Logf("Found explanation: %v, Found code: %v", foundExplanation, foundCode)
}

// TestIntegrationResultMessage tests result message details
func TestIntegrationResultMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgCh, errCh := Query(ctx, "What is 2 + 2?", nil)

	var resultMsg *ResultMessage

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				goto done
			}

			if rm, ok := msg.(ResultMessage); ok {
				resultMsg = &rm
			}

		case err := <-errCh:
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}
done:

	if resultMsg == nil {
		t.Fatal("Expected a result message")
	}

	// Verify result message fields
	if resultMsg.DurationMs <= 0 {
		t.Error("Expected positive duration")
	}

	if resultMsg.SessionID == "" {
		t.Error("Expected non-empty session ID")
	}

	t.Logf("Result message: subtype=%s, duration=%dms, turns=%d, session=%s",
		resultMsg.Subtype, resultMsg.DurationMs, resultMsg.NumTurns, resultMsg.SessionID)

	if resultMsg.Usage != nil {
		t.Logf("Usage: %+v", resultMsg.Usage)
	}

	if resultMsg.TotalCostUSD != nil {
		t.Logf("Cost: $%.4f", *resultMsg.TotalCostUSD)
	}
}

// Helper function to collect all messages from channels
func collectMessages(t *testing.T, ctx context.Context, msgCh <-chan Message, errCh <-chan error) []Message {
	var messages []Message

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return messages
			}
			messages = append(messages, msg)

		case err := <-errCh:
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

		case <-ctx.Done():
			t.Fatal("Context cancelled while collecting messages")
		}
	}
}

// TestIntegrationParallelQueries tests multiple concurrent queries
func TestIntegrationParallelQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	queries := []string{
		"What is 1 + 1?",
		"What is 2 + 2?",
		"What is 3 + 3?",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	type result struct {
		query    string
		messages []Message
		err      error
	}

	results := make(chan result, len(queries))

	// Launch parallel queries
	for _, query := range queries {
		go func(q string) {
			msgCh, errCh := Query(ctx, q, nil)

			var messages []Message
			var queryErr error

			for {
				select {
				case msg, ok := <-msgCh:
					if !ok {
						results <- result{query: q, messages: messages, err: queryErr}
						return
					}
					messages = append(messages, msg)

				case err := <-errCh:
					if err != nil {
						queryErr = err
					}

				case <-ctx.Done():
					results <- result{query: q, messages: messages, err: fmt.Errorf("context cancelled")}
					return
				}
			}
		}(query)
	}

	// Collect results
	for i := 0; i < len(queries); i++ {
		select {
		case r := <-results:
			if r.err != nil {
				t.Errorf("Query %q failed: %v", r.query, r.err)
			} else {
				t.Logf("Query %q completed with %d messages", r.query, len(r.messages))
			}
		case <-ctx.Done():
			t.Fatal("Timeout waiting for parallel queries to complete")
		}
	}
}
