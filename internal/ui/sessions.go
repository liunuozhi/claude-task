package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// compactItem is a list item that knows how to render itself on a single line.
// Both sessionItem and liveItem implement it; compactDelegate renders any of
// them, keeping the two sidebar lists visually consistent.
type compactItem interface {
	list.Item
	line(width int, selected bool, t Theme) string
}

// compactDelegate is a one-line list delegate driven by the active Theme. Only
// the focused pane's delegate highlights its selection, so an always-open pane
// doesn't show a stray highlight when the cursor lives elsewhere.
type compactDelegate struct {
	theme   Theme
	focused bool
}

func (d compactDelegate) Height() int                         { return 1 }
func (d compactDelegate) Spacing() int                        { return 0 }
func (d compactDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ci, ok := item.(compactItem)
	if !ok {
		return
	}
	selected := d.focused && index == m.Index()
	fmt.Fprint(w, ci.line(m.Width(), selected, d.theme))
}

// sessionItem wraps a session for the sessions list.
type sessionItem struct{ s claude.Session }

func (i sessionItem) FilterValue() string { return i.s.DisplayName() }

func (i sessionItem) line(width int, selected bool, t Theme) string {
	counts := fmt.Sprintf("%d/%d/%d", i.s.Pending, i.s.InProgress, i.s.Completed)
	name := truncate(i.s.DisplayName(), width-lipgloss.Width(counts)-3)
	gap := width - lipgloss.Width(name) - lipgloss.Width(counts) - 2
	if gap < 1 {
		gap = 1
	}
	prefix := "  "
	if selected {
		prefix = "› "
	}
	line := prefix + name + spaces(gap) + counts
	if selected {
		return t.Selected.Render(line)
	}
	return line
}

// spaces returns n spaces, guarding against the negative count strings.Repeat
// would panic on.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

// newCompactList builds a sidebar list.Model with our compact delegate and all
// of list's own chrome disabled (the pane border supplies the title). Both the
// Sessions and Live lists use it.
func newCompactList(theme Theme) list.Model {
	l := list.New(nil, compactDelegate{theme: theme}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	return l
}

// sessionItems converts sessions into list items.
func sessionItems(sessions []claude.Session) []list.Item {
	items := make([]list.Item, len(sessions))
	for i, s := range sessions {
		items[i] = sessionItem{s: s}
	}
	return items
}
