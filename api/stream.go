package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

// SendMessageWithToolsStreaming is like SendMessageWithTools but streams text tokens
// via the SSE API. onToken is called for each text token as it arrives; it may be nil.
// Tool-calling turns are handled the same way as in SendMessageWithTools.
func (c *Client) SendMessageWithToolsStreaming(ctx context.Context, messages []Message, tools []Tool, executor ToolExecutor, onToken func(string), onCall ...func(ToolCallResult)) (string, []Message, Usage, error) {
	if len(messages) == 0 {
		return "", nil, Usage{}, errors.New("no messages provided")
	}
	sdkTools := toSDKTools(tools)
	msgs := make([]Message, len(messages))
	copy(msgs, messages)
	return runToolLoop(ctx, msgs, tools, executor, onCall, func(loopMsgs []Message, _ []Tool) (apiResponse, error) {
		sdkMsgs, err := toSDKMessages(loopMsgs)
		if err != nil {
			return apiResponse{}, err
		}
		return c.doStreamRequest(ctx, sdkMsgs, sdkTools, onToken)
	})
}

// doStreamRequest opens a streaming SSE connection and accumulates the full response.
// onToken is called for each text token as it arrives; it may be nil.
// Retries are handled by the SDK before streaming begins; once a 200 OK is received
// and streaming starts, the result is returned directly to prevent duplicate tokens.
func (c *Client) doStreamRequest(ctx context.Context, messages []anthropic.MessageParam, tools []anthropic.ToolUnionParam, onToken func(string)) (apiResponse, error) {
	stream := c.sdkClient.Messages.NewStreaming(ctx, c.newParams(messages, tools), c.requestOpts(ctx)...)

	var msg anthropic.Message
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return apiResponse{}, fmt.Errorf("accumulate stream event: %w", err)
		}
		if onToken != nil {
			if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
				if text, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok {
					onToken(text.Text)
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return apiResponse{}, convertSDKError(err)
	}
	return fromSDKMessage(&msg), nil
}
