package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiURL           = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	requestTimeout   = 30 * time.Second
	// 1024 was too tight: extended thinking can consume the whole budget on
	// its own, leaving zero tokens for the actual reply (an empty bubble in
	// the chat UI) or cutting a Spanish response off mid-sentence. Seen live
	// while testing the propose_nutrition_target_change flow.
	maxTokens = 2048
)

// ContentBlock mirrors the Messages API's content block shape — the same
// struct is used for request messages, response messages, and what gets
// persisted to chat_messages, so history round-trips without translation.
//
// Thinking matters even though nothing here reads it: Haiku 5.5 returns a
// "thinking" block ahead of a tool call, and the API requires that block to
// come back on the next turn with its "thinking" key present — even when its
// value is "". A blanket `omitempty` drops that empty key, and the API then
// rejects the whole request with "thinking.thinking: Field required". The
// custom MarshalJSON below emits exactly the fields each block type actually
// has, instead of one fixed shape for all of them.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
	Data      string          `json:"data,omitempty"` // redacted_thinking blocks carry an opaque payload here instead
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
}

// rawContentBlock has the exact same fields as ContentBlock but no
// MarshalJSON method, so marshaling it falls back to plain reflection
// instead of recursing into ContentBlock's custom one.
type rawContentBlock ContentBlock

func (b ContentBlock) MarshalJSON() ([]byte, error) {
	switch b.Type {
	case "thinking":
		return json.Marshal(struct {
			Type      string `json:"type"`
			Thinking  string `json:"thinking"`
			Signature string `json:"signature"`
		}{b.Type, b.Thinking, b.Signature})
	case "redacted_thinking":
		return json.Marshal(struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}{b.Type, b.Data})
	case "text":
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{b.Type, b.Text})
	case "tool_use":
		return json.Marshal(struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}{b.Type, b.ID, b.Name, b.Input})
	case "tool_result":
		return json.Marshal(struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
			Content   string `json:"content"`
		}{b.Type, b.ToolUseID, b.Content})
	default:
		return json.Marshal(rawContentBlock(b))
	}
}

type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type Client struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewClient(apiKey, model string) *Client {
	return &Client{apiKey: apiKey, model: model, http: &http.Client{Timeout: requestTimeout}}
}

type apiRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
}

type apiResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Error      *apiError      `json:"error"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Send calls the Messages API and returns the assistant's content blocks and
// stop reason. Callers check StopReason == "tool_use" to know whether any
// block is a tool call worth acting on.
func (c *Client) Send(ctx context.Context, system string, messages []Message, tools []Tool) ([]ContentBlock, string, error) {
	body, err := json.Marshal(apiRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Tools:     tools,
	})
	if err != nil {
		return nil, "", fmt.Errorf("encoding Claude request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("building Claude request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("calling Claude API: %w", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading Claude response: %w", err)
	}
	var parsed apiResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, "", fmt.Errorf("decoding Claude response (status %d): %w", res.StatusCode, err)
	}
	if parsed.Error != nil {
		return nil, "", fmt.Errorf("Claude API error: %s", parsed.Error.Message)
	}
	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Claude API returned status %d", res.StatusCode)
	}

	return parsed.Content, parsed.StopReason, nil
}

// TextFrom concatenates every text block — tool_use/tool_result blocks in the
// same message are ignored, which is exactly what a chat transcript wants.
func TextFrom(blocks []ContentBlock) string {
	var out string
	for _, b := range blocks {
		if b.Type == "text" {
			out += b.Text
		}
	}
	return out
}
