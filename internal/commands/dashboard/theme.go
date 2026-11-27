package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The colors in use, set by UseTheme.
var (
	colorTeal      lipgloss.Color
	colorTealDim   lipgloss.Color
	colorGreen     lipgloss.Color
	colorRed       lipgloss.Color
	colorRedDim    lipgloss.Color
	colorBlue      lipgloss.Color
	colorGray      lipgloss.Color
	colorOrange    lipgloss.Color
	colorYellow    lipgloss.Color
	colorLink      lipgloss.Color
	colorLinkDim   lipgloss.Color
	colorStar      lipgloss.Color
	colorStarLit   lipgloss.Color
	colorText      lipgloss.Color
	colorMuted     lipgloss.Color
	colorFaint     lipgloss.Color
	colorHighlight lipgloss.Color
	colorPanel     lipgloss.Color
	colorNote      lipgloss.Color
	colorInk       lipgloss.Color
)

var projectColors = []lipgloss.Color{
	"#7AA2F7", "#BB9AF7", "#E0AF68", "#9ECE6A", "#F7768E", "#2AC3DE", "#FF9E64", "#C0CAF5",
}

var (
	styleTitle    lipgloss.Style
	styleMuted    lipgloss.Style
	styleFaint    lipgloss.Style
	styleTeal     lipgloss.Style
	styleRed      lipgloss.Style
	styleBoldRed  lipgloss.Style
	styleHeading  lipgloss.Style
	styleSelected lipgloss.Style
	styleBadge    lipgloss.Style
	styleBehind   lipgloss.Style
	styleKey      lipgloss.Style
	styleUpgrade  lipgloss.Style
	styleDeploy   lipgloss.Style
	styleNote     lipgloss.Style
)

func init() {
	if err := UseTheme(DefaultTheme); err != nil {
		panic(err)
	}
}

func restyle() {
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorText)
	styleMuted = lipgloss.NewStyle().Foreground(colorMuted)
	styleFaint = lipgloss.NewStyle().Foreground(colorFaint)
	styleTeal = lipgloss.NewStyle().Foreground(colorTeal)
	styleRed = lipgloss.NewStyle().Foreground(colorRed)
	styleBoldRed = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	styleHeading = lipgloss.NewStyle().Bold(true).Foreground(colorMuted)
	styleSelected = lipgloss.NewStyle().Bold(true).Foreground(colorInk).Background(colorTeal)
	styleBadge = lipgloss.NewStyle().Bold(true).Foreground(colorInk).Background(colorGreen).Padding(0, 1)
	styleBehind = lipgloss.NewStyle().Bold(true).Foreground(colorInk).Background(colorOrange).Padding(0, 1)
	styleKey = lipgloss.NewStyle().Bold(true).Foreground(colorTeal)
	styleUpgrade = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	styleDeploy = lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	styleNote = lipgloss.NewStyle().Foreground(colorNote)
}

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
