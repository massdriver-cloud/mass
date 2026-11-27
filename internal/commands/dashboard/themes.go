package dashboard

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DefaultPalette is the palette the dashboard starts in.
const DefaultPalette = "night"

// Palette is the colors the dashboard draws with. Red, orange, and yellow
// keep their meaning in every palette: alarms, warnings, and the blast
// radius.
type Palette struct {
	Accent, AccentDim, Green, Red, RedDim, Blue, Gray, Orange, Yellow lipgloss.Color
	Link, LinkDim, Back, BackLit, Text, Muted, Faint, Highlight       lipgloss.Color
	Panel, Note, Ink                                                  lipgloss.Color
}

type palette struct {
	colors     Palette
	background func(c *Canvas, frame int)
}

var palettes = map[string]palette{
	"night": {
		colors: Palette{
			Accent: "#5ED3B4", AccentDim: "#2F7F6C", Green: "#6CCF97", Red: "#F25F5C", RedDim: "#8E3A39",
			Blue: "#6C9EF0", Gray: "#7C8594", Orange: "#F59E42", Yellow: "#F2C94C",
			Link: "#5A6680", LinkDim: "#353C4C", Back: "#3B4252", BackLit: "#7B88A1",
			Text: "#E6EDF3", Muted: "#8B95A7", Faint: "#4C566A", Highlight: "#FFFFFF",
			Panel: "#3B4252", Note: "#C9B8F0", Ink: "#0B0F14",
		},
		background: starfield,
	},
	"rain": {
		colors: Palette{
			Accent: "#39FF88", AccentDim: "#178A43", Green: "#3DDC84", Red: "#FF4D4D", RedDim: "#7A2020",
			Blue: "#6FD3A8", Gray: "#4E7A5C", Orange: "#FFB347", Yellow: "#E8FF6A",
			Link: "#1F7A40", LinkDim: "#0F3D20", Back: "#0B3D1C", BackLit: "#2FAF55",
			Text: "#C8FFD8", Muted: "#5FBF7B", Faint: "#2A6B3D", Highlight: "#F0FFF4",
			Panel: "#14532D", Note: "#A8FFCF", Ink: "#021A0A",
		},
		background: digitalRain,
	},
	"blueprint": {
		colors: Palette{
			Accent: "#7FD4FF", AccentDim: "#4A8DB8", Green: "#7EE0B0", Red: "#FF6B6B", RedDim: "#8A3A3A",
			Blue: "#9DB8FF", Gray: "#7F93AD", Orange: "#FFB86B", Yellow: "#FFE08A",
			Link: "#5C86B0", LinkDim: "#2C4A6B", Back: "#1D3654", BackLit: "#2E5A85",
			Text: "#F2F8FF", Muted: "#9BB8D4", Faint: "#3E5F82", Highlight: "#FFFFFF",
			Panel: "#2E5A85", Note: "#D6E9FF", Ink: "#071828",
		},
		background: draftingGrid,
	},
}

var drawBackground = starfield

// Palettes lists the palette names, the default first.
func Palettes() []string {
	names := []string{DefaultPalette}
	for name := range palettes {
		if name != DefaultPalette {
			names = append(names, name)
		}
	}
	slices.Sort(names[1:])
	return names
}

// UsePalette switches every color and the map's background to a palette.
func UsePalette(name string) error {
	t, ok := palettes[name]
	if !ok {
		return fmt.Errorf("no palette %q; choose %s", name, strings.Join(Palettes(), ", "))
	}
	p := t.colors
	colorTeal, colorTealDim, colorGreen, colorRed, colorRedDim = p.Accent, p.AccentDim, p.Green, p.Red, p.RedDim
	colorBlue, colorGray, colorOrange, colorYellow = p.Blue, p.Gray, p.Orange, p.Yellow
	colorLink, colorLinkDim, colorStar, colorStarLit = p.Link, p.LinkDim, p.Back, p.BackLit
	colorText, colorMuted, colorFaint, colorHighlight = p.Text, p.Muted, p.Faint, p.Highlight
	colorPanel, colorNote, colorInk = p.Panel, p.Note, p.Ink
	drawBackground = t.background
	restyle()
	return nil
}

// starfield scatters dim stars, a few of them twinkling.
func starfield(c *Canvas, frame int) {
	for y := range c.DotH() {
		for x := range c.DotW() {
			if hash01(x, y, 7) > 0.0025 {
				continue
			}
			color := colorStar
			if hash01(x, y, frame/6) < 0.25 {
				color = colorStarLit
			}
			c.Dot(x, y, color, layerStars)
		}
	}
}

// digitalRain drops streams down a few columns, each at its own speed, with
// a bright head and a fading tail.
func digitalRain(c *Canvas, frame int) {
	h := c.DotH()
	for x := 0; x < c.DotW(); x += 3 {
		if hash01(x, 0, 11) > 0.35 {
			continue
		}
		speed := 0.4 + hash01(x, 1, 13)*0.9
		length := 6 + int(hash01(x, 2, 17)*18)
		head := int(math.Mod(float64(frame)*speed+hash01(x, 3, 19)*float64(h*2), float64(h+length)))
		for i := range length {
			y := head - i
			if y < 0 || y >= h || (i > 0 && hash01(x, y, 23) < 0.3) {
				continue
			}
			color := colorStar
			if i == 0 {
				color = colorStarLit
			}
			c.Dot(x, y, color, layerStars)
		}
	}
}

// draftingGrid marks a faint grid of crosses with dashed major lines, the
// way a blueprint is ruled.
func draftingGrid(c *Canvas, _ int) {
	for y := range c.DotH() {
		for x := range c.DotW() {
			switch {
			case (x%64 == 0 && y%6 == 0) || (y%64 == 0 && x%6 == 0):
				c.Dot(x, y, colorStarLit, layerStars)
			case cross(x%16, y%16):
				c.Dot(x, y, colorStar, layerStars)
			}
		}
	}
}

func cross(dx, dy int) bool {
	return (dx == 0 && (dy == 0 || dy == 1 || dy == 15)) || (dy == 0 && (dx == 1 || dx == 15))
}
