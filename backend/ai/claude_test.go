package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

// Regression test for a real bug: Claude returns thinking blocks with an
// empty "thinking" string, and the Messages API requires that key to be
// present on the next turn even when empty. A blanket `omitempty` strips it,
// and the API then rejects the whole request with
// "thinking.thinking: Field required" — which surfaced as the chat silently
// breaking right after the model's first tool call in every real test.
func TestMarshalThinkingBlockKeepsEmptyThinkingField(t *testing.T) {
	block := ContentBlock{Type: "thinking", Thinking: "", Signature: "sig123"}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshaling thinking block: %v", err)
	}

	if !strings.Contains(string(data), `"thinking":""`) {
		t.Fatalf(`expected marshaled block to contain "thinking":"", got %s`, data)
	}
	if !strings.Contains(string(data), `"signature":"sig123"`) {
		t.Fatalf("expected signature to round-trip, got %s", data)
	}
}

func TestMarshalTextBlockOmitsThinkingFields(t *testing.T) {
	block := ContentBlock{Type: "text", Text: "hola"}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshaling text block: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decoding marshaled block: %v", err)
	}
	for _, field := range []string{"thinking", "signature", "tool_use_id", "id", "name"} {
		if _, present := decoded[field]; present {
			t.Errorf("text block should not include %q, got %s", field, data)
		}
	}
}

func TestMarshalToolUseBlockRoundTrips(t *testing.T) {
	block := ContentBlock{Type: "tool_use", ID: "toolu_1", Name: "complete_onboarding", Input: json.RawMessage(`{"age":28}`)}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshaling tool_use block: %v", err)
	}

	var decoded ContentBlock
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshaling tool_use block: %v", err)
	}
	if decoded.ID != block.ID || decoded.Name != block.Name {
		t.Fatalf("tool_use block did not round-trip: got %+v", decoded)
	}
}
