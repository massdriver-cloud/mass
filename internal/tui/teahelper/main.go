package teahelper

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TeaKeyToByteArr(key tea.KeyType) []byte {
	return []byte{'\x1b', byte(key)}
}

func keyPress(key rune) tea.Msg {
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
		p.Send(keyPress(k))
	}
}

func AssertStdoutContains(t *testing.T, stdout bytes.Buffer, str string) {
	ui := stdout.String()
	if !strings.Contains(ui, str) {
		t.Errorf("Expected UI to contain '%s'\nGot:\n%s", str, ui)
	}
}

func AssertModelViewContains(t *testing.T, view string, str string) {
	if !strings.Contains(view, str) {
		t.Errorf("Expected model view to contain '%s'\nGot:\n%s", str, view)
	}
}
