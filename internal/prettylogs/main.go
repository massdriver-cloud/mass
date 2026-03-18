// Package prettylogs provides styled terminal output helpers using lipgloss.
package prettylogs

import "github.com/charmbracelet/lipgloss"

// Underline returns a lipgloss style with underline formatting applied to word.
func Underline(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Underline(true).Foreground(lipgloss.Color("#7D56f4"))
}

// Green returns a lipgloss style with green foreground color applied to word.
func Green(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Foreground(lipgloss.Color("#00FF00"))
}

// Orange returns a lipgloss style with orange foreground color applied to word.
func Orange(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Foreground(lipgloss.Color("#FFA500"))
}

// Red returns a lipgloss style with red foreground color applied to word.
func Red(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Foreground(lipgloss.Color("#FF0000"))
}
