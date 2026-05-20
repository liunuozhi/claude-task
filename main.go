package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liunuozhi/claude-task/internal/claude"
	"github.com/liunuozhi/claude-task/internal/ui"
	"github.com/liunuozhi/claude-task/internal/upgrade"
	"github.com/liunuozhi/claude-task/internal/watcher"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// Subcommands are dispatched before flag parsing. Flags (which start with
	// "-") fall through to the default behaviour of launching the TUI.
	if len(os.Args) > 1 && os.Args[1] == "upgrade" {
		if err := upgrade.Run(version); err != nil {
			fmt.Fprintln(os.Stderr, "claude-task:", err)
			os.Exit(1)
		}
		return
	}

	dirFlag := flag.String("dir", "", "Claude base directory (default: $CLAUDE_DIR or ~/.claude)")
	noWatch := flag.Bool("no-watch", false, "disable live file watching")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("claude-task", version)
		return
	}

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
