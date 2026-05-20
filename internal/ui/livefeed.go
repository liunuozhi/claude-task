package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// liveItem is one in-progress task in the global Live feed, shown by its
// activeForm (the "-ing" phrasing Claude is currently working on).
type liveItem struct {
	task claude.Task
}

func (i liveItem) FilterValue() string { return i.task.ActiveForm }

// text is what the feed displays: activeForm, falling back to the subject when
// no activeForm is recorded.
func (i liveItem) text() string {
	if i.task.ActiveForm != "" {
		return i.task.ActiveForm
	}
	return i.task.Subject
}

func (i liveItem) line(width int, selected bool, t Theme) string {
	if selected {
		return t.Selected.Render("● " + truncate(i.text(), width-2))
	}
	return t.InProgress.Render("●") + " " + truncate(i.text(), width-lipgloss.Width("● "))
}

// liveItems converts the flat in-progress task list into list items.
func liveItems(live []claude.Task) []list.Item {
	items := make([]list.Item, len(live))
	for i, task := range live {
		items[i] = liveItem{task: task}
	}
	return items
}
