package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// previewPane renders the selected task's markdown into a scrollable viewport.
// The glamour renderer's wrap width and style are baked in at render time, so
// the content is rebuilt whenever the width, theme, or selected task changes.
type previewPane struct {
	vp      viewport.Model
	width   int    // inner content width
	key     string // identity of the rendered task (re-render guard)
	hasTask bool
	task    claude.Task
}

func newPreview() previewPane {
	return previewPane{vp: viewport.New(0, 0)}
}

// setSize updates the viewport dimensions. A width change forces a re-render
// because the markdown wrap width is fixed at render time.
func (p *previewPane) setSize(w, h int, t Theme) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	widthChanged := w != p.width
	p.width = w
	p.vp.Width = w
	p.vp.Height = h
	if widthChanged {
		p.render(t)
	}
}

// setTask points the preview at a task (ok=false clears it). Re-render is
// guarded by a key combining task identity, width and theme.
func (p *previewPane) setTask(task claude.Task, ok bool, t Theme) {
	if !ok {
		if p.hasTask || p.key != "" {
			p.hasTask = false
			p.task = claude.Task{}
			p.key = ""
			p.vp.SetContent("")
		}
		return
	}
	p.hasTask = true
	p.task = task
	p.render(t)
}

// render builds markdown and renders it through glamour at the current width and
// theme. On any failure it falls back to raw text so the pane is never blank.
func (p *previewPane) render(t Theme) {
	key := fmt.Sprintf("%s/%s|%d|%s", p.task.SessionID, p.task.ID, p.width, t.GlamourStyle)
	if key == p.key {
		return // nothing changed
	}
	p.key = key

	if !p.hasTask {
		p.vp.SetContent("")
		return
	}

	md := buildMarkdown(p.task)
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(t.GlamourStyle),
		glamour.WithWordWrap(p.width),
	)
	if err != nil {
		p.vp.SetContent(md) // raw fallback
		p.vp.GotoTop()
		return
	}
	out, err := r.Render(md)
	if err != nil {
		p.vp.SetContent(md) // raw fallback
		p.vp.GotoTop()
		return
	}
	p.vp.SetContent(out)
	p.vp.GotoTop()
}

// scroll moves the viewport by one line (negative delta scrolls up).
func (p *previewPane) scroll(delta int) {
	if delta > 0 {
		p.vp.LineDown(1)
	} else {
		p.vp.LineUp(1)
	}
}

// Update forwards scroll keys/mouse to the viewport.
func (p previewPane) Update(msg tea.Msg) (previewPane, tea.Cmd) {
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return p, cmd
}

func (p previewPane) View() string { return p.vp.View() }

// buildMarkdown is the document shown for a task: an H1 subject, the description,
// and (when present) the blocking relationships — all read-only.
func buildMarkdown(task claude.Task) string {
	var b strings.Builder
	subject := task.Subject
	if subject == "" {
		subject = "(no subject)"
	}
	fmt.Fprintf(&b, "# %s\n\n", subject)

	statusLabel := map[string]string{
		claude.StatusPending:    "Pending",
		claude.StatusInProgress: "In Progress",
		claude.StatusCompleted:  "Completed",
	}[task.Status]
	if statusLabel == "" {
		statusLabel = task.Status
	}
	fmt.Fprintf(&b, "**Status:** %s · **#%s**\n\n", statusLabel, task.ID)

	if task.Description != "" {
		b.WriteString(task.Description)
		b.WriteString("\n")
	} else {
		b.WriteString("_No description._\n")
	}

	if len(task.BlockedBy) > 0 {
		fmt.Fprintf(&b, "\n**Blocked by:** %s\n", strings.Join(task.BlockedBy, ", "))
	}
	if len(task.Blocks) > 0 {
		fmt.Fprintf(&b, "\n**Blocks:** %s\n", strings.Join(task.Blocks, ", "))
	}
	return b.String()
}
