package report

import (
	"encoding/json"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
)

// JSONOutput is the stable JSON shape for CI and agent pipelines.
type JSONOutput struct {
	Decision   string          `json:"decision"`
	Severity   string          `json:"severity"`
	Statements []JSONStatement `json:"statements"`
}

// JSONStatement is one statement in the JSON output.
type JSONStatement struct {
	Index    int           `json:"index"`
	Type     string        `json:"type"`
	Table    string        `json:"table"`
	Findings []JSONFinding `json:"findings"`
}

// JSONFinding is one finding in the JSON output.
type JSONFinding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Decision string `json:"decision"`
	Message  string `json:"message"`
}

// JSON formats the analysis result as a JSON string.
func JSON(res *analyzer.Result) (string, error) {
	if res == nil {
		out := JSONOutput{Decision: "allow", Severity: "low", Statements: nil}
		b, err := json.Marshal(out)
		return string(b), err
	}
	out := JSONOutput{
		Decision:   string(res.Decision),
		Severity:   string(res.Severity),
		Statements: make([]JSONStatement, len(res.Statements)),
	}
	for i, st := range res.Statements {
		out.Statements[i] = JSONStatement{
			Index:    st.Index,
			Type:     st.Type,
			Table:    st.Table,
			Findings: make([]JSONFinding, len(st.Findings)),
		}
		for j, f := range st.Findings {
			out.Statements[i].Findings[j] = JSONFinding{
				Rule:     f.Rule,
				Severity: string(f.Severity),
				Decision: string(f.Decision),
				Message:  f.Message,
			}
		}
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
