package analyzer

import (
	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
	"github.com/ChimdumebiNebolisa/DBwall/internal/rules"
)

// Analyze runs the rule engine on parsed statements and returns the combined result.
func Analyze(stmts []parser.Statement, p *policy.Policy) *Result {
	if p == nil {
		p = policy.DefaultPolicy()
	}
	var statementResults []StatementResult
	for i, stmt := range stmts {
		findings := rules.Check(stmt, p)
		sr := StatementResult{
			Index:     i + 1,
			Type:      string(stmt.Type),
			Table:     stmt.Table,
			Object:    stmt.Object,
			StartLine: stmt.StartLine,
			Findings:  convertFindings(i+1, findings),
		}
		statementResults = append(statementResults, sr)
	}
	decision, severity, summary := aggregate(statementResults)
	return &Result{
		Decision:   decision,
		Severity:   severity,
		Summary:    summary,
		Statements: statementResults,
	}
}

func convertFindings(statementIndex int, fs []rules.Finding) []Finding {
	out := make([]Finding, len(fs))
	for i, f := range fs {
		out[i] = Finding{
			Rule:           f.Rule,
			Title:          f.Title,
			Category:       f.Category,
			Severity:       Severity(f.Severity),
			Decision:       f.Decision,
			Message:        f.Message,
			Rationale:      f.Rationale,
			Remediation:    f.Remediation,
			StatementIndex: statementIndex,
		}
	}
	return out
}

// aggregate returns the overall decision and severity (strictest wins).
func aggregate(srs []StatementResult) (policy.Decision, Severity, Summary) {
	var maxDecision policy.Decision
	var maxSev Severity
	summary := Summary{Statements: len(srs)}
	for _, sr := range srs {
		for _, f := range sr.Findings {
			summary.Findings++
			switch f.Decision {
			case policy.DecisionBlock:
				summary.Blocks++
			case policy.DecisionWarn:
				summary.Warnings++
			}
			if decisionStrictness(f.Decision) > decisionStrictness(maxDecision) {
				maxDecision = f.Decision
			}
			if severityLevel(f.Severity) > severityLevel(maxSev) {
				maxSev = f.Severity
			}
		}
	}
	if maxDecision == "" {
		maxDecision = policy.DecisionAllow
	}
	if maxSev == "" {
		maxSev = SeverityLow
	}
	return maxDecision, maxSev, summary
}

func decisionStrictness(d policy.Decision) int {
	switch d {
	case policy.DecisionBlock:
		return 3
	case policy.DecisionWarn:
		return 2
	case policy.DecisionAllow:
		return 1
	default:
		return 0
	}
}

func severityLevel(s Severity) int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}
