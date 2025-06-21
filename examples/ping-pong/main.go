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

func main() {
	// Create two Claude instances with different personas
	player1Options := claudecode.NewOptions()
	player1Options.SystemPrompt = "You are Player 1 in a ping-pong conversation. When you receive 'Ping', respond with 'Pong'. When you receive 'Pong', respond with 'Ping'. Keep your responses simple and include your player number."
	player1Options.MaxTurns = claudecode.IntPtr(1)

	player2Options := claudecode.NewOptions()
	player2Options.SystemPrompt = "You are Player 2 in a ping-pong conversation. When you receive 'Ping', respond with 'Pong'. When you receive 'Pong', respond with 'Ping'. Keep your responses simple and include your player number."
	player2Options.MaxTurns = claudecode.IntPtr(1)

	ctx := context.Background()

	// Channels to pass messages between players
	player1To2 := make(chan string, 1)
	player2To1 := make(chan string, 1)

	// Done channel to signal completion
	done := make(chan bool)

	// WaitGroup to ensure both players finish
	var wg sync.WaitGroup
	wg.Add(2)

	// Counter for the number of exchanges
	maxExchanges := 10

	// Player 1
	go func() {
		defer wg.Done()
		exchanges := 0

		// Start the game by sending "Ping" to Player 2
		player1To2 <- "Ping"
		fmt.Println("Player 1 starts with: Ping")

		for exchanges < maxExchanges/2 {
			// Wait for message from Player 2
			select {
			case msg := <-player2To1:
				fmt.Printf("Player 1 received: %s\n", msg)

				// Query Claude as Player 1
				msgCh, errCh := claudecode.Query(ctx, msg, player1Options)

				responseReceived := false
				for !responseReceived {
					select {
					case msg, ok := <-msgCh:
						if !ok {
							return
						}
						if assistantMsg, ok := msg.(claudecode.AssistantMessage); ok {
							for _, block := range assistantMsg.Content {
								if textBlock, ok := block.(claudecode.TextBlock); ok {
									response := strings.TrimSpace(textBlock.Text)
									fmt.Printf("Player 1 says: %s\n", response)

									// Send response to Player 2
									player1To2 <- response
									exchanges++
									responseReceived = true

									// Small delay for readability
									time.Sleep(500 * time.Millisecond)
								}
							}
						}
					case err := <-errCh:
						if err != nil {
							log.Printf("Player 1 error: %v\n", err)
							close(done)
							return
						}
					}
				}
			case <-done:
				return
			}
		}

		// Signal completion
		close(done)
	}()

	// Player 2
	go func() {
		defer wg.Done()
		exchanges := 0

		for exchanges < maxExchanges/2 {
			// Wait for message from Player 1
			select {
			case msg := <-player1To2:
				fmt.Printf("Player 2 received: %s\n", msg)

				// Query Claude as Player 2
				msgCh, errCh := claudecode.Query(ctx, msg, player2Options)

				responseReceived := false
				for !responseReceived {
					select {
					case msg, ok := <-msgCh:
						if !ok {
							return
						}
						if assistantMsg, ok := msg.(claudecode.AssistantMessage); ok {
							for _, block := range assistantMsg.Content {
								if textBlock, ok := block.(claudecode.TextBlock); ok {
									response := strings.TrimSpace(textBlock.Text)
									fmt.Printf("Player 2 says: %s\n", response)

									// Send response to Player 1
									player2To1 <- response
									exchanges++
									responseReceived = true

									// Small delay for readability
									time.Sleep(500 * time.Millisecond)
								}
							}
						}
					case err := <-errCh:
						if err != nil {
							log.Printf("Player 2 error: %v\n", err)
							close(done)
							return
						}
					}
				}
			case <-done:
				return
			}
		}
	}()

	// Wait for both players to finish
	wg.Wait()
	fmt.Printf("\nGame ended after %d exchanges!\n", maxExchanges)
}
