package artdeftable

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	RowDown         key.Binding
	RowUp           key.Binding
	RowSelectToggle key.Binding
	Quit            key.Binding
	Save            key.Binding
	Help            key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.RowDown, k.RowUp, k.RowSelectToggle, k.Save},
		{k.Help, k.Quit},
	}
}
