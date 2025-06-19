# Claude Code SDK for Go

Go SDK for interacting with Claude Code, Anthropic's AI coding assistant.

> This Go SDK is a reimplementation of the original [Python SDK by Anthropic, PBC](https://github.com/anthropics/claude-code-sdk-python), and is based on its publicly available source code and design.
>
> All Go code was written independently, based on the MIT-licensed Python SDK.

## Installation

```bash
go get github.com/f-pisani/claude-code-sdk-go
```

## Prerequisites

- Go 1.21 or later
- Claude Code CLI installed (`npm install -g @anthropic-ai/claude-code`)
- Valid Anthropic API key configured

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    claudecode "github.com/f-pisani/claude-code-sdk-go"
)

func main() {
    ctx := context.Background()
    msgCh, errCh := claudecode.Query(ctx, "What is 2 + 2?", nil)
    
    for {
        select {
        case msg, ok := <-msgCh:
            if !ok {
                return // Done
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
                log.Fatal(err)
            }
        }
    }
}
```

## Usage with Options

```go
options := claudecode.NewOptions()
options.SystemPrompt = "You are a helpful assistant"
options.AllowedTools = []string{"Read", "Write"}
options.Cwd = "/path/to/project"

// Control tool permissions
mode := claudecode.PermissionModeAcceptEdits
options.PermissionMode = &mode

msgCh, errCh := claudecode.Query(ctx, "Help me with my code", options)
```

## API Reference

### Main Function

#### `Query(ctx context.Context, prompt string, options *Options) (<-chan Message, <-chan error)`

Sends a prompt to Claude Code and returns channels for messages and errors.

**Parameters:**
- `ctx`: Context for cancellation
- `prompt`: The prompt to send to Claude
- `options`: Configuration options (optional, use `nil` for defaults)

**Returns:**
- `msgCh`: Channel yielding messages from the conversation
- `errCh`: Buffered error channel (receives at most one error)

### Types

#### Message Types
- `UserMessage`: Message from the user
- `AssistantMessage`: Message from Claude with content blocks
- `SystemMessage`: System message with metadata
- `ResultMessage`: Final result with cost and usage information

#### Content Block Types
- `TextBlock`: Plain text content
- `ToolUseBlock`: Tool invocation
- `ToolResultBlock`: Tool execution result

#### Options
- `AllowedTools`: List of allowed tool names
- `DisallowedTools`: List of disallowed tool names
- `SystemPrompt`: System prompt to prepend
- `PermissionMode`: Tool permission mode ("default", "acceptEdits", "bypassPermissions")
- `MaxTurns`: Maximum conversation turns
- `Model`: Model to use
- `Cwd`: Working directory

### Error Types
- `SDKError`: Base error type
- `CLIConnectionError`: Connection issues
- `CLINotFoundError`: Claude Code CLI not found
- `ProcessError`: CLI process failures
- `CLIJSONDecodeError`: JSON parsing errors

## Examples

See the [examples](examples/) directory for more detailed examples:
- [quickstart](examples/quickstart/main.go) - Basic usage examples

## Architecture

This SDK communicates with Claude Code through the CLI subprocess, using JSON streaming for message exchange. It follows the same architecture as the Python SDK with Go idioms:

- Channels instead of async generators
- Context-based cancellation
- Strong typing with interfaces
- Concurrent-safe design

## License

This project is licensed under the MIT License.