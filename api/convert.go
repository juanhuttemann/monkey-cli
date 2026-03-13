package api

import (
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

// toSDKMessages converts our internal Message slice to SDK MessageParam slice.
func toSDKMessages(messages []Message) ([]anthropic.MessageParam, error) {
	result := make([]anthropic.MessageParam, 0, len(messages))
	for _, m := range messages {
		param, err := toSDKMessage(m)
		if err != nil {
			return nil, err
		}
		result = append(result, param)
	}
	return result, nil
}

func toSDKMessage(m Message) (anthropic.MessageParam, error) {
	switch content := m.Content.(type) {
	case string:
		block := anthropic.NewTextBlock(content)
		if m.Role == "user" {
			return anthropic.NewUserMessage(block), nil
		}
		return anthropic.NewAssistantMessage(block), nil
	case []ContentBlock:
		blocks, err := toSDKContentBlocks(content)
		if err != nil {
			return anthropic.MessageParam{}, err
		}
		if m.Role == "user" {
			return anthropic.NewUserMessage(blocks...), nil
		}
		return anthropic.NewAssistantMessage(blocks...), nil
	default:
		return anthropic.MessageParam{}, fmt.Errorf("unexpected message content type: %T", m.Content)
	}
}

func toSDKContentBlocks(blocks []ContentBlock) ([]anthropic.ContentBlockParamUnion, error) {
	result := make([]anthropic.ContentBlockParamUnion, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "text":
			result = append(result, anthropic.NewTextBlock(b.Text))
		case "tool_use":
			result = append(result, anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    b.ID,
					Name:  b.Name,
					Input: b.Input,
				},
			})
		case "tool_result":
			result = append(result, anthropic.NewToolResultBlock(b.ToolUseID, b.Content, false))
		}
	}
	return result, nil
}

// toSDKTools converts our internal Tool slice to SDK ToolUnionParam slice.
func toSDKTools(tools []Tool) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		result = append(result, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: t.InputSchema.Properties,
					Required:   t.InputSchema.Required,
				},
			},
		})
	}
	return result
}

// fromSDKMessage converts an *anthropic.Message to our internal apiResponse type,
// preserving all content blocks and usage data.
func fromSDKMessage(msg *anthropic.Message) apiResponse {
	if msg == nil {
		return apiResponse{}
	}
	content := make([]ContentBlock, 0, len(msg.Content))
	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			content = append(content, ContentBlock{Type: "text", Text: b.Text})
		case anthropic.ToolUseBlock:
			var input map[string]any
			if len(b.Input) > 0 {
				_ = json.Unmarshal(b.Input, &input)
			}
			content = append(content, ContentBlock{
				Type:  "tool_use",
				ID:    b.ID,
				Name:  b.Name,
				Input: input,
			})
		}
	}
	return apiResponse{
		Content:    content,
		StopReason: string(msg.StopReason),
		Usage: struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}{
			InputTokens:  int(msg.Usage.InputTokens),
			OutputTokens: int(msg.Usage.OutputTokens),
		},
	}
}
