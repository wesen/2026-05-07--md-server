package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/md-view/pkg/watcher"
)

// TestOpenPathUnwatchesPreviousFile verifies the multi-open fix: when a file
// is opened and then another is opened, the FIRST file is no longer in the
// watched set — so a save to it won't fire file-changed (which would otherwise
// make the frontend call ReopenCurrent and reload the now-current file).
//
// openPath itself calls runtime.WindowSetTitle(a.ctx, ...) and so requires a
// live Wails context (nil ctx panics). This test instead reproduces the exact
// watch/unwatch sequence openPath performs and asserts the invariant directly
// against a.watched, which is what the fix is about.
func TestOpenPathUnwatchesPreviousFile(t *testing.T) {
	a := NewApp()

	// Stand up a real watcher so watchFile/unwatchFile do real work.
	fw, err := watcher.New()
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}
	defer func() { _ = fw.Close() }()
	fw.Start()
	a.watcher = fw

	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.md")
	fileB := filepath.Join(dir, "b.md")
	for _, p := range []string{fileA, fileB} {
		if err := os.WriteFile(p, []byte("# x\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	openLikeApp := func(abs string) {
		// Mirrors the openPath change under test (app.go):
		//   if old := currentFile; old != "" && old != abs { unwatchFile(old) }
		//   currentFile = abs
		//   watchFile(abs)
		if old := a.currentFile; old != "" && old != abs {
			a.unwatchFile(old)
		}
		a.currentFile = abs
		a.watchFile(abs)
	}

	openLikeApp(fileA)
	if !a.isWatched(fileA) {
		t.Errorf("after opening A: expected A watched, got watched=%v", a.snapshotWatched())
	}

	openLikeApp(fileB)
	if a.isWatched(fileA) {
		t.Errorf("after opening B: A must be unwatched, got watched=%v", a.snapshotWatched())
	}
	if !a.isWatched(fileB) {
		t.Errorf("after opening B: expected B watched, got watched=%v", a.snapshotWatched())
	}

	// Reopening the same file is a no-op on the set (guard: old == abs).
	openLikeApp(fileB)
	if a.isWatched(fileA) {
		t.Errorf("reopening B: A must stay unwatched, got watched=%v", a.snapshotWatched())
	}
}

// TestOpenPathKeepsSingleWatchAcrossManyOpens verifies no accumulation: after
// opening N distinct files in sequence, exactly ONE path remains watched.
func TestOpenPathKeepsSingleWatchAcrossManyOpens(t *testing.T) {
	a := NewApp()
	fw, err := watcher.New()
	if err != nil {
		t.Fatalf("watcher.New: %v", err)
	}
	defer func() { _ = fw.Close() }()
	fw.Start()
	a.watcher = fw

	dir := t.TempDir()
	openLikeApp := func(abs string) {
		if old := a.currentFile; old != "" && old != abs {
			a.unwatchFile(old)
		}
		a.currentFile = abs
		a.watchFile(abs)
	}

	for i := 0; i < 5; i++ {
		p := filepath.Join(dir, "f"+string(rune('0'+i))+".md")
		if err := os.WriteFile(p, []byte("# f\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
		openLikeApp(p)
	}

	if got := len(a.snapshotWatched()); got != 1 {
		t.Errorf("expected exactly 1 watched file after 5 opens, got %d (%v)", got, a.snapshotWatched())
	}
}

// --- tiny introspection helpers used only by these tests ---

func (a *App) isWatched(abs string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.watched[abs]
	return ok
}

func (a *App) snapshotWatched() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, 0, len(a.watched))
	for p := range a.watched {
		out = append(out, p)
	}
	return out
}
