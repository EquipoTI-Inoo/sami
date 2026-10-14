package controllers

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"coachia-backend/ai"
	"coachia-backend/db"
	"coachia-backend/stores"
)

// newTestChatController returns a controller backed by a fresh SQLite test
// DB that already has one user row (id 1) — the pending_changes and
// user_profile tables both have a foreign key on users(id).
func newTestChatController(t *testing.T) *ChatController {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := db.Migrate(conn); err != nil {
		t.Fatalf("migrating test db: %v", err)
	}

	users := stores.NewUserStore(conn)
	if _, err := users.Create("chat-test@coachia.local", "hashed-password"); err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	profiles := stores.NewProfileStore(conn)
	chats := stores.NewChatStore(conn)
	changes := stores.NewPendingChangeStore(conn)
	exercises := stores.NewExerciseStore(conn)
	programs := stores.NewProgramStore(conn)
	return NewChatController(profiles, chats, changes, exercises, programs, ai.NewClient("test-key", "claude-haiku-5-5"))
}

// Regression test for a real bug: every path through handleToolCall must
// return a ToolResult, even when rejecting an invalid call — a tool_use
// block with no tool_result in the very next message permanently breaks the
// next request that replays this history (the Messages API refuses the
// whole call with "tool_use ids were found without tool_result blocks").
// This broke in production: the model called propose_nutrition_target_change
// with an incomplete payload, the original code silently `continue`d past
// it with no result, and the following chat message failed outright.
func TestHandleToolCallAlwaysReturnsAToolResult(t *testing.T) {
	cc := newTestChatController(t)

	cases := []struct {
		name  string
		block ai.ContentBlock
	}{
		{"unknown tool", ai.ContentBlock{ID: "toolu_1", Name: "not_a_real_tool", Input: json.RawMessage(`{}`)}},
		{"onboarding missing required fields", ai.ContentBlock{ID: "toolu_2", Name: "complete_onboarding", Input: json.RawMessage(`{"age":30}`)}},
		{"proposal missing reason", ai.ContentBlock{ID: "toolu_3", Name: "propose_nutrition_target_change", Input: json.RawMessage(`{"calories":2000,"protein_g":150,"carbs_g":200,"fat_g":60}`)}},
		{"proposal missing calories", ai.ContentBlock{ID: "toolu_4", Name: "propose_nutrition_target_change", Input: json.RawMessage(`{"protein_g":150,"carbs_g":200,"fat_g":60,"reason":"porque sí"}`)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := cc.handleToolCall(1, tc.block)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ToolResult.Type != "tool_result" {
				t.Fatalf("expected a tool_result block, got type %q", result.ToolResult.Type)
			}
			if result.ToolResult.ToolUseID != tc.block.ID {
				t.Fatalf("expected tool_result to reference %q, got %q", tc.block.ID, result.ToolResult.ToolUseID)
			}
			if result.ToolResult.Content == "" {
				t.Fatal("expected a non-empty tool_result content explaining what happened")
			}
		})
	}
}

func TestHandleToolCallCreatesPendingChangeOnValidProposal(t *testing.T) {
	cc := newTestChatController(t)

	block := ai.ContentBlock{
		ID:    "toolu_5",
		Name:  "propose_nutrition_target_change",
		Input: json.RawMessage(`{"calories":2000,"protein_g":150,"carbs_g":200,"fat_g":60,"reason":"no bajó de peso en dos semanas"}`),
	}

	result, err := cc.handleToolCall(1, block)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PendingChange == nil {
		t.Fatal("expected a pending change to be created")
	}
	if result.PendingChange.Status != "pending" {
		t.Fatalf("expected status pending, got %q", result.PendingChange.Status)
	}
}

func TestHandleToolCallAppliesOnboardingWithValidFields(t *testing.T) {
	cc := newTestChatController(t)

	block := ai.ContentBlock{
		ID:    "toolu_6",
		Name:  "complete_onboarding",
		Input: json.RawMessage(`{"age":30,"sex":"male","height_cm":180,"weight_kg":80,"activity_level":"moderate","goal":"cut","training_location":"gym"}`),
	}

	result, err := cc.handleToolCall(1, block)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.OnboardingCompleted || result.Targets == nil {
		t.Fatalf("expected onboarding to complete with targets, got %+v", result)
	}
}

// Regression test for a real bug: a thinking block's signature is bound to
// the exact system prompt + tools list active when Claude produced it.
// Onboarding and post-onboarding turns use different system prompts/tools,
// so once a stored thinking block from the onboarding tool-call turn got
// replayed on the next, unrelated chat request, the API rejected the whole
// call with "Invalid signature in thinking block. The block is bound to a
// different conversation." stripThinkingBlocks must drop thinking and
// redacted_thinking blocks from history loaded from storage.
func TestStripThinkingBlocksRemovesThinkingAndRedacted(t *testing.T) {
	messages := []ai.Message{
		{Role: "user", Content: []ai.ContentBlock{{Type: "text", Text: "hola"}}},
		{Role: "assistant", Content: []ai.ContentBlock{
			{Type: "thinking", Thinking: "", Signature: "sig"},
			{Type: "tool_use", ID: "toolu_1", Name: "complete_onboarding"},
		}},
		{Role: "user", Content: []ai.ContentBlock{{Type: "tool_result", ToolUseID: "toolu_1", Content: "ok"}}},
		{Role: "assistant", Content: []ai.ContentBlock{
			{Type: "redacted_thinking", Data: "opaque"},
			{Type: "text", Text: "listo"},
		}},
	}

	out := stripThinkingBlocks(messages)

	if len(out) != len(messages) {
		t.Fatalf("expected %d messages to survive (each has a non-thinking block), got %d", len(messages), len(out))
	}
	for _, m := range out {
		for _, b := range m.Content {
			if b.Type == "thinking" || b.Type == "redacted_thinking" {
				t.Fatalf("expected thinking/redacted_thinking blocks to be stripped, found %q in %+v", b.Type, m)
			}
		}
	}
	if out[1].Content[0].Type != "tool_use" {
		t.Fatalf("expected assistant message to keep its tool_use block, got %+v", out[1].Content)
	}
	if out[3].Content[0].Type != "text" || out[3].Content[0].Text != "listo" {
		t.Fatalf("expected assistant message to keep its text block, got %+v", out[3].Content)
	}
}

func TestStripThinkingBlocksDropsMessageThatBecomesEmpty(t *testing.T) {
	messages := []ai.Message{
		{Role: "assistant", Content: []ai.ContentBlock{{Type: "thinking", Thinking: "", Signature: "sig"}}},
	}

	out := stripThinkingBlocks(messages)

	if len(out) != 0 {
		t.Fatalf("expected the now-empty message to be dropped entirely, got %+v", out)
	}
}
