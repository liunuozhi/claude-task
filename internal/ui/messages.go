package ui

import "github.com/liunuozhi/claude-task/internal/claude"

// dataLoadedMsg carries a fresh scan into the Model — the only way session/task
// data enters it, produced by the loadData Cmd on startup and after every file
// change. live is every in-progress task across sessions (the global Live feed);
// tasksByID holds each session's full task slice so selecting a session needs no
// further disk reads.
type dataLoadedMsg struct {
	sessions  []claude.Session
	live      []claude.Task
	tasksByID map[string][]claude.Task
	err       error
}

// fileChangedMsg is emitted by the watcher-listening Cmd when the filesystem
// under ~/.claude changes. The Update loop reacts by re-issuing loadData.
type fileChangedMsg struct{}

// errMsg reports a non-fatal error to display in the status line.
type errMsg struct{ err error }
