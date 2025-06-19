package claudecode

import (
	"context"
	"testing"
	"time"
)

func TestQueryTimeout(t *testing.T) {
	tests := []struct {
		name           string
		options        *Options
		contextTimeout time.Duration
		expectTimeout  bool
	}{
		{
			name: "no timeout set",
			options: &Options{
				MaxThinkingTokens: 8000,
			},
			contextTimeout: 5 * time.Second,
			expectTimeout:  false,
		},
		{
			name: "query timeout set",
			options: &Options{
				MaxThinkingTokens: 8000,
				QueryTimeout:      1, // 1 second
			},
			contextTimeout: 5 * time.Second,
			expectTimeout:  true,
		},
		{
			name: "context timeout shorter than query timeout",
			options: &Options{
				MaxThinkingTokens: 8000,
				QueryTimeout:      10, // 10 seconds
			},
			contextTimeout: 1 * time.Second,
			expectTimeout:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test doesn't actually run a query since that would require
			// Claude Code CLI to be installed. Instead, we test the timeout logic.

			// Verify GetQueryTimeout works correctly
			timeout := tt.options.GetQueryTimeout()
			if tt.options.QueryTimeout > 0 {
				expectedTimeout := time.Duration(tt.options.QueryTimeout) * time.Second
				if timeout != expectedTimeout {
					t.Errorf("GetQueryTimeout() = %v, want %v", timeout, expectedTimeout)
				}
			} else if timeout != 0 {
				t.Errorf("GetQueryTimeout() = %v, want 0", timeout)
			}
		})
	}
}

func TestOptionsGetQueryTimeout(t *testing.T) {
	tests := []struct {
		name     string
		options  *Options
		expected time.Duration
	}{
		{
			name:     "nil options",
			options:  nil,
			expected: 0,
		},
		{
			name:     "zero timeout",
			options:  &Options{QueryTimeout: 0},
			expected: 0,
		},
		{
			name:     "negative timeout returns zero",
			options:  &Options{QueryTimeout: -5},
			expected: 0,
		},
		{
			name:     "1 second timeout",
			options:  &Options{QueryTimeout: 1},
			expected: 1 * time.Second,
		},
		{
			name:     "30 second timeout",
			options:  &Options{QueryTimeout: 30},
			expected: 30 * time.Second,
		},
		{
			name:     "5 minute timeout",
			options:  &Options{QueryTimeout: 300},
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.options.GetQueryTimeout()
			if got != tt.expected {
				t.Errorf("GetQueryTimeout() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTimeoutIntegration(t *testing.T) {
	// This test demonstrates how timeout would work in practice
	// It's skipped by default since it requires Claude Code CLI
	t.Skip("Integration test - requires Claude Code CLI")

	ctx := context.Background()

	options := &Options{
		QueryTimeout:      2, // 2 second timeout
		MaxThinkingTokens: 8000,
	}

	startTime := time.Now()
	msgCh, errCh := Query(ctx, "Count from 1 to 1000000 slowly", options)

	// Collect messages until timeout or completion
	timedOut := false
	for {
		select {
		case _, ok := <-msgCh:
			if !ok {
				goto done
			}
		case err := <-errCh:
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					timedOut = true
				}
				t.Logf("Error (possibly expected): %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Test timeout - query didn't complete or timeout within 5 seconds")
			goto done
		}
	}
done:

	elapsed := time.Since(startTime)
	t.Logf("Query took %v, timed out: %v", elapsed, timedOut)

	// If we set a 2-second timeout, it should complete within ~2-3 seconds
	if options.QueryTimeout > 0 && elapsed > time.Duration(options.QueryTimeout+1)*time.Second {
		t.Errorf("Query took %v, expected to timeout after ~%d seconds", elapsed, options.QueryTimeout)
	}
}
