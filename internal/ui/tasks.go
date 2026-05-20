package ui

import (
	"github.com/charmbracelet/bubbles/list"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// taskItem is one task in a Todo/Doing/Done pane.
type taskItem struct{ task claude.Task }

func (i taskItem) FilterValue() string { return i.task.Subject }

func (i taskItem) line(width int, selected bool, t Theme) string {
	if selected {
		return t.Selected.Render("▸ " + truncate(i.task.Subject, width-2))
	}
	return t.statusStyle(i.task.Status).Render("•") + " " + truncate(i.task.Subject, width-2)
}

// taskItems wraps tasks as list items.
func taskItems(tasks []claude.Task) []list.Item {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}
	return items
}

// selectedTaskItem returns the task currently selected in a list, if it holds
// taskItems.
func selectedTaskItem(l list.Model) (claude.Task, bool) {
	if it, ok := l.SelectedItem().(taskItem); ok {
		return it.task, true
	}
	return claude.Task{}, false
}
