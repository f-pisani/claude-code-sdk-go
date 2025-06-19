package claudecode

import (
	"context"
	"fmt"

	"github.com/f-pisani/claude-code-sdk-go/internal"
)

// Query sends a prompt to Claude Code and returns channels for messages and errors.
//
// Go SDK for interacting with Claude Code.
//
// Parameters:
//   - ctx: Context for cancellation
//   - prompt: The prompt to send to Claude
//   - options: Optional configuration (uses NewOptions() if nil).
//     Set options.PermissionMode to control tool execution:
//   - 'default': CLI prompts for dangerous tools
//   - 'acceptEdits': Auto-accept file edits
//   - 'bypassPermissions': Allow all tools (use with caution)
//     Set options.Cwd for working directory.
//
// Returns:
//   - msgCh: Channel that yields messages from the conversation
//   - errCh: Channel for errors (buffered, receives at most one error)
//
// Example:
//
//	// Simple usage
//	msgCh, errCh := Query(context.Background(), "Hello", nil)
//	for {
//	    select {
//	    case msg, ok := <-msgCh:
//	        if !ok {
//	            return // Channel closed, done
//	        }
//	        fmt.Printf("%+v\n", msg)
//	    case err := <-errCh:
//	        if err != nil {
//	            log.Fatal(err)
//	        }
//	    }
//	}
//
//	// With options
//	options := NewOptions()
//	options.SystemPrompt = "You are helpful"
//	options.Cwd = "/home/user"
//	msgCh, errCh := Query(context.Background(), "Hello", options)
func Query(ctx context.Context, prompt string, options *Options) (<-chan Message, <-chan error) {
	if options == nil {
		options = NewOptions()
	}

	// Apply query timeout if specified
	queryCtx := ctx
	var cancel context.CancelFunc
	if timeout := options.GetQueryTimeout(); timeout > 0 {
		queryCtx, cancel = context.WithTimeout(ctx, timeout)
	}

	client := internal.NewClient()

	// Get raw channels from internal client
	rawMsgCh, rawErrCh := client.ProcessQuery(queryCtx, prompt, options)

	// Create typed channels with configurable buffer sizes
	msgCh := make(chan Message, options.GetMessageBufferSize())
	errCh := make(chan error, options.GetErrorBufferSize())

	// Convert raw messages to typed messages
	go func() {
		// Add panic recovery to ensure channels are always closed
		defer func() {
			if r := recover(); r != nil {
				// Try to send panic error, but don't block
				select {
				case errCh <- fmt.Errorf("panic in message conversion: %v", r):
				default:
				}
			}
			close(msgCh)
			close(errCh)
			// Cancel timeout if it was set
			if cancel != nil {
				cancel()
			}
		}()

		for {
			select {
			case rawMsg, ok := <-rawMsgCh:
				if !ok {
					return
				}
				if msg := convertMessage(rawMsg); msg != nil {
					select {
					case msgCh <- msg:
					case <-queryCtx.Done():
						return
					}
				}
			case err, ok := <-rawErrCh:
				if !ok {
					// Error channel closed, we're done
					return
				}
				if err != nil {
					// Try to send error without blocking
					select {
					case errCh <- err:
					case <-queryCtx.Done():
						return
					default:
						// Error channel full, prioritize most recent error
						select {
						case <-errCh:
							errCh <- err
						default:
						}
					}
					return
				}
			case <-queryCtx.Done():
				return
			}
		}
	}()

	return msgCh, errCh
}

// convertMessage converts raw message map to typed Message
func convertMessage(raw interface{}) Message {
	data, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}

	msgType, ok := data["_type"].(string)
	if !ok {
		return nil
	}

	switch msgType {
	case "user":
		if content, ok := data["content"].(string); ok {
			return UserMessage{Content: content}
		}

	case "assistant":
		if contentData, ok := data["content"].([]interface{}); ok {
			var contentBlocks []ContentBlock
			for _, blockData := range contentData {
				if block := convertContentBlock(blockData); block != nil {
					contentBlocks = append(contentBlocks, block)
				}
			}
			return AssistantMessage{Content: contentBlocks}
		}

	case "system":
		subtype, _ := data["subtype"].(string)
		systemData, _ := data["data"].(map[string]interface{})
		return SystemMessage{
			Subtype: subtype,
			Data:    systemData,
		}

	case "result":
		msg := ResultMessage{
			Subtype:       getString(data, "subtype"),
			DurationMs:    getInt(data, "duration_ms"),
			DurationAPIMs: getInt(data, "duration_api_ms"),
			IsError:       getBool(data, "is_error"),
			NumTurns:      getInt(data, "num_turns"),
			SessionID:     getString(data, "session_id"),
		}

		if totalCostUSD, ok := data["total_cost_usd"].(float64); ok {
			msg.TotalCostUSD = &totalCostUSD
		}
		if usage, ok := data["usage"].(map[string]interface{}); ok {
			msg.Usage = usage
		}
		if result, ok := data["result"].(string); ok {
			msg.Result = &result
		}

		return msg
	}

	return nil
}

// convertContentBlock converts raw content block to typed ContentBlock
func convertContentBlock(raw interface{}) ContentBlock {
	data, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}

	blockType, ok := data["_blockType"].(string)
	if !ok {
		return nil
	}

	switch blockType {
	case "text":
		if text, ok := data["text"].(string); ok {
			return TextBlock{Text: text}
		}

	case "tool_use":
		return ToolUseBlock{
			ID:    getString(data, "id"),
			Name:  getString(data, "name"),
			Input: getMap(data, "input"),
		}

	case "tool_result":
		block := ToolResultBlock{
			ToolUseID: getString(data, "tool_use_id"),
		}
		if content, ok := data["content"]; ok {
			block.Content = content
		}
		if isError, ok := data["is_error"].(bool); ok {
			block.IsError = &isError
		}
		return block
	}

	return nil
}

// Helper functions for type conversions
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key].(int); ok {
		return val
	}
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return false
}

func getMap(data map[string]interface{}, key string) map[string]interface{} {
	if val, ok := data[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}
