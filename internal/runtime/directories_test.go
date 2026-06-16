package runtime

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"codexpocket/internal/config"
)

func TestBrowseDirectoriesListsChildDirectories(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project-a")
	filePath := filepath.Join(root, "notes.txt")
	if err := os.Mkdir(projectDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	agent := NewAgent(config.Config{}, logger)

	result, err := agent.BrowseDirectories(root)
	if err != nil {
		t.Fatalf("BrowseDirectories() error = %v", err)
	}
	if result.CurrentPath == "" {
		t.Fatalf("CurrentPath is empty")
	}
	if len(result.Entries) != 1 {
		t.Fatalf("len(result.Entries) = %d, want 1", len(result.Entries))
	}
	if result.Entries[0].Name != "project-a" {
		t.Fatalf("Entries[0].Name = %q, want %q", result.Entries[0].Name, "project-a")
	}
	if !result.Entries[0].IsDir {
		t.Fatalf("Entries[0].IsDir = false, want true")
	}
	if len(result.Roots) == 0 {
		t.Fatalf("Roots should not be empty")
	}
}
