package bundle

import (
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/prettylogs"
)

// LintSeverity represents the severity level of a lint issue
type LintSeverity int

const (
	// LintWarning represents a non-blocking issue that should be reported but not halt execution
	LintWarning LintSeverity = iota
	// LintError represents a blocking issue that should halt execution
	LintError
)

// String returns the string representation of LintSeverity
func (s LintSeverity) String() string {
	switch s {
	case LintWarning:
		return prettylogs.Orange("WARNING").String()
	case LintError:
		return prettylogs.Red("ERROR").String()
	default:
		return "UNKNOWN"
	}
}

// LintIssue represents a single lint issue with its severity and message
type LintIssue struct {
	Severity LintSeverity
	Message  string
	Rule     string // The name of the lint rule that generated this issue
}

// Error implements the error interface for LintIssue
func (i LintIssue) Error() string {
	return fmt.Sprintf("[%s]: %s", i.Severity, i.Message)
}

// LintResult holds the results of a linting operation
type LintResult struct {
	Issues []LintIssue
}

// AddError adds an error-level issue to the result
func (r *LintResult) AddError(rule, message string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: LintError,
		Message:  message,
		Rule:     rule,
	})
}

// AddWarning adds a warning-level issue to the result
func (r *LintResult) AddWarning(rule, message string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: LintWarning,
		Message:  message,
		Rule:     rule,
	})
}

// HasIssues returns true if the result contains any error-level issues
func (r *LintResult) HasIssues() bool {
	return len(r.Issues) > 0
}

// HasErrors returns true if the result contains any error-level issues
func (r *LintResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == LintError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if the result contains any warning-level issues
func (r *LintResult) HasWarnings() bool {
	for _, issue := range r.Issues {
		if issue.Severity == LintWarning {
			return true
		}
	}
	return false
}

// Errors returns all error-level issues
func (r *LintResult) Errors() []LintIssue {
	var errors []LintIssue
	for _, issue := range r.Issues {
		if issue.Severity == LintError {
			errors = append(errors, issue)
		}
	}
	return errors
}

// Warnings returns all warning-level issues
func (r *LintResult) Warnings() []LintIssue {
	var warnings []LintIssue
	for _, issue := range r.Issues {
		if issue.Severity == LintWarning {
			warnings = append(warnings, issue)
		}
	}
	return warnings
}

// Merge combines this result with another result
func (r *LintResult) Merge(other LintResult) {
	r.Issues = append(r.Issues, other.Issues...)
}

// IsClean returns true if there are no issues at all
func (r *LintResult) IsClean() bool {
	return len(r.Issues) == 0
}
