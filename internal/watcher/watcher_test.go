package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// waitEvent waits up to d for an event, returning whether one arrived.
func waitEvent(t *testing.T, w *Watcher, d time.Duration) bool {
	t.Helper()
	select {
	case <-w.Events:
		return true
	case <-time.After(d):
		return false
	}
}

func TestWatcherDebouncesWrites(t *testing.T) {
	base := t.TempDir()
	sess := filepath.Join(claude.TasksDir(base), "sess1")
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claude.ProjectsDir(base), 0o755); err != nil {
		t.Fatal(err)
	}

	w, err := New(base)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	// A burst of writes to an existing watched dir should coalesce into one
	// event (delivered after the debounce window).
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(sess, "1.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waitEvent(t, w, 2*time.Second) {
		t.Fatal("expected a change event after writes")
	}

	// Drain any trailing coalesced event so the next assertion is clean.
	waitEvent(t, w, 300*time.Millisecond)
}

func TestWatcherPicksUpNewSessionDir(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(claude.TasksDir(base), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claude.ProjectsDir(base), 0o755); err != nil {
		t.Fatal(err)
	}

	w, err := New(base)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer w.Close()

	// Create a brand-new session dir (watched roots are added on Create), then
	// write a task into it. The watcher must Add the new dir and report the
	// subsequent write.
	newSess := filepath.Join(claude.TasksDir(base), "fresh")
	if err := os.Mkdir(newSess, 0o755); err != nil {
		t.Fatal(err)
	}
	if !waitEvent(t, w, 2*time.Second) {
		t.Fatal("expected event for new session dir creation")
	}
	waitEvent(t, w, 300*time.Millisecond) // drain

	if err := os.WriteFile(filepath.Join(newSess, "1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitEvent(t, w, 2*time.Second) {
		t.Fatal("expected event for write inside the new session dir")
	}
}
