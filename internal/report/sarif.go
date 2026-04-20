package report

import (
	"encoding/json"
	"fmt"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
	"github.com/ChimdumebiNebolisa/DBwall/internal/rulemeta"
	"github.com/ChimdumebiNebolisa/DBwall/internal/version"
)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	ShortDescription sarifMultiformat `json:"shortDescription"`
	FullDescription  sarifMultiformat `json:"fullDescription"`
	Help             sarifMultiformat `json:"help"`
	Properties       map[string]any   `json:"properties,omitempty"`
}

type sarifMultiformat struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"`
	Message   sarifMultiformat `json:"message"`
	Locations []sarifLocation  `json:"locations,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine,omitempty"`
}

// SARIF formats the analysis result as SARIF for code scanning systems.
func SARIF(res *analyzer.Result, opts Options) (string, error) {
	rules := make([]sarifRule, 0, len(rulemeta.All()))
	for _, rule := range rulemeta.All() {
		rules = append(rules, sarifRule{
			ID:               rule.ID,
			Name:             rule.Title,
			ShortDescription: sarifMultiformat{Text: rule.Title},
			FullDescription:  sarifMultiformat{Text: rule.Rationale},
			Help:             sarifMultiformat{Text: rule.Remediation},
			Properties: map[string]any{
				"category":        rule.Category,
				"defaultDecision": rule.DefaultDecision,
				"severity":        rule.Severity,
			},
		})
	}
	log := sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "dbguard",
				Version:        version.Version,
				InformationURI: "https://github.com/ChimdumebiNebolisa/DBwall",
				Rules:          rules,
			}},
		}},
	}
	if res != nil {
		for _, st := range res.Statements {
			for _, f := range st.Findings {
				result := sarifResult{
					RuleID:  f.Rule,
					Level:   sarifLevel(f),
					Message: sarifMultiformat{Text: fmt.Sprintf("%s. %s", f.Message, f.Remediation)},
				}
				if opts.SourcePath != "" {
					result.Locations = []sarifLocation{{
						PhysicalLocation: sarifPhysicalLocation{
							ArtifactLocation: sarifArtifactLocation{URI: opts.SourcePath},
							Region:           sarifRegion{StartLine: max(st.StartLine, 1)},
						},
					}}
				}
				log.Runs[0].Results = append(log.Runs[0].Results, result)
			}
		}
	}
	b, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func sarifLevel(f analyzer.Finding) string {
	switch f.Decision {
	case "block":
		return "error"
	case "warn":
		return "warning"
	default:
		return "note"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
