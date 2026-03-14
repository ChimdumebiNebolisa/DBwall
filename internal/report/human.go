package report

import (
	"fmt"
	"strings"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
)

// Human formats the analysis result for terminal output.
func Human(res *analyzer.Result) string {
	if res == nil {
		return "Decision: ALLOW\nSeverity: LOW\n"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Decision: %s\n", strings.ToUpper(string(res.Decision))))
	b.WriteString(fmt.Sprintf("Severity: %s\n\n", strings.ToUpper(string(res.Severity))))
	for _, st := range res.Statements {
		b.WriteString(fmt.Sprintf("Statement %d:\n", st.Index))
		b.WriteString(fmt.Sprintf("  Type: %s\n", st.Type))
		if st.Table != "" {
			b.WriteString(fmt.Sprintf("  Table: %s\n", st.Table))
		}
		if len(st.Findings) > 0 {
			b.WriteString("  Triggered Rules:\n")
			for _, f := range st.Findings {
				b.WriteString(fmt.Sprintf("    - %s\n", f.Rule))
			}
			b.WriteString("  Reason:\n")
			for _, f := range st.Findings {
				b.WriteString(fmt.Sprintf("    - %s\n", f.Message))
			}
			b.WriteString("  Recommendation:\n")
			b.WriteString("    - Add a restricting predicate or require manual approval\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n\n")
}
