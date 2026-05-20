package ui

import (
	"slices"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// Focus identifies which pane receives keys. FocusLive and FocusSessions are not
// part of the main accordion; they surface as floating overlays instead.
type Focus int

const (
	FocusLive Focus = iota
	FocusSessions
	FocusTodo
	FocusDoing
	FocusDone
	FocusPreview
)

// paneCount sizes the per-pane arrays keyed by Focus.
const paneCount = 6

// mainPanes are the panes stacked in the main accordion, top to bottom. Live and
// Sessions are deliberately absent — they open as overlays. A pane's position
// here is its displayed index ([1]..[4]) and its number-jump key.
var mainPanes = []Focus{FocusTodo, FocusDoing, FocusDone, FocusPreview}

// taskPanes are the panes h/l (and Tab) cycle through. Preview is excluded: it's
// reached by diving in with Enter or jumping with its number key, not by cycling.
var taskPanes = []Focus{FocusTodo, FocusDoing, FocusDone}

func (f Focus) String() string {
	switch f {
	case FocusLive:
		return "Active"
	case FocusSessions:
		return "Sessions"
	case FocusTodo:
		return "Pending"
	case FocusDoing:
		return "In Progress"
	case FocusDone:
		return "Completed"
	case FocusPreview:
		return "Preview"
	default:
		return "?"
	}
}

// cycleTaskPane steps focus through the task panes by delta (wrapping). When
// focus is off the ring (e.g. on Preview), it enters at the near end so forward
// lands on the first pane and backward on the last.
func cycleTaskPane(f Focus, delta int) Focus {
	n := len(taskPanes)
	i := slices.Index(taskPanes, f)
	if i < 0 {
		if delta > 0 {
			return taskPanes[0]
		}
		return taskPanes[n-1]
	}
	return taskPanes[((i+delta)%n+n)%n]
}

// paneIndex is a pane's displayed [N] label and number-jump key. The task panes
// are 1/2/3; Preview is 0, set apart from the cyclable task panes.
func paneIndex(f Focus) int {
	if f == FocusPreview {
		return 0
	}
	return slices.Index(taskPanes, f) + 1
}

// paneForDigit maps a number-jump key back to its pane (0 = Preview, 1-3 = task
// panes), reporting false for any other key.
func paneForDigit(d byte) (Focus, bool) {
	if d == '0' {
		return FocusPreview, true
	}
	if i := int(d - '1'); i >= 0 && i < len(taskPanes) {
		return taskPanes[i], true
	}
	return 0, false
}

// focusForStatus returns the task pane that holds tasks of the given status.
func focusForStatus(status string) Focus {
	switch status {
	case claude.StatusInProgress:
		return FocusDoing
	case claude.StatusCompleted:
		return FocusDone
	default:
		return FocusTodo
	}
}
