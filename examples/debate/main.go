package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	claudecode "github.com/f-pisani/claude-code-sdk-go"
)

// Debater represents a participant in the debate
type Debater struct {
	name      string
	emoji     string
	sessionID string
	options   *claudecode.Options
	mu        sync.Mutex
}

// respond generates a response to the opponent's statement
func (d *Debater) respond(ctx context.Context, statement string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Use Resume if we have a session ID from a previous turn
	if d.sessionID != "" {
		d.options.Resume = d.sessionID
	}

	msgCh, errCh := claudecode.Query(ctx, statement, d.options)

	var response string
	var newSessionID string

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				if response == "" {
					return "", fmt.Errorf("%s: channel closed without response", d.name)
				}
				// Update session ID for next turn
				if newSessionID != "" {
					d.sessionID = newSessionID
				}
				return response, nil
			}

			switch m := msg.(type) {
			case claudecode.AssistantMessage:
				for _, block := range m.Content {
					if textBlock, ok := block.(claudecode.TextBlock); ok {
						response = strings.TrimSpace(textBlock.Text)
					}
				}
			case claudecode.ResultMessage:
				if m.SessionID != "" {
					newSessionID = m.SessionID
				}
			}

		case err := <-errCh:
			if err != nil {
				return "", fmt.Errorf("%s error: %w", d.name, err)
			}
		case <-ctx.Done():
			return "", fmt.Errorf("%s: context cancelled", d.name)
		}
	}
}

func main() {
	ctx := context.Background()

	// Create optimist debater
	optimist := &Debater{
		name:    "Optimist",
		emoji:   "🔵",
		options: claudecode.NewOptions(),
	}
	optimist.options.SystemPrompt = `You are an AI optimist in a debate about whether AI will replace software developers. 
You believe AI will augment rather than replace developers. Present thoughtful, nuanced arguments about:
- How AI tools enhance developer productivity
- The irreplaceable human elements in software development
- Historical parallels with other technological advances
Keep responses concise (2-3 sentences) and directly address your opponent's points.`
	optimist.options.MaxTurns = claudecode.IntPtr(1)

	// Create pessimist debater
	pessimist := &Debater{
		name:    "Pessimist",
		emoji:   "🔴",
		options: claudecode.NewOptions(),
	}
	pessimist.options.SystemPrompt = `You are an AI pessimist in a debate about whether AI will replace software developers.
You believe AI will eventually replace most developer jobs. Present thoughtful, nuanced arguments about:
- Rapid AI capabilities growth
- Economic incentives for automation
- Examples of AI already handling complex programming tasks
Keep responses concise (2-3 sentences) and directly address your opponent's points.`
	pessimist.options.MaxTurns = claudecode.IntPtr(1)

	// Number of debate rounds
	maxRounds := 40

	fmt.Println("=== AI Debate: Will AI Replace Software Developers? ===")
	fmt.Println()

	// Start the debate
	openingStatement := "AI will enhance developers, not replace them. Just as IDEs and compilers didn't eliminate programmers but made them more productive, AI tools will handle routine tasks while developers focus on architecture, problem-solving, and understanding business needs."
	fmt.Printf("%s %s: %s\n\n", optimist.emoji, optimist.name, openingStatement)

	currentStatement := openingStatement
	var currentDebater *Debater

	// Alternate between debaters
	for round := 0; round < maxRounds; round++ {
		time.Sleep(1 * time.Second) // Pause for readability

		// Determine who responds
		if round%2 == 0 {
			currentDebater = pessimist
		} else {
			currentDebater = optimist
		}

		// Generate response
		response, err := currentDebater.respond(ctx, currentStatement)
		if err != nil {
			log.Fatalf("Failed to get response: %v", err)
		}

		fmt.Printf("%s %s: %s\n\n", currentDebater.emoji, currentDebater.name, response)
		currentStatement = response
	}

	fmt.Printf("=== Debate concluded after %d exchanges ===\n", maxRounds)
}
