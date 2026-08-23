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

// Staging-copy exclusion closed: INSERT ... SELECT from a protected source warns.
func TestCheck_InsertSelectFromProtectedWarns(t *testing.T) {
	p := policy.DefaultPolicy()
	p.ProtectedTables = []string{"users"}
	stmt := parser.Statement{
		Type: parser.StmtTypeInsert,
		Relations: []parser.RelationRef{
			{Name: "archive", Role: parser.RelationWrite},
			{Name: "users", Role: parser.RelationRead},
		},
	}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleInsertSelectFromProtected, policy.DecisionWarn)

	// Unprotected source stays silent.
	p2 := policy.DefaultPolicy()
	p2.ProtectedTables = []string{"payments"}
	if findings := Check(stmt, p2); len(findings) != 0 {
		t.Fatalf("unprotected INSERT..SELECT must not fire: %#v", findings)
	}

	// Plain VALUES insert has no read source and never fires.
	valuesStmt := parser.Statement{
		Type: parser.StmtTypeInsert,
		Relations: []parser.RelationRef{
			{Name: "users", Role: parser.RelationWrite},
		},
	}
	for _, f := range Check(valuesStmt, p) {
		if f.Rule == policy.RuleInsertSelectFromProtected {
			t.Fatalf("write-only INSERT must not fire read rule: %#v", f)
		}
	}
}
