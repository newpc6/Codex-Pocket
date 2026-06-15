package runtime

import (
	"testing"

	"codexflow/internal/codex"
)

func TestFlattenCodexMessageContentIncludesImages(t *testing.T) {
	got := flattenCodexMessageContent([]map[string]any{
		{"type": "input_text", "text": "hello"},
		{"type": "input_image", "image_url": "data:image/png;base64,abc"},
		{"type": "image", "path": "C:\\images\\user.png"},
	})

	want := "hello\n\n![Attached image](data:image/png;base64,abc)\n\n[Attached image: C:\\images\\user.png]"
	if got != want {
		t.Fatalf("flattenCodexMessageContent() = %q, want %q", got, want)
	}
}

func TestBuildCodexEventMessageTextIncludesLocalImages(t *testing.T) {
	got := buildCodexEventMessageText(codexEventMessagePayload{
		Message:     "prompt",
		LocalImages: []any{"C:\\images\\from-event.png"},
	})

	want := "prompt\n\n[Attached image: C:\\images\\from-event.png]"
	if got != want {
		t.Fatalf("buildCodexEventMessageText() = %q, want %q", got, want)
	}
}

func TestAppendCodexAgentMessagePreservesInterleavedCommentary(t *testing.T) {
	turns := []codex.Turn{}
	turnsByID := map[string]*codex.Turn{}

	appendCodexAgentMessage(&turns, turnsByID, "turn-1", "first", "commentary", "2026-06-15T08:00:00Z")
	turn := ensureCodexTurn(&turns, turnsByID, "turn-1")
	turn.Items = append(turn.Items, map[string]any{
		"id":      "call-1",
		"type":    "dynamicToolCall",
		"title":   "shell_command",
		"summary": `{"command":"git status --short"}`,
	})
	appendCodexAgentMessage(&turns, turnsByID, "turn-1", "second", "commentary", "2026-06-15T08:00:01Z")

	if len(turns) != 1 || len(turns[0].Items) != 3 {
		t.Fatalf("items len = %d, want 3", len(turns[0].Items))
	}
	if got := toString(turns[0].Items[0]["text"]); got != "first" {
		t.Fatalf("first agent text = %q, want first", got)
	}
	if got := toString(turns[0].Items[1]["type"]); got != "dynamicToolCall" {
		t.Fatalf("second item type = %q, want dynamicToolCall", got)
	}
	if got := toString(turns[0].Items[2]["text"]); got != "second" {
		t.Fatalf("third agent text = %q, want second", got)
	}
}

func TestExtractShellCommand(t *testing.T) {
	got := extractShellCommand(`{"command":"git add web/src/stores/app.ts web/src/views/SessionDetail.vue","workdir":"F:\\project\\ai\\codexflow"}`)
	want := "git add web/src/stores/app.ts web/src/views/SessionDetail.vue"
	if got != want {
		t.Fatalf("extractShellCommand() = %q, want %q", got, want)
	}
}

func TestAppendStructuredUserInputAddsSteerMessage(t *testing.T) {
	turn := codex.Turn{
		ID: "turn-1",
		Items: []map[string]any{
			composeUserMessageItemFromInput([]map[string]any{textInput("first")}),
		},
	}

	turn = appendStructuredUserInput(turn, []map[string]any{textInput("second")})

	if len(turn.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(turn.Items))
	}
	if got := codex.FirstUserText([]map[string]any{turn.Items[1]}); got != "second" {
		t.Fatalf("second user message = %q, want second", got)
	}
}
