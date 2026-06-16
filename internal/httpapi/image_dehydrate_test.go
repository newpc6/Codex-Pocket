package httpapi

import (
	"log/slog"
	"strings"
	"testing"

	"codexpocket/internal/runtime"
)

func TestDehydrateInlineImagesInSessionDetail(t *testing.T) {
	server := &Server{
		logger:  slog.Default(),
		uploads: newImageUploadStore(),
	}
	dataURL := "data:image/png;base64,iVBORw0KGgo="
	detail := runtime.SessionDetail{
		Turns: []runtime.TurnDetail{
			{
				Items: []runtime.TurnItem{
					{Body: "hello\n\n![Attached image](" + dataURL + ")"},
					{Body: "[Attached image: " + dataURL + "]"},
				},
			},
		},
	}

	server.dehydrateSessionImages(&detail)

	first := detail.Turns[0].Items[0].Body
	second := detail.Turns[0].Items[1].Body
	if strings.Contains(first, "data:image/") || strings.Contains(second, "data:image/") {
		t.Fatalf("inline image data was not removed: %q / %q", first, second)
	}
	if !strings.Contains(first, "inline:inline-") || !strings.Contains(second, "inline:inline-") {
		t.Fatalf("inline image refs not found: %q / %q", first, second)
	}
}
