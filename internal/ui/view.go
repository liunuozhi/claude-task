package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// statusBarH is the single row reserved for the bottom status line.
const statusBarH = 1

// View composes the main accordion (task panes + Preview) above the status line,
// with a floating overlay — help, Live feed, or Sessions — composited on top.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "" // not sized yet
	}

	rows := paneBodyRows(m.height-statusBarH, m.focus)
	panes := make([]pane, len(mainPanes))
	for i, f := range mainPanes {
		p := m.makePane(f, rows[f])
		p.index = paneIndex(f)
		p.boxed = paneExpanded(f, m.focus)
		panes[i] = p
	}

	body := renderAccordion(m.width, panes, m.theme)
	status := m.statuslineView(m.width)
	view := lipgloss.JoinVertical(lipgloss.Left, body, status)

	switch m.overlay {
	case overlayHelp:
		return overlayCenter(view, m.helpBox(), m.width, m.height)
	case overlayLive:
		return overlayCenter(view, m.liveBox(), m.width, m.height)
	case overlaySessions:
		return overlayCenter(view, m.sessionBox(), m.width, m.height)
	}
	return view
}

// makePane builds the renderable pane for a focus; collapsed panes (bodyRows==0)
// carry no body.
func (m Model) makePane(f Focus, bodyRows int) pane {
	p := pane{title: m.paneTitle(f), bodyRows: bodyRows}
	if bodyRows > 0 {
		p.body = m.paneContent(f)
	}
	return p
}

func (m Model) paneContent(f Focus) string {
	switch f {
	case FocusLive:
		return m.liveList.View()
	case FocusSessions:
		return m.sessionList.View()
	case FocusTodo:
		return m.todoList.View()
	case FocusDoing:
		return m.doingList.View()
	case FocusDone:
		return m.doneList.View()
	case FocusPreview:
		return m.preview.View()
	}
	return ""
}

func (m Model) paneTitle(f Focus) string {
	// The bracketed pane index is supplied by the accordion frame, so titles
	// here carry only the label.
	switch f {
	case FocusLive:
		return fmt.Sprintf("Active (%d)", len(m.liveList.Items()))
	case FocusSessions:
		title := fmt.Sprintf("Sessions (%d)", len(m.sessions))
		if m.projectFilter != "" {
			title += " ⦿"
		}
		if m.activeOnly {
			title += " ◐"
		}
		return title
	case FocusTodo:
		return fmt.Sprintf("Pending (%d)", len(m.todoList.Items()))
	case FocusDoing:
		return fmt.Sprintf("In Progress (%d)", len(m.doingList.Items()))
	case FocusDone:
		return fmt.Sprintf("Completed (%d)", len(m.doneList.Items()))
	case FocusPreview:
		if m.preview.hasTask {
			return "Preview · #" + m.preview.task.ID
		}
		return "Preview"
	}
	return ""
}

// Overlay panels share a border (1 each side) plus padding (2 horizontal, 1
// vertical); overlayMargin keeps them off the screen edge. overlayChromeW/H are
// the total width/height the frame and its title+footer lines consume.
const (
	overlayChromeW = 2*1 + 2*2     // border + horizontal padding
	overlayChromeH = 2*1 + 2*1 + 4 // border + vertical padding + title/2 blanks/footer
	overlayMargin  = 4
)

// overlayPanel frames content as a floating panel with the accented border and
// interior padding (the chrome overlayChromeW/H account for).
func (m Model) overlayPanel(content string) string {
	return m.theme.FocusedBorder.Padding(1, 2).Render(content)
}

// helpBox renders the full key reference as a bordered box. Each binding group is
// a column block; the blocks wrap onto new rows so the table always fits the
// terminal width instead of being clipped on the right.
func (m Model) helpBox() string {
	t := m.theme
	keyStyle := lipgloss.NewStyle().Bold(true)

	var cols []string
	for _, g := range m.keys.FullHelp() {
		var keyW int
		for _, kb := range g {
			if w := lipgloss.Width(kb.Help().Key); w > keyW {
				keyW = w
			}
		}
		rows := make([]string, len(g))
		for i, kb := range g {
			h := kb.Help()
			rows[i] = keyStyle.Render(padTo(h.Key, keyW)) + " " + t.Subtle.Render(h.Desc)
		}
		cols = append(cols, lipgloss.JoinVertical(lipgloss.Left, rows...))
	}

	wrapW := m.width - overlayChromeW - overlayMargin
	if wrapW < 1 {
		wrapW = 1
	}
	rows := flowRow(cols, "   ", wrapW)
	body := make([]string, 0, 2*len(rows))
	for i, r := range rows {
		if i > 0 {
			body = append(body, "")
		}
		body = append(body, r)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		t.Title.Render("claude-task — keys"),
		"",
		lipgloss.JoinVertical(lipgloss.Left, body...),
		"",
		t.Subtle.Render("press any key to close"),
	)
	return m.overlayPanel(content)
}

// liveBox renders the global in-progress feed (the "Active" pane) as a menu.
func (m Model) liveBox() string {
	return m.listBox(m.paneTitle(FocusLive), m.liveList, "j/k move · space select · esc close")
}

// sessionBox renders the session picker as a menu; its filters are toggled here.
func (m Model) sessionBox() string {
	return m.listBox(m.paneTitle(FocusSessions), m.sessionList,
		"j/k move · space select · a active-only · p project · esc close")
}

// listBox frames a list as a bordered overlay panel: a title, the list (sized by
// applyLayout to hug its items), and a key-hint footer. An empty list shows a
// placeholder so the panel never collapses to a blank box.
func (m Model) listBox(title string, l list.Model, footer string) string {
	t := m.theme
	body := l.View()
	if len(l.Items()) == 0 {
		body = t.Subtle.Render("— none —")
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		t.Title.Render(title),
		"",
		body,
		"",
		t.Subtle.Render(footer),
	)
	return m.overlayPanel(content)
}

// overlayCenter composites fg centered over the bg screen (w x h), splicing each
// covered background line around fg so the live UI shows through around the box.
func overlayCenter(bg, fg string, w, h int) string {
	fgLines := strings.Split(fg, "\n")
	fgW := lipgloss.Width(fg)
	x := (w - fgW) / 2
	y := (h - len(fgLines)) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	bgLines := strings.Split(bg, "\n")
	for i, fl := range fgLines {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		left := ansi.Truncate(bgLines[row], x, "")
		right := ansi.TruncateLeft(bgLines[row], x+lipgloss.Width(fl), "")
		bgLines[row] = left + "\x1b[0m" + fl + right
	}
	return strings.Join(bgLines, "\n")
}
