package transport

import (
	"context"
)

// Transport is the abstract interface for Claude communication
type Transport interface {
	// Connect initializes the connection
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect() error

	// ReceiveMessages returns a channel that yields messages from Claude
	ReceiveMessages(ctx context.Context) (<-chan map[string]interface{}, <-chan error)

	// IsConnected checks if transport is connected
	IsConnected() bool
}