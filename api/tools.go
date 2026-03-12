package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// extractText returns the concatenated text from all text blocks in a response.
func extractText(resp apiResponse) (string, error) {
	var parts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return "", errors.New("no text content in response")
	}
	return strings.Join(parts, "\n"), nil
}

// collectToolUseBlocks returns all tool_use content blocks from resp.Content.
func collectToolUseBlocks(content []ContentBlock) []ContentBlock {
	var blocks []ContentBlock
	for _, b := range content {
		if b.Type == "tool_use" {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

// runToolCalls appends the assistant message, executes every tool call, fires
// onCall callbacks, and appends the tool_result user message. Returns updated msgs.
func runToolCalls(ctx context.Context, msgs []Message, respContent []ContentBlock, toolUseBlocks []ContentBlock, executor ToolExecutor, onCall []func(ToolCallResult)) []Message {
	msgs = append(msgs, Message{Role: "assistant", Content: respContent})
	toolResults := make([]ContentBlock, 0, len(toolUseBlocks))
	for _, tu := range toolUseBlocks {
		output, execErr := executor.ExecuteTool(ctx, tu.Name, tu.Input)
		content := output
		if execErr != nil && content == "" {
			content = fmt.Sprintf("error: %v", execErr)
		}
		toolResults = append(toolResults, ContentBlock{
			Type:      "tool_result",
			ToolUseID: tu.ID,
			Content:   content,
		})
		for _, fn := range onCall {
			fn(ToolCallResult{Name: tu.Name, Input: tu.Input, Output: output, Err: execErr})
		}
	}
	return append(msgs, Message{Role: "user", Content: toolResults})
}

// runToolLoop drives the agentic tool-calling loop. fetch is called for each turn;
// tool results are fed back until the model returns a final text-only response.
func runToolLoop(
	ctx context.Context,
	msgs []Message,
	tools []Tool,
	executor ToolExecutor,
	onCall []func(ToolCallResult),
	fetch func(msgs []Message, tools []Tool) (apiResponse, error),
) (string, []Message, Usage, error) {
	var totalUsage Usage
	for {
		resp, err := fetch(msgs, tools)
		if err != nil {
			return "", nil, Usage{}, err
		}
		totalUsage = totalUsage.Add(Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		})

		toolUseBlocks := collectToolUseBlocks(resp.Content)
		if len(toolUseBlocks) == 0 {
			text, err := extractText(resp)
			if err != nil {
				return "", nil, Usage{}, err
			}
			return text, append(msgs, Message{Role: "assistant", Content: text}), totalUsage, nil
		}

		msgs = runToolCalls(ctx, msgs, resp.Content, toolUseBlocks, executor, onCall)
	}
}
