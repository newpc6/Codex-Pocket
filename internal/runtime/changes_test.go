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
