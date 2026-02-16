package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	HalfUp   key.Binding
	HalfDown key.Binding
	Top       key.Binding
	Bottom    key.Binding
	NextTask  key.Binding
	PrevTask  key.Binding
	NextFeat  key.Binding
	PrevFeat  key.Binding
	Search    key.Binding
	NextMatch key.Binding
	Confirm   key.Binding
	Escape    key.Binding
	Quit      key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/up", "scroll up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/down", "scroll down"),
	),
	HalfUp: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "half page up"),
	),
	HalfDown: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "half page down"),
	),
	Top: key.NewBinding(
		key.WithKeys("home"),
		key.WithHelp("gg/Home", "go to top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("end"),
		key.WithHelp("G/End", "go to bottom"),
	),
	NextTask: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next task"),
	),
	PrevTask: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "prev task"),
	),
	NextFeat: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "next feature"),
	),
	PrevFeat: key.NewBinding(
		key.WithKeys("shift+f"),
		key.WithHelp("F", "prev feature"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	NextMatch: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next match"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
}
