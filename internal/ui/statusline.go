package ui

import (
	"fmt"
	"strings"
)

// statuslineView renders the bottom bar: just the key hints (or an error).
func (m Model) statuslineView(width int) string {
	content := " " + m.keyHints()
	if m.err != nil {
		content = truncate(fmt.Sprintf(" error: %s", m.err), width)
	}
	return m.theme.StatusBar.Width(width).MaxWidth(width).Render(content)
}

// keyHints is the status line's key reference, styled like the help component's
// short view. Inside a Live/Sessions overlay it shows that overlay's keys; the
// full table lives in the `?` overlay.
func (m Model) keyHints() string {
	s := m.help.Styles
	pair := func(k, d string) string {
		return s.ShortKey.Render(k) + " " + s.ShortDesc.Render(d)
	}
	var pairs []string
	switch m.overlay {
	case overlayLive:
		pairs = []string{pair("j/k", "move"), pair("space", "select"), pair("esc", "close")}
	case overlaySessions:
		pairs = []string{
			pair("j/k", "move"),
			pair("space", "select"),
			pair("a", "active-only"),
			pair("p", "project"),
			pair("esc", "close"),
		}
	default:
		pairs = []string{
			pair("a", "active"),
			pair("s", "sessions"),
			pair("?", "help"),
			pair("q", "quit"),
		}
	}
	return strings.Join(pairs, s.ShortSeparator.Render(" · "))
}
