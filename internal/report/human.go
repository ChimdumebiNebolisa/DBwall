package report

import (
	"fmt"
	"strings"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
)

// Human formats the analysis result for terminal output.
func Human(res *analyzer.Result, opts ...Options) string {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if res == nil {
		return "Decision: ALLOW\nSeverity: LOW\n"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Decision: %s\n", strings.ToUpper(string(res.Decision))))
	b.WriteString(fmt.Sprintf("Severity: %s\n", strings.ToUpper(string(res.Severity))))
	b.WriteString(fmt.Sprintf("Summary: %d statement(s), %d finding(s), %d block(s), %d warning(s)\n", res.Summary.Statements, res.Summary.Findings, res.Summary.Blocks, res.Summary.Warnings))
	if opt.CoverageMode != "" {
		b.WriteString(fmt.Sprintf("Coverage Mode: %s\n", strings.ToUpper(opt.CoverageMode)))
		if opt.CoverageMode == "full" {
			b.WriteString("Note: full mode derives security metadata from the PostgreSQL AST.\n")
		} else {
			b.WriteString("Note: core mode uses a portable token parser with reduced semantic coverage.\n")
		}
	}
	b.WriteString("\n")
	for _, st := range res.Statements {
		b.WriteString(fmt.Sprintf("Statement %d:\n", st.Index))
		b.WriteString(fmt.Sprintf("  Type: %s\n", st.Type))
		if st.Table != "" {
			b.WriteString(fmt.Sprintf("  Table: %s\n", st.Table))
		} else if st.Object != "" {
			b.WriteString(fmt.Sprintf("  Object: %s\n", st.Object))
		}
		if st.Location != nil && st.Location.Path != "" {
			b.WriteString(fmt.Sprintf("  File: %s\n", st.Location.Path))
		}
		if st.StartLine > 0 {
			b.WriteString(fmt.Sprintf("  Start Line: %d\n", st.StartLine))
		}
		if st.Completeness != "" {
			b.WriteString(fmt.Sprintf("  Completeness: %s\n", st.Completeness))
			if len(st.IncompleteReasons) > 0 {
				b.WriteString(fmt.Sprintf("  Incomplete Reasons: %s\n", strings.Join(st.IncompleteReasons, "; ")))
			}
		}
		if len(st.Findings) == 0 {
			b.WriteString("  Findings: none\n\n")
			continue
		}
		for _, f := range st.Findings {
			b.WriteString(fmt.Sprintf("  - [%s] %s (%s)\n", strings.ToUpper(string(f.Decision)), f.Title, f.Rule))
			b.WriteString(fmt.Sprintf("    Message: %s\n", f.Message))
			b.WriteString(fmt.Sprintf("    Rationale: %s\n", f.Rationale))
			b.WriteString(fmt.Sprintf("    Remediation: %s\n", f.Remediation))
		}
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n\n")
}
