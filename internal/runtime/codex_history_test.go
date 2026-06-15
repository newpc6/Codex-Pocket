package runtime

import "testing"

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
