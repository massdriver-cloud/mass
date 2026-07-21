package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// emptyDash renders an empty string as an em dash so blank cells read clearly.
func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// paramValueCell renders one side of a leaf-level param comparison. A missing
// key is shown as "—" to distinguish it from a present-but-empty value.
func paramValueCell(v types.ParamValue) string {
	if !v.Present {
		return "—"
	}
	if v.Value == "" {
		return `""`
	}
	return v.Value
}

// formatVersionComparison renders a version diff as "unchanged" or "src → tgt".
func formatVersionComparison(v types.VersionComparison) string {
	if v.Equal {
		return fmt.Sprintf("%s (unchanged)", emptyDash(v.Source))
	}
	return fmt.Sprintf("%s → %s", emptyDash(v.Source), emptyDash(v.Target))
}

// printParamComparisons prints a table of leaf-level param diffs. When showAll
// is false, only entries that differ are shown. Returns the number of params
// that differ (regardless of showAll).
func printParamComparisons(params []types.ParamComparison, showAll bool) int {
	diffCount := 0
	tbl := cli.NewTable("", "Path", "Source", "Target")
	rows := 0
	for _, p := range params {
		if !p.Equal {
			diffCount++
		}
		if p.Equal && !showAll {
			continue
		}
		marker := "="
		if !p.Equal {
			marker = "≠"
		}
		tbl.AddRow(marker, p.Path, paramValueCell(p.Source), paramValueCell(p.Target))
		rows++
	}
	if rows > 0 {
		tbl.Print()
	}
	return diffCount
}
