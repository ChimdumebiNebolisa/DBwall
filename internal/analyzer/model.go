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
	Rule     string
	Severity Severity
	Decision policy.Decision
	Message  string
}

// StatementResult holds the analysis result for one statement.
type StatementResult struct {
	Index    int
	Type     string
	Table    string
	Findings []Finding
}

// Result is the full analysis result.
type Result struct {
	Decision   policy.Decision
	Severity   Severity
	Statements []StatementResult
}
