package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the full set of bindings. It implements help.KeyMap so the `?`
// overlay is generated directly from these definitions.
type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	Tab          key.Binding
	ShiftTab     key.Binding
	Jump         key.Binding
	Enter        key.Binding
	Select       key.Binding
	OpenLive     key.Binding
	OpenSessions key.Binding
	ActiveOnly   key.Binding
	Project      key.Binding
	Theme        key.Binding
	Help         key.Binding
	Back         key.Binding
	Quit         key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down: key.NewBinding(
			key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		ScrollUp: key.NewBinding(
			key.WithKeys("K"), key.WithHelp("K", "scroll up")),
		ScrollDown: key.NewBinding(
			key.WithKeys("J"), key.WithHelp("J", "scroll down")),
		Tab: key.NewBinding(
			key.WithKeys("tab", "l"), key.WithHelp("tab/l", "next pane")),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab", "h"), key.WithHelp("⇧tab/h", "prev pane")),
		Jump: key.NewBinding(
			key.WithKeys("0", "1", "2", "3"), key.WithHelp("0-3", "jump to pane")),
		Enter: key.NewBinding(
			key.WithKeys("enter"), key.WithHelp("enter", "select/open")),
		Select: key.NewBinding(
			key.WithKeys(" "), key.WithHelp("space", "select")),
		OpenLive: key.NewBinding(
			key.WithKeys("a"), key.WithHelp("a", "active feed")),
		OpenSessions: key.NewBinding(
			key.WithKeys("s"), key.WithHelp("s", "sessions")),
		ActiveOnly: key.NewBinding(
			key.WithKeys("a"), key.WithHelp("a", "active-only")),
		Project: key.NewBinding(
			key.WithKeys("p"), key.WithHelp("p", "cycle project")),
		Theme: key.NewBinding(
			key.WithKeys("t"), key.WithHelp("t", "theme")),
		Help: key.NewBinding(
			key.WithKeys("?"), key.WithHelp("?", "help")),
		Back: key.NewBinding(
			key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp is the primary one-line hint set: open the overlays, help, and quit.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.OpenLive, k.OpenSessions, k.Help, k.Quit}
}

// FullHelp is the grouped set shown in the `?` overlay. The active-only and
// project filters live in the Sessions overlay, hence grouped with it.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.ScrollUp, k.ScrollDown},
		{k.Tab, k.ShiftTab, k.Jump},
		{k.OpenLive, k.OpenSessions, k.Enter, k.Select, k.Back},
		{k.ActiveOnly, k.Project, k.Theme},
		{k.Help, k.Quit},
	}
}
