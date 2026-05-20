package ui

import (
	"slices"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// Update is the single message router: dispatch by message type, then route key
// presses by the focused pane. Data only enters via dataLoadedMsg.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applyLayout()
		return m, nil

	case dataLoadedMsg:
		return m.onDataLoaded(msg)

	case fileChangedMsg:
		// Reload everything and re-arm the watcher subscription.
		return m, tea.Batch(loadData(m.base), listenForChanges(m.events))

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.onKey(msg)
	}
	return m, nil
}

// onDataLoaded reconciles a fresh scan, preserving the active session and (on a
// same-session reload) the user's place in the task lists.
func (m Model) onDataLoaded(msg dataLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	m.err = nil
	m.allSessions = msg.sessions
	m.live = msg.live
	m.tasksByID = msg.tasksByID

	m.projects = claude.ProjectList(m.allSessions)
	if m.projectFilter != "" && !slices.Contains(m.projects, m.projectFilter) {
		m.projectFilter = "" // a filtered project disappeared
	}
	m.applyFilters()

	// First load: auto-select the newest session and reset selections.
	firstLoad := m.activeSession == ""
	if firstLoad && len(m.sessions) > 0 {
		m.activeSession = m.sessions[0].ID
		m.selectSessionByID(m.activeSession)
	}
	m.loadActiveTasks(firstLoad)
	if firstLoad {
		m.focusFirstTaskPane() // land on the first pane that has tasks
	}
	return m, nil
}

// loadActiveTasks installs the active session's tasks and repopulates the
// Todo/Doing/Done panes. When reset, selections jump to the top and the preview
// shows the session's first task; otherwise the user's place is preserved.
func (m *Model) loadActiveTasks(reset bool) {
	m.tasks = m.tasksByID[m.activeSession]
	m.refreshTaskLists()
	m.applyLayout() // task pane heights track their item counts
	if reset {
		for _, l := range m.taskLists() {
			l.Select(0)
		}
		m.previewFirstTask()
	} else {
		m.refreshPreview()
	}
}

// refreshTaskLists splits the active session's tasks into the three status panes.
func (m *Model) refreshTaskLists() {
	var todo, doing, done []claude.Task
	for _, t := range m.tasks {
		switch t.Status {
		case claude.StatusInProgress:
			doing = append(doing, t)
		case claude.StatusCompleted:
			done = append(done, t)
		default:
			todo = append(todo, t)
		}
	}
	m.todoList.SetItems(taskItems(todo))
	m.doingList.SetItems(taskItems(doing))
	m.doneList.SetItems(taskItems(done))
}

// onKey routes an overlay's keys when one is open, else global keys, then keys
// specific to the focused main pane.
func (m Model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.overlay != overlayNone {
		return m.onOverlayKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.overlay = overlayHelp
		return m, nil
	case key.Matches(msg, m.keys.OpenLive):
		m.openOverlay(overlayLive)
		return m, nil
	case key.Matches(msg, m.keys.OpenSessions):
		m.openOverlay(overlaySessions)
		return m, nil
	case key.Matches(msg, m.keys.Tab):
		m.setFocus(cycleTaskPane(m.focus, 1))
		return m, nil
	case key.Matches(msg, m.keys.ShiftTab):
		m.setFocus(cycleTaskPane(m.focus, -1))
		return m, nil
	case key.Matches(msg, m.keys.ScrollDown):
		m.scrollFocused(1)
		return m, nil
	case key.Matches(msg, m.keys.ScrollUp):
		m.scrollFocused(-1)
		return m, nil
	case key.Matches(msg, m.keys.Jump):
		if d := msg.String(); len(d) == 1 {
			if f, ok := paneForDigit(d[0]); ok {
				m.setFocus(f)
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Theme):
		m.setTheme(m.theme.toggle())
		return m, nil
	case key.Matches(msg, m.keys.Back):
		return m.onBack(), nil
	}

	return m.routeFocusKey(msg)
}

// onOverlayKey routes keys while a floating overlay is open. The help overlay
// closes on any key (quit still quits); the Live and Sessions overlays support
// j/k navigation, space/enter to select-and-close, and esc to dismiss.
func (m Model) onOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.overlay == overlayHelp {
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		m.overlay = overlayNone
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Back):
		m.closeOverlay()
		return m, nil
	case key.Matches(msg, m.keys.Select), key.Matches(msg, m.keys.Enter):
		return m.selectFromOverlay()
	}

	if m.overlay == overlayLive {
		var cmd tea.Cmd
		m.liveList, cmd = m.liveList.Update(msg)
		return m, cmd
	}

	// Sessions overlay: the session-list filters live here, since this is where
	// the list they narrow is on screen.
	switch {
	case key.Matches(msg, m.keys.ActiveOnly):
		m.activeOnly = !m.activeOnly
		m.applyFilters()
		m.applyLayout()
		return m, nil
	case key.Matches(msg, m.keys.Project):
		m.cycleProject()
		m.applyFilters()
		m.applyLayout()
		return m, nil
	}
	var cmd tea.Cmd
	m.sessionList, cmd = m.sessionList.Update(msg)
	return m, cmd
}

