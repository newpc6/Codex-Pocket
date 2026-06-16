package runtime

import "testing"

func TestCleanChangePathsRejectsUnsafePaths(t *testing.T) {
	cwd := t.TempDir()

	paths, err := cleanChangePaths(cwd, []string{"web/src/App.vue", `internal\runtime\changes.go`, "web/src/App.vue"})
	if err != nil {
		t.Fatalf("cleanChangePaths returned error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("len(paths) = %d, want 2", len(paths))
	}
	if paths[0] != "web/src/App.vue" {
		t.Fatalf("paths[0] = %q", paths[0])
	}
	if paths[1] != "internal/runtime/changes.go" {
		t.Fatalf("paths[1] = %q", paths[1])
	}

	for _, unsafe := range [][]string{
		{"../outside.go"},
		{"/tmp/outside.go"},
		{"nested/../../outside.go"},
	} {
		if _, err := cleanChangePaths(cwd, unsafe); err == nil {
			t.Fatalf("cleanChangePaths(%q) returned nil error", unsafe)
		}
	}
}

func TestFilterChangedFilesHidesGeneratedAndZeroLineChanges(t *testing.T) {
	files := filterChangedFiles([]ChangedFile{
		{Path: "internal/runtime/changes.go", Status: "M", Additions: 12},
		{Path: "a/internal/httpapi/server.go", Status: "M"},
		{Path: "b/internal/store/store.go", Status: "M", Deletions: 3},
		{Path: ".gitignore", Status: "M", Additions: 1, Deletions: 1},
		{Path: "dist/index.html", Status: "??", Untracked: true},
		{Path: "web/dist/assets/app.js", Status: "??", Untracked: true},
	})

	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2: %#v", len(files), files)
	}
	if files[0].Path != "internal/runtime/changes.go" {
		t.Fatalf("files[0].Path = %q", files[0].Path)
	}
	if files[1].Path != "internal/store/store.go" {
		t.Fatalf("files[1].Path = %q", files[1].Path)
	}
}
