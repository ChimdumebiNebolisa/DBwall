package report

import (
	"encoding/json"
	"time"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
	"github.com/ChimdumebiNebolisa/DBwall/internal/version"
)

// JSONOutput is the stable JSON shape for CI and agent pipelines.
type JSONOutput struct {
	Tool         string          `json:"tool"`
	Version      string          `json:"version"`
	GeneratedAt  string          `json:"generated_at"`
	CoverageMode string          `json:"coverage_mode"`
	Decision     string          `json:"decision"`
	Severity     string          `json:"severity"`
	Summary      JSONSummary     `json:"summary"`
	Statements   []JSONStatement `json:"statements"`
}

// JSONSummary summarizes the run outcome.
type JSONSummary struct {
	Statements int `json:"statements"`
	Findings   int `json:"findings"`
	Blocks     int `json:"blocks"`
	Warnings   int `json:"warnings"`
}

// JSONStatement is one statement in the JSON output.
type JSONStatement struct {
	Index     int                      `json:"index"`
	Type      string                   `json:"type"`
	Table     string                   `json:"table"`
	Object    string                   `json:"object,omitempty"`
	StartLine int                      `json:"start_line,omitempty"`
	Location  *analyzer.SourceLocation `json:"location,omitempty"`
	Findings  []JSONFinding            `json:"findings"`
}

// JSONFinding is one finding in the JSON output.
type JSONFinding struct {
	Rule           string `json:"rule"`
	Title          string `json:"title"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	Decision       string `json:"decision"`
	Message        string `json:"message"`
	Rationale      string `json:"rationale"`
	Remediation    string `json:"remediation"`
	StatementIndex int    `json:"statement_index"`
}

// JSON formats the analysis result as a JSON string.
func JSON(res *analyzer.Result, opts ...Options) (string, error) {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if res == nil {
		out := JSONOutput{
			Tool:         "dbguard",
			Version:      version.Version,
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			CoverageMode: opt.CoverageMode,
			Decision:     "allow",
			Severity:     "low",
			Statements:   nil,
		}
		b, err := json.Marshal(out)
		return string(b), err
	}
	out := JSONOutput{
		Tool:         "dbguard",
		Version:      version.Version,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		CoverageMode: opt.CoverageMode,
		Decision:     string(res.Decision),
		Severity:     string(res.Severity),
		Summary: JSONSummary{
			Statements: res.Summary.Statements,
			Findings:   res.Summary.Findings,
			Blocks:     res.Summary.Blocks,
			Warnings:   res.Summary.Warnings,
		},
		Statements: make([]JSONStatement, len(res.Statements)),
	}
	for i, st := range res.Statements {
		out.Statements[i] = JSONStatement{
			Index:     st.Index,
			Type:      st.Type,
			Table:     st.Table,
			Object:    st.Object,
			StartLine: st.StartLine,
			Location:  st.Location,
			Findings:  make([]JSONFinding, len(st.Findings)),
		}
		for j, f := range st.Findings {
			out.Statements[i].Findings[j] = JSONFinding{
				Rule:           f.Rule,
				Title:          f.Title,
				Category:       f.Category,
				Severity:       string(f.Severity),
				Decision:       string(f.Decision),
				Message:        f.Message,
				Rationale:      f.Rationale,
				Remediation:    f.Remediation,
				StatementIndex: f.StatementIndex,
			}
		}
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
