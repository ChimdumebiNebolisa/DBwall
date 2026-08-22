package rules

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

// Audit finding F-001: revoke statements must never trigger grant rules.
func TestCheck_RevokeNeverTriggersGrantRules(t *testing.T) {
	p := policy.DefaultPolicy()
	p.ProtectedTables = []string{"users"}
	p.ProtectedRoles = []string{"pg_read_all_data"}

	stmt := parser.Statement{
		Type:                   parser.StmtTypeRevoke,
		IsRoleMembershipRevoke: true,
		IsGrantToPublic:        false,
		Grantees:               []string{"public"},
		Relations: []parser.RelationRef{
			{Name: "users", Role: parser.RelationTarget},
		},
	}
	findings := Check(stmt, p)
	if len(findings) != 0 {
		t.Fatalf("revoke must produce no grant findings, got %#v", findings)
	}
}

// Audit finding F-005: unsupported statement shapes must surface the
// semantic_analysis_incomplete rule in core mode too.
func TestCheck_UnsupportedCoreStatementFiresIncomplete(t *testing.T) {
	stmt := parser.Statement{
		Type:              parser.StmtTypeOther,
		Completeness:      parser.AnalysisUnsupported,
		IncompleteReasons: []string{"unsupported_statement_in_core_mode"},
	}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleSemanticAnalysisIncomplete, policy.DecisionWarn)
}
