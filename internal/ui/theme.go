package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// Theme is a complete set of styles plus the glamour style name for markdown.
// Two instances exist (dark, light) and the user toggles between them with `t`.
type Theme struct {
	Name         string
	GlamourStyle string // "dark" | "light", passed to glamour

	// Pane borders.
	FocusedBorder lipgloss.Style
	BlurredBorder lipgloss.Style

	// Text.
	Title        lipgloss.Style // focused pane titles
	BlurredTitle lipgloss.Style // unfocused pane titles
	Subtle       lipgloss.Style // dimmed/help text and blurred borders
	Accent       lipgloss.Style // focused border runes
	Selected     lipgloss.Style // selected list row

	// Status badges, by task status.
	Pending    lipgloss.Style
	InProgress lipgloss.Style
	Completed  lipgloss.Style

	StatusBar lipgloss.Style // bottom status line
}

// borderStyle returns the style for border runes: accented when focused.
func (t Theme) borderStyle(focused bool) lipgloss.Style {
	if focused {
		return t.Accent
	}
	return t.Subtle
}

// titleStyle returns the style for a pane title: bold-accent when focused, plain
// bold otherwise so collapsed panes stay readable.
func (t Theme) titleStyle(focused bool) lipgloss.Style {
	if focused {
		return t.Title
	}
	return t.BlurredTitle
}

// statusStyle returns the badge style for a task status.
func (t Theme) statusStyle(status string) lipgloss.Style {
	switch status {
	case claude.StatusPending:
		return t.Pending
	case claude.StatusInProgress:
		return t.InProgress
	case claude.StatusCompleted:
		return t.Completed
	default:
		return t.Subtle
	}
}

// palette is the small set of colors a theme varies; everything else in the
// theme is built identically from these by buildTheme.
type palette struct {
	name, glamour                 string
	blue, green, yellow, gray, fg string
}

func buildTheme(p palette) Theme {
	blue, green, yellow := lipgloss.Color(p.blue), lipgloss.Color(p.green), lipgloss.Color(p.yellow)
	gray, fg := lipgloss.Color(p.gray), lipgloss.Color(p.fg)
	return Theme{
		Name:         p.name,
		GlamourStyle: p.glamour,
		FocusedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(blue),
		BlurredBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(gray),
		Title:        lipgloss.NewStyle().Bold(true).Foreground(blue),
		BlurredTitle: lipgloss.NewStyle().Bold(true),
		Subtle:       lipgloss.NewStyle().Foreground(gray),
		Accent:       lipgloss.NewStyle().Foreground(blue),
		Selected:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(blue),
		Pending:      lipgloss.NewStyle().Foreground(yellow),
		InProgress:   lipgloss.NewStyle().Foreground(blue).Bold(true),
		Completed:    lipgloss.NewStyle().Foreground(green),
		// No background: the status line blends with the panes.
		StatusBar: lipgloss.NewStyle().Foreground(fg),
	}
}

func darkTheme() Theme {
	return buildTheme(palette{
		name: "dark", glamour: "dark",
		blue: "#61AFEF", green: "#98C379", yellow: "#E5C07B",
		gray: "#5C6370", fg: "#ABB2BF",
	})
}

func lightTheme() Theme {
	return buildTheme(palette{
		name: "light", glamour: "light",
		blue: "#0184BC", green: "#50A14F", yellow: "#C18401",
		gray: "#A0A1A7", fg: "#383A42",
	})
}

// toggle returns the other theme.
func (t Theme) toggle() Theme {
	if t.Name == "dark" {
		return lightTheme()
	}
	return darkTheme()
}
