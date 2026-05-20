// Package claude is the pure data layer: it discovers Claude Code sessions and
// tasks under a ~/.claude directory. It has no TUI dependencies and is the
// foundation the UI is built on.
package claude

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveClaudeDir picks the base ~/.claude directory using the precedence
// flagDir > $CLAUDE_DIR > ~/.claude. A leading "~" in any source is expanded to
// the user's home directory.
func ResolveClaudeDir(flagDir string) string {
	if flagDir != "" {
		return expandHome(flagDir)
	}
	if env := os.Getenv("CLAUDE_DIR"); env != "" {
		return expandHome(env)
	}
	return expandHome("~/.claude")
}

// TasksDir is the directory holding per-session task folders.
func TasksDir(base string) string { return filepath.Join(base, "tasks") }

// ProjectsDir is the directory holding per-project session JSONL transcripts.
func ProjectsDir(base string) string { return filepath.Join(base, "projects") }

// expandHome replaces a leading "~" with the user's home directory. If the home
// directory can't be determined, the path is returned unchanged.
func expandHome(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
