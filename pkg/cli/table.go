package cli

import (
	"github.com/fatih/color"
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
