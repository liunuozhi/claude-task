package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// pane is one section of the accordion: a title and pre-rendered body lines. An
// empty body collapses the pane to its title bar.
type pane struct {
	title    string
	body     string // content already sized to innerW x bodyRows
	bodyRows int
	index    int  // displayed [N] in the divider/title
	boxed    bool // full box (focused pane or Preview) vs a bare division rule
}

// paneExpanded reports whether a pane shows its body. Only the focused pane and
// the always-open Preview expand; the other task panes collapse to their header.
func paneExpanded(f, focus Focus) bool {
	return f == focus || f == FocusPreview
}

// previewRatioNum/Den give Preview's share of the body height when a task pane is
// focused — 0.6, so the detail pane stays taller than the list.
const previewRatioNum, previewRatioDen = 3, 5

// paneBodyRows distributes the content height between the two expanded panes: the
// focused pane and the always-open Preview. A focused task pane gives Preview 0.6
// of the space and keeps the rest; when Preview itself is focused it takes all of
// it. Every other pane collapses to its single header line (zero body rows).
func paneBodyRows(contentH int, focus Focus) [paneCount]int {
	var rows [paneCount]int
	// One header line per main pane; each boxed (expanded) pane adds a bottom
	// border. Preview is always boxed; a focused task pane is a second box.
	boxes := 1
	if focus != FocusPreview {
		boxes = 2
	}
	avail := contentH - len(mainPanes) - boxes
	if avail < 0 {
		avail = 0
	}
	if focus == FocusPreview {
		rows[FocusPreview] = avail
		return rows
	}
	rows[FocusPreview] = avail * previewRatioNum / previewRatioDen
	rows[focus] = avail - rows[FocusPreview]
	return rows
}

// renderAccordion draws the panes stacked exactly width wide. A boxed pane gets a
// full box (top/side/bottom edges, accented) — that's the focused pane and the
// always-open Preview. Every other pane collapses to a horizontal division rule
// (its header), with no body lines.
func renderAccordion(width int, panes []pane, t Theme) string {
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}

	var b strings.Builder
	for _, p := range panes {
		if p.boxed {
			b.WriteString(dividerLine("╭", "╮", p.index, p.title, width, true, t))
			b.WriteByte('\n')
			side := t.borderStyle(true).Render("│")
			for _, ln := range bodyLines(p.body, p.bodyRows, innerW) {
				b.WriteString(side + ln + side + "\n")
			}
			b.WriteString(t.borderStyle(true).Render("╰"+strings.Repeat("─", innerW)+"╯") + "\n")
			continue
		}
		// Non-focused: a bare division rule (plain "─" in place of the corners,
		// so the index lines up with the boxed pane) then inset, edgeless body.
		b.WriteString(dividerLine("─", "─", p.index, p.title, width, false, t))
		b.WriteByte('\n')
		for _, ln := range bodyLines(p.body, p.bodyRows, innerW) {
			b.WriteString(" " + ln + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// dividerLine renders a lazygit-style "<corner>─[N]─title─────<corner>" line
// filling to width: the index is attached to the frame with dashes rather than
// padded with spaces. When focused the "[N]─title" unit and border runes accent.
func dividerLine(left, right string, index int, title string, width int, focused bool, t Theme) string {
	bs := t.borderStyle(focused)
	idx := "[" + strconv.Itoa(index) + "]"
	// frame = left corner + leading "─" + right corner; the rest holds the unit
	// "[N]─title" and the fill "─". The unit's own "─" separator costs one column.
	const frame = 3
	if budget := width - frame - lipgloss.Width(idx) - 1; lipgloss.Width(title) > budget {
		title = truncate(title, budget)
	}
	unit := idx + "─" + title
	fill := width - frame - lipgloss.Width(unit)
	if fill < 0 {
		fill = 0
	}
	return bs.Render(left+"─") +
		t.titleStyle(focused).Render(unit) +
		bs.Render(strings.Repeat("─", fill)+right)
}

// bodyLines splits body into exactly rows lines, each padded/truncated to width.
func bodyLines(body string, rows, width int) []string {
	if rows <= 0 {
		return nil
	}
	var src []string
	if body != "" {
		src = strings.Split(body, "\n")
	}
	out := make([]string, rows)
	for i := 0; i < rows; i++ {
		line := ""
		if i < len(src) {
			line = src[i]
		}
		out[i] = padTo(line, width)
	}
	return out
}

// flowRow greedily packs column blocks left to right, wrapping to a new row when
// the next block (plus the gap) would exceed maxW. Each returned string is one
// row of blocks joined horizontally.
func flowRow(cols []string, gap string, maxW int) []string {
	var rows []string
	var line []string
	lineW := 0
	for _, c := range cols {
		need := lipgloss.Width(c)
		if len(line) > 0 {
			need += lipgloss.Width(gap)
		}
		if len(line) > 0 && lineW+need > maxW {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, line...))
			line, lineW = nil, 0
			need = lipgloss.Width(c)
		}
		if len(line) > 0 {
			line = append(line, gap)
		}
		line = append(line, c)
		lineW += need
	}
	if len(line) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, line...))
	}
	return rows
}

// padTo pads s with spaces (or truncates) to exactly width display columns.
func padTo(s string, width int) string {
	w := lipgloss.Width(s)
	if w == width {
		return s
	}
	if w > width {
		return truncate(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}
