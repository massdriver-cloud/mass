package cli

import (
	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"github.com/rodaine/table"
)

// NewTable creates a new table with consistent formatting
func NewTable(headers ...any) table.Table {
	headerFmt := color.New(color.FgHiBlue, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgHiWhite).SprintfFunc()

	tbl := table.New(headers...)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)
	return tbl
}

// TruncateString truncates a string to maxWidth runes, handling emojis correctly
func TruncateString(s string, maxWidth int) string {
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	// Truncate by rune width, not byte length
	truncated := runewidth.Truncate(s, maxWidth, "...")
	return truncated
}
