package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	colorTeal      = lipgloss.Color("#5ED3B4")
	colorTealDim   = lipgloss.Color("#2F7F6C")
	colorGreen     = lipgloss.Color("#6CCF97")
	colorRed       = lipgloss.Color("#F25F5C")
	colorRedDim    = lipgloss.Color("#8E3A39")
	colorBlue      = lipgloss.Color("#6C9EF0")
	colorGray      = lipgloss.Color("#7C8594")
	colorOrange    = lipgloss.Color("#F59E42")
	colorYellow    = lipgloss.Color("#F2C94C")
	colorLink      = lipgloss.Color("#5A6680")
	colorLinkDim   = lipgloss.Color("#353C4C")
	colorStar      = lipgloss.Color("#3B4252")
	colorStarLit   = lipgloss.Color("#7B88A1")
	colorText      = lipgloss.Color("#E6EDF3")
	colorMuted     = lipgloss.Color("#8B95A7")
	colorFaint     = lipgloss.Color("#4C566A")
	colorHighlight = lipgloss.Color("#FFFFFF")
	colorPanel     = lipgloss.Color("#3B4252")
)

var projectColors = []lipgloss.Color{
	"#7AA2F7", "#BB9AF7", "#E0AF68", "#9ECE6A", "#F7768E", "#2AC3DE", "#FF9E64", "#C0CAF5",
}

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colorText)
	styleMuted    = lipgloss.NewStyle().Foreground(colorMuted)
	styleFaint    = lipgloss.NewStyle().Foreground(colorFaint)
	styleTeal     = lipgloss.NewStyle().Foreground(colorTeal)
	styleRed      = lipgloss.NewStyle().Foreground(colorRed)
	styleBoldRed  = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	styleHeading  = lipgloss.NewStyle().Bold(true).Foreground(colorMuted)
	styleSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F14")).Background(colorTeal)
	styleBadge    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F14")).Background(colorGreen).Padding(0, 1)
	styleBehind   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0B0F14")).Background(colorOrange).Padding(0, 1)
	styleKey      = lipgloss.NewStyle().Bold(true).Foreground(colorTeal)
	styleUpgrade  = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	styleDeploy   = lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	styleNote     = lipgloss.NewStyle().Foreground(lipgloss.Color("#C9B8F0"))
)

func statusColor(status string) lipgloss.Color {
	switch status {
	case StatusProvisioned:
		return colorGreen
	case StatusFailed:
		return colorRed
	case StatusInitialized:
		return colorBlue
	default:
		return colorGray
	}
}

func statusLabel(status string) string {
	switch status {
	case StatusProvisioned:
		return "provisioned"
	case StatusFailed:
		return "failed"
	case StatusInitialized:
		return "not deployed"
	case StatusDecommissioned:
		return "decommissioned"
	default:
		return status
	}
}

// Health states the UI has colors for.
const (
	HealthHealthy  = "healthy"
	HealthDegraded = "degraded"
	HealthDown     = "down"
)

// healthColor colors the health states the Massdriver UI knows; any other
// state is gray.
func healthColor(state string) lipgloss.Color {
	switch strings.ToLower(state) {
	case HealthHealthy, "ok", "up":
		return colorGreen
	case HealthDegraded, "warning", "recovering":
		return colorOrange
	case HealthDown, "failed", "error":
		return colorRed
	default:
		return colorGray
	}
}

func levelColor(level string) lipgloss.Color {
	switch level {
	case "error":
		return colorRed
	case "warning":
		return colorOrange
	default:
		return colorBlue
	}
}

func usageColor(u *Usage) lipgloss.Color {
	switch ratio := u.Value / u.Max; {
	case ratio >= 0.95:
		return colorRed
	case ratio >= 0.8:
		return colorOrange
	default:
		return colorTeal
	}
}

// hyperlink makes label a link the terminal can open, where it supports OSC 8.
func hyperlink(url, label string) string {
	if !webURL(url) {
		return label
	}
	return "\x1b]8;;" + clean(url) + "\x1b\\" + label + "\x1b]8;;\x1b\\"
}

func hopColor(hop int) lipgloss.Color {
	switch hop {
	case 0:
		return colorRed
	case 1:
		return colorOrange
	default:
		return colorYellow
	}
}

func projectColor(g *Graph, projectID string) lipgloss.Color {
	for i, id := range g.Projects {
		if id == projectID {
			return projectColors[i%len(projectColors)]
		}
	}
	return colorMuted
}

func dot(color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render("●")
}
