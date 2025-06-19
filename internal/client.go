package internal

import (
	"context"
	"fmt"

	"github.com/f-pisani/claude-code-sdk-go/internal/transport"
)

// Client handles internal query processing
type Client struct{}

// NewClient creates a new internal client
func NewClient() *Client {
	return &Client{}
}

// ProcessQuery processes a query through the transport
func (c *Client) ProcessQuery(ctx context.Context, prompt string, options interface{}) (<-chan interface{}, <-chan error) {
	// Get buffer sizes from options if available
	msgBufSize := 10
	errBufSize := 1

	// Check if options has buffer size methods
	if opt, ok := options.(interface {
		GetMessageBufferSize() int
		GetErrorBufferSize() int
	}); ok {
		msgBufSize = opt.GetMessageBufferSize()
		errBufSize = opt.GetErrorBufferSize()
	}

	// Create channels with configurable buffer sizes
	msgCh := make(chan interface{}, msgBufSize)
	errCh := make(chan error, errBufSize)

	go func() {
		// Add panic recovery to ensure channels are always closed
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic in ProcessQuery: %v", r)
			}
			close(msgCh)
			close(errCh)
		}()

		// Create transport
		trans := transport.NewSubprocessCLITransport(prompt, options, "")

		// Connect
		if err := trans.Connect(ctx); err != nil {
			errCh <- err
			return
		}
		defer trans.Disconnect()

		// Receive messages
		dataCh, dataErrCh := trans.ReceiveMessages(ctx)

		for {
			select {
			case data, ok := <-dataCh:
				if !ok {
					return
				}
				if msg := c.parseMessage(data); msg != nil {
					select {
					case msgCh <- msg:
					case <-ctx.Done():
						return
					}
				}
			case err, ok := <-dataErrCh:
				if !ok {
					// Error channel closed
					return
				}
				if err != nil {
					// Try to send error without blocking
					select {
					case errCh <- err:
					case <-ctx.Done():
						return
					default:
						// Error channel full, replace with latest error
						select {
						case <-errCh:
							errCh <- err
						default:
						}
					}
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgCh, errCh
}

// parseMessage parses a message from CLI output and returns a map
func (c *Client) parseMessage(data map[string]interface{}) interface{} {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil
	}

	switch msgType {
	case "user":
		if msgData, ok := data["message"].(map[string]interface{}); ok {
			if content, ok := msgData["content"].(string); ok {
				return map[string]interface{}{"_type": "user", "content": content}
			}
		}

	case "assistant":
		if msgData, ok := data["message"].(map[string]interface{}); ok {
			if contentData, ok := msgData["content"].([]interface{}); ok {
				var contentBlocks []interface{}
				for _, blockData := range contentData {
					if blockMap, ok := blockData.(map[string]interface{}); ok {
						if block := c.parseContentBlock(blockMap); block != nil {
							contentBlocks = append(contentBlocks, block)
						}
					}
				}
				return map[string]interface{}{"_type": "assistant", "content": contentBlocks}
			}
		}

	case "system":
		subtype, _ := data["subtype"].(string)
		return map[string]interface{}{"_type": "system", "subtype": subtype, "data": data}

	case "result":
		subtype, _ := data["subtype"].(string)
		durationMs, _ := data["duration_ms"].(float64)
		durationAPIMs, _ := data["duration_api_ms"].(float64)
		isError, _ := data["is_error"].(bool)
		numTurns, _ := data["num_turns"].(float64)
		sessionID, _ := data["session_id"].(string)

		msg := map[string]interface{}{
			"_type":           "result",
			"subtype":         subtype,
			"duration_ms":     int(durationMs),
			"duration_api_ms": int(durationAPIMs),
			"is_error":        isError,
			"num_turns":       int(numTurns),
			"session_id":      sessionID,
		}

		if totalCostUSD, ok := data["total_cost_usd"].(float64); ok {
			msg["total_cost_usd"] = totalCostUSD
		}
		if usage, ok := data["usage"].(map[string]interface{}); ok {
			msg["usage"] = usage
		}
		if result, ok := data["result"].(string); ok {
			msg["result"] = result
		}

		return msg
	}

	return nil
}

// parseContentBlock parses a content block from data
func (c *Client) parseContentBlock(data map[string]interface{}) interface{} {
	blockType, ok := data["type"].(string)
	if !ok {
		return nil
	}

	switch blockType {
	case "text":
		if text, ok := data["text"].(string); ok {
			return map[string]interface{}{"_blockType": "text", "text": text}
		}

	case "tool_use":
		id, _ := data["id"].(string)
		name, _ := data["name"].(string)
		input, _ := data["input"].(map[string]interface{})
		return map[string]interface{}{"_blockType": "tool_use", "id": id, "name": name, "input": input}

	case "tool_result":
		toolUseID, _ := data["tool_use_id"].(string)
		block := map[string]interface{}{"_blockType": "tool_result", "tool_use_id": toolUseID}
		if content, ok := data["content"]; ok {
			block["content"] = content
		}
		if isError, ok := data["is_error"].(bool); ok {
			block["is_error"] = isError
		}
		return block
	}

	return nil
}
