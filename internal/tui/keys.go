package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines every binding the TUI responds to and doubles as the data
// source for the help footer (it implements help.KeyMap).
type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Inspect   key.Binding
	Kill      key.Binding
	ForceKill key.Binding
	Filter    key.Binding
	Sort      key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Quit      key.Binding

	Confirm key.Binding
	Cancel  key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Inspect:   key.NewBinding(key.WithKeys("enter", "i"), key.WithHelp("enter", "details")),
		Kill:      key.NewBinding(key.WithKeys("x", "delete"), key.WithHelp("x", "kill")),
		ForceKill: key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "force-kill")),
		Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Sort:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),

		Confirm: key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y", "confirm")),
		Cancel:  key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n", "cancel")),
	}
}

// ShortHelp implements help.KeyMap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Inspect, k.Kill, k.Filter, k.Sort, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Inspect},
		{k.Kill, k.ForceKill},
		{k.Filter, k.Sort, k.Refresh},
		{k.Help, k.Quit},
	}
}
