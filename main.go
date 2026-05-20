package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liunuozhi/claude-task/internal/claude"
	"github.com/liunuozhi/claude-task/internal/ui"
	"github.com/liunuozhi/claude-task/internal/watcher"
)

func main() {
	dirFlag := flag.String("dir", "", "Claude base directory (default: $CLAUDE_DIR or ~/.claude)")
	noWatch := flag.Bool("no-watch", false, "disable live file watching")
	flag.Parse()

	base := claude.ResolveClaudeDir(*dirFlag)
	if fi, err := os.Stat(base); err != nil || !fi.IsDir() {
		fmt.Fprintf(os.Stderr, "claude-task: cannot read %s: %v\n", base, err)
		os.Exit(1)
	}

	// Start the file watcher unless disabled. A watcher failure is non-fatal:
	// the app still runs, just without live updates.
	var events <-chan watcher.Event
	if !*noWatch {
		w, err := watcher.New(base)
		if err != nil {
			fmt.Fprintf(os.Stderr, "claude-task: live updates disabled: %v\n", err)
		} else {
			defer w.Close()
			events = w.Events
		}
	}

	p := tea.NewProgram(
		ui.New(base, events),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "claude-task: %v\n", err)
		os.Exit(1)
	}
}
