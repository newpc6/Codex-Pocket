package runtime

import (
	"context"
	"strings"
	"testing"

	"codexpocket/internal/codex"
	"codexpocket/internal/store"
)

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
	files := filterChangedFiles(context.Background(), "", []ChangedFile{
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

func TestParseDiffFilesAndExtractFileDiff(t *testing.T) {
	diff := `diff --git a/internal/runtime/agent.go b/internal/runtime/agent.go
index 111..222 100644
--- a/internal/runtime/agent.go
+++ b/internal/runtime/agent.go
@@ -1,2 +1,3 @@
 package runtime
-old
+new
+more
diff --git a/web/src/views/SessionDetail.vue b/web/src/views/SessionDetail.vue
index 333..444 100644
--- a/web/src/views/SessionDetail.vue
+++ b/web/src/views/SessionDetail.vue
@@ -10,2 +10,2 @@
-before
+after`

	files := parseDiffFiles(diff)
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Path != "internal/runtime/agent.go" || files[0].Additions != 2 || files[0].Deletions != 1 {
		t.Fatalf("first file = %#v", files[0])
	}
	if files[1].Path != "web/src/views/SessionDetail.vue" || files[1].Additions != 1 || files[1].Deletions != 1 {
		t.Fatalf("second file = %#v", files[1])
	}

	fileDiff := extractFileDiff(diff, "web/src/views/SessionDetail.vue")
	if strings.Contains(fileDiff, "internal/runtime/agent.go") {
		t.Fatalf("file diff included previous file: %s", fileDiff)
	}
	if !strings.Contains(fileDiff, "SessionDetail.vue") || !strings.Contains(fileDiff, "+after") {
		t.Fatalf("file diff missing selected file content: %s", fileDiff)
	}
}

func TestSessionChangesTurnScopeUsesStoredTurnDiff(t *testing.T) {
	sessionStore, err := store.New(nil)
	if err != nil {
		t.Fatalf("create session store: %v", err)
	}
	sessionStore.UpsertThread(codex.Thread{
		ID:        "thread-1",
		CWD:       t.TempDir(),
		Status:    codex.ThreadStatus{Type: "idle"},
		CreatedAt: 100,
		UpdatedAt: 200,
		Turns: []codex.Turn{
			{ID: "turn-1", Status: "completed"},
		},
	})
	sessionStore.RecordDiff("thread-1", "turn-1", `diff --git a/web/src/views/SessionDetail.vue b/web/src/views/SessionDetail.vue
index 111..222 100644
--- a/web/src/views/SessionDetail.vue
+++ b/web/src/views/SessionDetail.vue
@@ -1,2 +1,3 @@
 <template>
+  <div>new</div>
 </template>`)

	agent := &Agent{store: sessionStore}
	changes, err := agent.SessionChanges(context.Background(), "thread-1", ChangeScopeTurn, "", "", "turn-1", "")
	if err != nil {
		t.Fatalf("SessionChanges returned error: %v", err)
	}
	if got := changes.Scope; got != ChangeScopeTurn {
		t.Fatalf("scope = %q, want %q", got, ChangeScopeTurn)
	}
	if got := changes.TurnID; got != "turn-1" {
		t.Fatalf("turn id = %q, want turn-1", got)
	}
	if changes.Summary.Files != 1 || changes.Summary.Additions != 1 {
		t.Fatalf("summary = %#v", changes.Summary)
	}
	if len(changes.Files) != 1 || changes.Files[0].Path != "web/src/views/SessionDetail.vue" {
		t.Fatalf("files = %#v", changes.Files)
	}
}
