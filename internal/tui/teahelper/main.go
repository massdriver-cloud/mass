package teahelper

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func KeyPress(key rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}, Alt: false}
}

func SpecialKeyPress(keyType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: keyType, Alt: false, Runes: []rune{}}
}

func SendSpecialKeyPress(p *tea.Program, keyType tea.KeyType) {
	p.Send(SpecialKeyPress(keyType))
}

func SendKeyPresses(p *tea.Program, keys string) {
	for _, k := range keys {
		p.Send(KeyPress(k))
	}
}

func AssertUIContains(t *testing.T, stdout bytes.Buffer, str string) {
	ui := stdout.String()
	if !strings.Contains(ui, str) {
		t.Errorf("Expected TUI to contain '%s'\nGot:\n%s", str, ui)
	}
}
