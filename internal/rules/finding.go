package rules

import "github.com/ChimdumebiNebolisa/DBwall/internal/policy"

// Finding is a single rule trigger (rule name, severity, decision, message).
// The analyzer converts these into report output.
type Finding struct {
	Rule     string
	Severity string
	Decision policy.Decision
	Message  string
}
