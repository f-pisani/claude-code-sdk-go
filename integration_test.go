//go:build integration
// +build integration

package claudecode

import (
	"context"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	t.Skip("Integration test - run manually with -tags=integration")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgCh, errCh := Query(ctx, "What is 2 + 2?", nil)

	gotResponse := false
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				if !gotResponse {
					t.Error("Channel closed without receiving response")
				}
				return
			}

			switch m := msg.(type) {
			case AssistantMessage:
				gotResponse = true
				t.Logf("Assistant: %+v", m)
				for _, block := range m.Content {
					if tb, ok := block.(TextBlock); ok {
						t.Logf("Text: %s", tb.Text)
					}
				}
			case ResultMessage:
				t.Logf("Result: %+v", m)
			}

		case err := <-errCh:
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}
}
