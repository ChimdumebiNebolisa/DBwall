package rules

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

func TestCheck_WritesAllMutationRelations(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"users", "payments"}}
	stmt := parser.Statement{
		Type: parser.StmtTypeTruncate,
		Relations: []parser.RelationRef{
			{Name: "users", Role: parser.RelationWrite},
			{Name: "payments", Role: parser.RelationWrite},
			{Name: "orders", Role: parser.RelationWrite},
		},
	}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleWritesToProtectedTable, policy.DecisionWarn)
	var count int
	for _, f := range findings {
		if f.Rule == policy.RuleWritesToProtectedTable {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("want 2 protected write findings, got %d (%#v)", count, findings)
	}
}

func TestCheck_GrantMultipleProtectedTargets(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"a", "b"}}
	stmt := parser.Statement{
		Type:            parser.StmtTypeGrant,
		IsGrantToPublic: true,
		Relations: []parser.RelationRef{
			{Name: "a", Role: parser.RelationTarget},
			{Name: "b", Role: parser.RelationTarget},
		},
	}
	findings := Check(stmt, p)
	var count int
	for _, f := range findings {
		if f.Rule == policy.RuleGrantToPublicProtected {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("want 2 grant findings, got %d", count)
	}
}

func TestCheck_SemanticAnalysisIncomplete(t *testing.T) {
	stmt := parser.Statement{
		Type:              parser.StmtTypeOther,
		Completeness:      parser.AnalysisUnsupported,
		IncompleteReasons: []string{"unsupported_statement_type"},
	}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleSemanticAnalysisIncomplete, policy.DecisionWarn)
}

func TestCheck_CoreModeReasonDoesNotFireIncomplete(t *testing.T) {
	stmt := parser.Statement{
		Type:              parser.StmtTypeSelect,
		Completeness:      parser.AnalysisPartial,
		IncompleteReasons: []string{"core_mode_token_parser"},
	}
	findings := Check(stmt, policy.DefaultPolicy())
	for _, f := range findings {
		if f.Rule == policy.RuleSemanticAnalysisIncomplete {
			t.Fatal("core-mode marker alone must not emit incomplete rule")
		}
	}
}

func TestCheck_NestedDeleteRules(t *testing.T) {
	stmt := parser.Statement{
		Type: parser.StmtTypeSelect,
		Nested: []parser.Statement{{
			Type:     parser.StmtTypeDelete,
			Table:    "users",
			HasWhere: false,
			Relations: []parser.RelationRef{
				{Name: "users", Role: parser.RelationWrite},
			},
		}},
	}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleDeleteWithoutWhere, policy.DecisionBlock)
}