// selectFromOverlay applies the highlighted overlay item and returns to the main
// view: a live task jumps to its session/pane; a session loads its tasks.
func (m Model) selectFromOverlay() (tea.Model, tea.Cmd) {
	switch m.overlay {
	case overlayLive:
		m.closeOverlay()
		if it, ok := m.liveList.SelectedItem().(liveItem); ok {
			return m.jumpToTask(it.task)
		}
		return m, nil
	case overlaySessions:
		if it, ok := m.sessionList.SelectedItem().(sessionItem); ok && it.s.ID != m.activeSession {
			m.activeSession = it.s.ID
			m.loadActiveTasks(true)
			m.closeOverlay()
			m.focusFirstTaskPane()
			return m, nil
		}
		m.closeOverlay()
		return m, nil
	}
	return m, nil
}

// onBack: leave the Preview pane, else clear active filters.
func (m Model) onBack() Model {
	if m.focus == FocusPreview {
		f := FocusTodo
		if m.preview.hasTask {
			f = focusForStatus(m.preview.task.Status)
		}
		m.setFocus(f)
		return m
	}
	if m.projectFilter != "" || m.activeOnly {
		m.projectFilter, m.activeOnly = "", false
		m.applyFilters()
	}
	return m
}

// routeFocusKey sends navigation keys to the currently focused main pane.
func (m Model) routeFocusKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.focus {
	case FocusTodo, FocusDoing, FocusDone:
		if key.Matches(msg, m.keys.Enter) {
			m.setFocus(FocusPreview) // dive into the description
			return m, nil
		}
		l := m.listFor(m.focus)
		var cmd tea.Cmd
		*l, cmd = l.Update(msg)
		m.previewFromList(*l)
		return m, cmd

	case FocusPreview:
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		return m, cmd
	}
	return m, nil
}

// jumpToTask switches to the task's session (if needed) and selects it in its
// status pane, focusing that pane.
func (m Model) jumpToTask(task claude.Task) (tea.Model, tea.Cmd) {
	if task.SessionID != m.activeSession {
		m.activeSession = task.SessionID
		m.selectSessionByID(task.SessionID)
		m.tasks = m.tasksByID[m.activeSession]
		m.refreshTaskLists()
	}
	f := focusForStatus(task.Status)
	m.selectTaskInList(f, task.ID)
	m.setFocus(f)
	return m, nil
}

// --- helpers -------------------------------------------------------------

// setFocus changes the expanded pane, re-lays out, and updates the preview.
func (m *Model) setFocus(f Focus) {
	m.focus = f
	m.syncDelegates()
	m.applyLayout()
	m.refreshPreview()
}

// navFocus is the pane currently taking navigation: the open overlay's list, or
// the focused main pane when no overlay is up.
func (m Model) navFocus() Focus {
	switch m.overlay {
	case overlayLive:
		return FocusLive
	case overlaySessions:
		return FocusSessions
	}
	return m.focus
}

// openOverlay floats a list overlay and moves the selection highlight onto it.
func (m *Model) openOverlay(o overlay) {
	m.overlay = o
	m.syncDelegates()
}

// closeOverlay dismisses the floating overlay and restores the main highlight.
func (m *Model) closeOverlay() {
	m.overlay = overlayNone
	m.syncDelegates()
}

// syncDelegates rebuilds each list's delegate so only the list taking navigation
// (see navFocus) renders a selection highlight.
func (m *Model) syncDelegates() {
	nav := m.navFocus()
	for _, f := range []Focus{FocusLive, FocusSessions, FocusTodo, FocusDoing, FocusDone} {
		m.listFor(f).SetDelegate(compactDelegate{theme: m.theme, focused: f == nav})
	}
}

// scrollFocused scrolls the focused pane's content by one line. In a task pane
// the Preview is co-expanded, so J/K scrolls the previewed detail just as when
// the Preview itself is focused; elsewhere it moves the list cursor (a list
// can't scroll apart from its selection).
func (m *Model) scrollFocused(delta int) {
	switch m.focus {
	case FocusPreview, FocusTodo, FocusDoing, FocusDone:
		m.preview.scroll(delta)
	default:
		l := m.listFor(m.focus)
		if l == nil {
			return
		}
		if delta > 0 {
			l.CursorDown()
		} else {
			l.CursorUp()
		}
		m.refreshPreview()
	}
}

