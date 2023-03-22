package prettylogs

import "github.com/charmbracelet/lipgloss"

func Underline(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Underline(true).Foreground(lipgloss.Color("#7D56f4"))
}
