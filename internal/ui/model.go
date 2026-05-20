package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/liunuozhi/claude-task/internal/claude"
	"github.com/liunuozhi/claude-task/internal/watcher"
)

// overlay is the floating panel composited over the main view, if any. Live and
// Sessions are reached only through their overlays; help is the `?` reference.
type overlay int

const (
	overlayNone overlay = iota
	overlayHelp
	overlayLive
	overlaySessions
)

// Model is the root Bubble Tea model. Per the MVU design, session/task data only
// ever enters through the dataLoadedMsg message; no goroutine mutates it directly.
type Model struct {
	base string

	// Data.
	allSessions   []claude.Session         // unfiltered, newest-first
	sessions      []claude.Session         // after filters; drives sessionList
	live          []claude.Task            // global in-progress feed
	tasksByID     map[string][]claude.Task // every session's tasks, from the last scan
	tasks         []claude.Task            // active session's tasks (tasksByID[activeSession])
	activeSession string                   // tracked by ID, never by index

	// Filters.
	projects      []string // distinct projects, for cycling
	projectFilter string   // "" = all projects
	activeOnly    bool

	// UI state.
	focus   Focus
	theme   Theme
	width   int
	height  int
	overlay overlay // none, or a floated panel (help / live / sessions)
	err     error

	// Components. The five list panes plus the preview make up the accordion.
	liveList    list.Model
	sessionList list.Model
	todoList    list.Model
	doingList   list.Model
	doneList    list.Model
	preview     previewPane
	help        help.Model
	keys        keyMap

	// Watcher subscription (nil if watching is disabled).
	events <-chan watcher.Event
}

// New constructs the root model. events may be nil (no live updates).
func New(base string, events <-chan watcher.Event) Model {
	theme := darkTheme()
	m := Model{
		base:        base,
		theme:       theme,
		focus:       FocusTodo,
		liveList:    newCompactList(theme),
		sessionList: newCompactList(theme),
		todoList:    newCompactList(theme),
		doingList:   newCompactList(theme),
		doneList:    newCompactList(theme),
		preview:     newPreview(),
		help:        help.New(),
		keys:        defaultKeys(),
		events:      events,
	}
	m.syncDelegates() // highlight only the initially focused pane
	return m
}

// taskLists returns the three status panes in stack order (Todo, Doing, Done),
// as pointers so callers can mutate them.
func (m *Model) taskLists() [3]*list.Model {
	return [3]*list.Model{&m.todoList, &m.doingList, &m.doneList}
}

// listFor returns the list backing a pane focus, or nil for Preview.
func (m *Model) listFor(f Focus) *list.Model {
	switch f {
	case FocusLive:
		return &m.liveList
	case FocusSessions:
		return &m.sessionList
	case FocusTodo:
		return &m.todoList
	case FocusDoing:
		return &m.doingList
	case FocusDone:
		return &m.doneList
	default:
		return nil
	}
}

// Init kicks off the first data scan and, if watching, the change subscription.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{loadData(m.base)}
	if m.events != nil {
		cmds = append(cmds, listenForChanges(m.events))
	}
	return tea.Batch(cmds...)
}
