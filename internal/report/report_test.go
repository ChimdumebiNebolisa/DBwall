package report

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

func TestHuman_ContainsDecisionAndSeverity(t *testing.T) {
	res := &analyzer.Result{
		Decision: policy.DecisionBlock,
		Severity: analyzer.SeverityCritical,
		Summary: analyzer.Summary{
			Statements: 1,
			Findings:   1,
			Blocks:     1,
		},
		Statements: []analyzer.StatementResult{{
			Index: 1,
			Type:  "DELETE",
			Table: "users",
			Findings: []analyzer.Finding{{
				Rule:        "delete_without_where",
				Title:       "DELETE without WHERE",
				Category:    "destructive_dml",
				Severity:    analyzer.SeverityCritical,
				Decision:    policy.DecisionBlock,
				Message:     "DELETE statement has no WHERE clause",
				Rationale:   "An unbounded DELETE can remove every row from the target relation.",
				Remediation: "Add a selective WHERE clause.",
			}},
		}},
	}
	out := Human(res, Options{CoverageMode: "core"})
	if !strings.Contains(out, "Decision: BLOCK") {
		t.Error("output should contain Decision: BLOCK")
	}
	if !strings.Contains(out, "Severity: CRITICAL") {
		t.Error("output should contain Severity: CRITICAL")
	}
	if !strings.Contains(out, "delete_without_where") {
		t.Error("output should contain rule name")
	}
	if !strings.Contains(out, "DELETE statement has no WHERE clause") {
		t.Error("output should contain message")
	}
	if !strings.Contains(out, "Rationale:") || !strings.Contains(out, "Remediation:") {
		t.Error("output should contain rationale and remediation")
	}
}

func TestJSON_ValidStructure(t *testing.T) {
	res := &analyzer.Result{
		Decision: policy.DecisionBlock,
		Severity: analyzer.SeverityCritical,
		Summary: analyzer.Summary{
			Statements: 1,
			Findings:   1,
			Blocks:     1,
		},
		Statements: []analyzer.StatementResult{{
			Index: 1,
			Type:  "DELETE",
			Table: "users",
			Findings: []analyzer.Finding{{
				Rule:           "delete_without_where",
				Title:          "DELETE without WHERE",
				Category:       "destructive_dml",
				Severity:       analyzer.SeverityCritical,
				Decision:       policy.DecisionBlock,
				Message:        "DELETE statement has no WHERE clause",
				Rationale:      "An unbounded DELETE can remove every row from the target relation.",
				Remediation:    "Add a selective WHERE clause.",
				StatementIndex: 1,
			}},
		}},
	}
	out, err := JSON(res, Options{CoverageMode: "core"})
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var parsed JSONOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.Decision != "block" {
		t.Errorf("decision want block, got %s", parsed.Decision)
	}
	if parsed.Severity != "critical" {
		t.Errorf("severity want critical, got %s", parsed.Severity)
	}
	if len(parsed.Statements) != 1 {
		t.Fatalf("statements want 1, got %d", len(parsed.Statements))
	}
	if len(parsed.Statements[0].Findings) != 1 {
		t.Fatalf("findings want 1, got %d", len(parsed.Statements[0].Findings))
	}
	if parsed.Statements[0].Findings[0].Rule != "delete_without_where" {
		t.Errorf("finding rule want delete_without_where, got %s", parsed.Statements[0].Findings[0].Rule)
	}
	if parsed.Tool != "dbguard" || parsed.CoverageMode != "core" {
		t.Errorf("unexpected metadata: %#v", parsed)
	}
	if parsed.Statements[0].Findings[0].Title == "" || parsed.Statements[0].Findings[0].Rationale == "" {
		t.Error("extended finding fields should be populated")
	}
}

func TestJSON_NilResult(t *testing.T) {
	out, err := JSON(nil, Options{CoverageMode: "core"})
	if err != nil {
		t.Fatalf("JSON(nil): %v", err)
	}
	var parsed JSONOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.Decision != "allow" {
		t.Errorf("nil result should default to allow, got %s", parsed.Decision)
	}
}

func TestSARIF_ContainsRuleAndLocation(t *testing.T) {
	res := &analyzer.Result{
		Decision: policy.DecisionBlock,
		Severity: analyzer.SeverityCritical,
		Summary:  analyzer.Summary{Statements: 1, Findings: 1, Blocks: 1},
		Statements: []analyzer.StatementResult{{
			Index:     1,
			Type:      "DELETE",
			Table:     "users",
			StartLine: 7,
			Findings: []analyzer.Finding{{
				Rule:        "delete_without_where",
				Title:       "DELETE without WHERE",
				Category:    "destructive_dml",
				Severity:    analyzer.SeverityCritical,
				Decision:    policy.DecisionBlock,
				Message:     "DELETE statement has no WHERE clause",
				Rationale:   "An unbounded DELETE can remove every row from the target relation.",
				Remediation: "Add a selective WHERE clause.",
			}},
		}},
	}
	out, err := SARIF(res, Options{SourcePath: "migrations/001.sql"})
	if err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	if !strings.Contains(out, "\"ruleId\": \"delete_without_where\"") {
		t.Fatalf("SARIF missing rule id: %s", out)
	}
	if !strings.Contains(out, "\"uri\": \"migrations/001.sql\"") {
		t.Fatalf("SARIF missing source location: %s", out)
	}
}
