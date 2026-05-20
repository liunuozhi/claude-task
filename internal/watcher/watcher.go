// Package watcher wraps fsnotify to emit coalesced, debounced change events for
// a ~/.claude directory. It owns exactly one goroutine and one *fsnotify.Watcher;
// the UI consumes a receive-only channel and never touches fsnotify directly.
package watcher

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/liunuozhi/claude-task/internal/claude"
)

// Event signals that something under the watched tree changed. It carries no
// payload: the UI responds by re-scanning from scratch.
type Event struct{}

// debounce is how long to wait for activity to settle before emitting an event.
// Writes to a task file often arrive as several rapid ops; coalescing them into
// one redraw avoids thrashing.
const debounce = 200 * time.Millisecond

// Watcher emits debounced change events on Events.
type Watcher struct {
	Events <-chan Event

	fsw  *fsnotify.Watcher
	out  chan Event
	base string
	done chan struct{}
}

// New starts watching the tasks/ and projects/ trees under base. The returned
// Watcher's goroutine runs until Close is called.
func New(base string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	out := make(chan Event, 1)
	w := &Watcher{
		Events: out,
		fsw:    fsw,
		out:    out,
		base:   base,
		done:   make(chan struct{}),
	}

	// fsnotify is not recursive: add the roots and every existing subdirectory.
	w.addTree(claude.TasksDir(base))
	w.addTree(claude.ProjectsDir(base))

	go w.loop()
	return w, nil
}

// Close stops the goroutine and releases the underlying watcher.
func (w *Watcher) Close() error {
	close(w.done)
	return w.fsw.Close()
}

// addTree adds root and all of its descendant directories to the watcher,
// ignoring errors (a vanished or unreadable dir is non-fatal).
func (w *Watcher) addTree(root string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries, keep walking
		}
		if d.IsDir() {
			_ = w.fsw.Add(path)
		}
		return nil
	})
}

// loop is the single goroutine: it debounces fsnotify events and watches for new
// directories so freshly-created sessions are picked up.
func (w *Watcher) loop() {
	var timer *time.Timer
	var fire <-chan time.Time

	for {
		select {
		case <-w.done:
			return

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			// A new directory (e.g. a new session's task folder) must be added
			// so its files are watched too.
			if ev.Op.Has(fsnotify.Create) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = w.fsw.Add(ev.Name)
				}
			}
			// (Re)start the debounce timer.
			if timer == nil {
				timer = time.NewTimer(debounce)
			} else {
				timer.Reset(debounce)
			}
			fire = timer.C

		case <-fire:
			fire = nil
			// Coalesce: if an event is already queued, drop this one.
			select {
			case w.out <- Event{}:
			default:
			}

		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			// fsnotify errors are non-fatal here; ignore and keep watching.
		}
	}
}
