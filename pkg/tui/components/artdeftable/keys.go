package artdeftable

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	RowDown         key.Binding
	RowUp           key.Binding
	RowSelectToggle key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.RowDown, k.RowUp, k.RowSelectToggle},
	}
}
