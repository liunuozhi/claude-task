package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/liunuozhi/claude-task/internal/claude"
	"github.com/liunuozhi/claude-task/internal/watcher"
)

// loadData scans every session, the global in-progress list, and every
// session's tasks in one pass. Run on startup and after each file change.
func loadData(base string) tea.Cmd {
	return func() tea.Msg {
		sessions, live, tasksByID, err := claude.ScanWithLive(base)
		return dataLoadedMsg{sessions: sessions, live: live, tasksByID: tasksByID, err: err}
	}
}

// listenForChanges blocks on the watcher channel and returns a fileChangedMsg.
// It is self-re-arming: Update re-issues it after every fileChangedMsg so the
// single subscription persists for the life of the program. The watcher's own
// goroutine is the only thing touching fsnotify; here we merely await it.
func listenForChanges(events <-chan watcher.Event) tea.Cmd {
	return func() tea.Msg {
		if events == nil {
			return nil
		}
		<-events
		return fileChangedMsg{}
	}
}
