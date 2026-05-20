package ui

import "github.com/charmbracelet/x/ansi"

// truncate shortens s to at most n display columns, appending an ellipsis when
// it cuts. ansi.Truncate is width-, grapheme- and escape-aware.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return ansi.Truncate(s, n, "…")
}
