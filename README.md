# claude-task

A keyboard-driven terminal UI for observing [Claude Code](https://claude.ai/code)
tasks live — a lazygit-style accordion over your `~/.claude` sessions, with no
browser, no server, and no writes. It watches the filesystem and redraws as
Claude works.

```
──[1]─Pending (5)───────────────────────────────────────
╭─[2]─In Progress (1)──────────────────────────────────╮
│▸ build the accordion layout                          │
│                                                      │
╰──────────────────────────────────────────────────────╯
──[3]─Completed (6)─────────────────────────────────────
╭─[0]─Preview · #2─────────────────────────────────────╮
│                                                      │
│   build the accordion layout                         │
│                                                      │
│  Status: In Progress · #2                            │
│  Stack the panes into one foldable column…           │
╰──────────────────────────────────────────────────────╯
 a active · s sessions · ? help · q quit
```

The main view stacks the active session's tasks — **Pending**, **In Progress**,
**Completed** (named for the task's frontmatter `status`) — above an always-open
**Preview**. It's a true accordion: only the focused task pane shows its list;
the others collapse to a single title rule. The Preview is always expanded and
tracks the selected task, so you see a task's detail just by moving the cursor —
no `Enter` needed. While a task pane is focused, Preview takes about 0.6 of the
shared height; focus Preview itself (`0`) and it fills the area.

The global **Active** feed and the **Sessions** picker open as floating overlays
(`a` and `s`) over the live UI, dismissed with `Esc`.

## Install

### curl (prebuilt binary, no Go required)

```sh
curl -fsSL https://raw.githubusercontent.com/liunuozhi/claude-task/main/install.sh | sh
```

Installs to `/usr/local/bin` by default. Override with `BIN_DIR=$HOME/.local/bin`.

### Go

Requires Go ≥ 1.26.

```sh
go install github.com/liunuozhi/claude-task@latest
# or, from a clone:
go build -o claude-task .
```

## Usage

```sh
claude-task                 # watch ~/.claude
claude-task --dir /path     # watch a specific Claude directory
claude-task --no-watch      # one-shot, no live updates
```

The base directory is resolved as `--dir` > `$CLAUDE_DIR` > `~/.claude`.

## Keys

| Key | Action |
|-----|--------|
| `1` / `2` / `3` | jump to Pending / In Progress / Completed |
| `0` | jump to Preview |
| `Tab` / `Shift-Tab` (or `l` / `h`) | cycle the three task panes |
| `j` / `k` | move the selection in the focused pane (the Preview follows), or scroll the Preview when it's focused |
| `J` / `K` | scroll the Preview a line |
| `Enter` | open the selected task in the Preview |
| `a` | open the Active feed overlay |
| `s` | open the Sessions overlay |
| `t` | toggle light / dark theme |
| `Esc` | close an overlay, leave the Preview, or clear active filters |
| `?` | help overlay |
| `q` / `Ctrl-C` | quit |

**Inside the Active / Sessions overlay:** `j` / `k` navigate, `Space` (or
`Enter`) selects and returns to the main view, `Esc` closes. The Sessions
overlay also takes `a` to toggle active-only (sessions with an in-progress task)
and `p` to cycle the project filter.

The status line shows only the primary keys (`a active · s sessions · ? help ·
q quit`); the full table above lives in the `?` overlay.

## What it shows

Four panes stacked top to bottom, plus two overlays:

- **Pending / In Progress / Completed** — the selected session's tasks split by
  their frontmatter `status`.
- **Preview** — the selected task's subject and description, rendered as markdown.
- **Active** (overlay, `a`) — a global feed of every in-progress task across
  sessions, showing the `activeForm` ("what Claude is doing now"). Selecting one
  jumps to its session and task.
- **Sessions** (overlay, `s`) — every session that has tasks, newest activity
  first, named by `customTitle` > `aiTitle` > `slug` > session ID, with `P/I/C`
  task counts. Selecting one loads its tasks.

## Design

It is read-only by construction — it never writes to `~/.claude`. Data flows in
one direction: a debounced [fsnotify](https://github.com/fsnotify/fsnotify)
watcher emits change events, a [Bubble Tea](https://github.com/charmbracelet/bubbletea)
command turns each into a message, and the model re-scans. No goroutine ever
mutates the model directly.

```
internal/
  claude/    pure data layer — session/task discovery, no TUI deps (fully unit-tested)
  watcher/   fsnotify wrapper: one goroutine, debounced events on a channel
  ui/        Bubble Tea model/update/view, Lip Gloss accordion, glamour preview
```

Malformed files are skipped rather than fatal: a single bad task JSON never
breaks a scan.

## Credits

Inspired by [L1AD/claude-task-viewer](https://github.com/L1AD/claude-task-viewer).

## License

[MIT](LICENSE)

