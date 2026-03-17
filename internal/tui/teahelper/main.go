// Package teahelper provides test utilities for Bubble Tea TUI programs.
package teahelper

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TeaKeyToByteArr converts tea KeyTypes (special characters) to byte arrays
func TeaKeyToByteArr(key tea.KeyType) []byte {
	return []byte{'\x1b', byte(key)}
}

func keyPress(key rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}, Alt: false}
}

// SpecialKeyPress constructs a tea.KeyMsg for a special (non-rune) key type.
func SpecialKeyPress(keyType tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: keyType, Alt: false, Runes: []rune{}}
}

// SendSpecialKeyPress sends a special (non-rune) key press to the given Bubble Tea program.
func SendSpecialKeyPress(p *tea.Program, keyType tea.KeyType) {
	p.Send(SpecialKeyPress(keyType))
}

// SendKeyPresses sends each rune in keys as a separate key press to the given Bubble Tea program.
func SendKeyPresses(p *tea.Program, keys string) {
	for _, k := range keys {
		p.Send(keyPress(k))
	}
}

// AssertStdoutContains fails the test if the captured stdout does not contain str.
func AssertStdoutContains(t *testing.T, stdout bytes.Buffer, str string) {
	ui := stdout.String()
	if !strings.Contains(ui, str) {
		t.Errorf("Expected UI to contain '%s'\nGot:\n%s", str, ui)
	}
}

// AssertModelViewContains fails the test if the model's rendered view does not contain str.
func AssertModelViewContains(t *testing.T, view string, str string) {
	if !strings.Contains(view, str) {
		t.Errorf("Expected model view to contain '%s'\nGot:\n%s", str, view)
	}
}