// focusFirstTaskPane focuses the first non-empty status pane, falling back to
// the (empty) Pending pane when the session has no tasks.
func (m *Model) focusFirstTaskPane() {
	for _, f := range []Focus{FocusTodo, FocusDoing, FocusDone} {
		if len(m.listFor(f).Items()) > 0 {
			m.setFocus(f)
			return
		}
	}
	m.setFocus(FocusTodo)
}

// refreshPreview re-renders the preview from the focused task pane's selection.
// The Preview pane itself leaves the rendered content untouched.
func (m *Model) refreshPreview() {
	switch m.focus {
	case FocusTodo, FocusDoing, FocusDone:
		m.previewFromList(*m.listFor(m.focus))
	}
}

func (m *Model) previewFromList(l list.Model) {
	task, ok := selectedTaskItem(l)
	m.preview.setTask(task, ok, m.theme)
}

// previewFirstTask shows the active session's first task (Todo, then Doing, then
// Done), or clears the preview when there are none.
func (m *Model) previewFirstTask() {
	for _, l := range m.taskLists() {
		if task, ok := selectedTaskItem(*l); ok {
			m.preview.setTask(task, true, m.theme)
			return
		}
	}
	m.preview.setTask(claude.Task{}, false, m.theme)
}

// applyFilters recomputes the visible session list and live feed, then restores
// the selection by active session ID.
func (m *Model) applyFilters() {
	m.sessions = claude.ApplyFilters(m.allSessions, m.projectFilter, m.activeOnly)
	m.sessionList.SetItems(sessionItems(m.sessions))
	m.selectSessionByID(m.activeSession)

	visible := make(map[string]bool, len(m.sessions))
	for _, s := range m.sessions {
		visible[s.ID] = true
	}
	var filtered []claude.Task
	for _, t := range m.live {
		if visible[t.SessionID] {
			filtered = append(filtered, t)
		}
	}
	m.liveList.SetItems(liveItems(filtered))
}

// selectSessionByID moves the sessions list cursor to the given session.
func (m *Model) selectSessionByID(id string) {
	for i, s := range m.sessions {
		if s.ID == id {
			m.sessionList.Select(i)
			return
		}
	}
	if len(m.sessions) > 0 {
		m.sessionList.Select(0)
	}
}

// selectTaskInList positions a status pane's cursor on the task with the given ID.
func (m *Model) selectTaskInList(f Focus, id string) {
	l := m.listFor(f)
	if l == nil {
		return
	}
	for i, it := range l.Items() {
		if ti, ok := it.(taskItem); ok && ti.task.ID == id {
			l.Select(i)
			return
		}
	}
}

// cycleProject advances the project filter through ["" + sorted projects].
func (m *Model) cycleProject() {
	options := append([]string{""}, m.projects...)
	cur := slices.Index(options, m.projectFilter)
	if cur < 0 {
		cur = 0
	}
	m.projectFilter = options[(cur+1)%len(options)]
}

// setTheme swaps the theme and propagates it to the list delegates and preview.
func (m *Model) setTheme(t Theme) {
	m.theme = t
	m.syncDelegates()
	m.preview.render(t)
}

// applyLayout sizes each accordion list to its pane's body height and the
// preview to its expanded height. The overlay lists (Live, Sessions) are sized
// independently so their floating panels are ready whenever they open.
func (m *Model) applyLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	rows := paneBodyRows(m.height-statusBarH, m.focus)
	innerW := m.width - 2
	if innerW < 1 {
		innerW = 1
	}
	m.todoList.SetSize(innerW, rows[FocusTodo])
	m.doingList.SetSize(innerW, rows[FocusDoing])
	m.doneList.SetSize(innerW, rows[FocusDone])
	m.preview.setSize(innerW, rows[FocusPreview], m.theme)

	ow, maxH := m.overlayListBounds()
	m.liveList.SetSize(ow, min(len(m.liveList.Items()), maxH))
	m.sessionList.SetSize(ow, min(len(m.sessions), maxH))
}

// overlayListBounds returns the inner list width and the maximum number of rows
// for a floating list panel, deducting the panel chrome (overlayChromeW/H) and a
// margin off the screen edge, and capping the width for readability.
func (m Model) overlayListBounds() (width, maxRows int) {
	width = max(min(m.width-overlayChromeW-overlayMargin, 76), 10)
	maxRows = max(m.height-overlayChromeH-overlayMargin, 1)
	return
}
