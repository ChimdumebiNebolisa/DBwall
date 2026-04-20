// Package analyzer combines parsed statements with policy and rules
// to produce findings and overall allow/warn/block decisions.

package analyzer

import "github.com/ChimdumebiNebolisa/DBwall/internal/policy"

// Severity of a finding or overall result.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Finding is one rule trigger for a statement.
type Finding struct {
	Rule           string
	Title          string
	Category       string
	Severity       Severity
	Decision       policy.Decision
	Message        string
	Rationale      string
	Remediation    string
	StatementIndex int
}

// SourceLocation describes where a statement originated in a source file.
type SourceLocation struct {
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
}

// StatementResult holds the analysis result for one statement.
type StatementResult struct {
	Index     int
	Type      string
	Table     string
	Object    string
	StartLine int
	Location  *SourceLocation
	Findings  []Finding
}

// Summary captures the overall finding counts.
type Summary struct {
	Statements int
	Findings   int
	Blocks     int
	Warnings   int
}

// Result is the full analysis result.
type Result struct {
	Decision   policy.Decision
	Severity   Severity
	Summary    Summary
	Statements []StatementResult
}
