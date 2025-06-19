package main

import (
	"context"
	"fmt"
	"log"

	claudecode "github.com/f-pisani/claude-code-sdk-go"
)

func main() {
	// Run all examples
	basicExample()
	withOptionsExample()
	withToolsExample()
}

// basicExample demonstrates a simple question
func basicExample() {
	fmt.Println("=== Basic Example ===")

	ctx := context.Background()
	msgCh, errCh := claudecode.Query(ctx, "What is 2 + 2?", nil)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				fmt.Println()
				return
			}
			if assistantMsg, ok := msg.(claudecode.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claudecode.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		case err := <-errCh:
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
	}
}

// withOptionsExample demonstrates usage with custom options
func withOptionsExample() {
	fmt.Println("=== With Options Example ===")

	options := claudecode.NewOptions()
	options.SystemPrompt = "You are a helpful assistant that explains things simply."
	maxTurns := 1
	options.MaxTurns = &maxTurns

	ctx := context.Background()
	msgCh, errCh := claudecode.Query(ctx, "Explain what Go is in one sentence.", options)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				fmt.Println()
				return
			}
			if assistantMsg, ok := msg.(claudecode.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(claudecode.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			}
		case err := <-errCh:
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
	}
}

// withToolsExample demonstrates usage with tools
func withToolsExample() {
	fmt.Println("=== With Tools Example ===")

	options := claudecode.NewOptions()
	options.AllowedTools = []string{"Read", "Write"}
	options.SystemPrompt = "You are a helpful file assistant."

	ctx := context.Background()
	msgCh, errCh := claudecode.Query(ctx, "Create a file called hello.txt with 'Hello, World!' in it", options)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				fmt.Println()
				return
			}
			switch m := msg.(type) {
			case claudecode.AssistantMessage:
				for _, block := range m.Content {
					if textBlock, ok := block.(claudecode.TextBlock); ok {
						fmt.Printf("Claude: %s\n", textBlock.Text)
					}
				}
			case claudecode.ResultMessage:
				// Use safe helper to avoid nil pointer dereference
				cost := claudecode.SafeFloat64Ptr(m.TotalCostUSD)
				if cost > 0 {
					fmt.Printf("\nCost: $%.4f\n", cost)
				}
			}
		case err := <-errCh:
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
		}
	}
}
