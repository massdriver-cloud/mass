package prettylogs

import "github.com/charmbracelet/lipgloss"

func Underline(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Underline(true).Foreground(lipgloss.Color("#7D56f4"))
}

func Green(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Foreground(lipgloss.Color("#00FF00"))
}

func Orange(word string) lipgloss.Style {
	return lipgloss.NewStyle().SetString(word).Foreground(lipgloss.Color("#FFA500"))
}
