package analyzer

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

func TestAnalyze_Allow_WhenNoViolations(t *testing.T) {
	stmts := []parser.Statement{
		{Type: parser.StmtTypeSelect, Table: "users"},
	}
	p := policy.DefaultPolicy()
	res := Analyze(stmts, p)
	if res.Decision != policy.DecisionAllow {
		t.Errorf("want allow, got %s", res.Decision)
	}
	if len(res.Statements) != 1 {
		t.Fatalf("want 1 statement result, got %d", len(res.Statements))
	}
}

func TestAnalyze_Block_DeleteWithoutWhere(t *testing.T) {
	stmts := []parser.Statement{
		{Type: parser.StmtTypeDelete, Table: "users", HasWhere: false},
	}
	res := Analyze(stmts, policy.DefaultPolicy())
	if res.Decision != policy.DecisionBlock {
		t.Errorf("want block, got %s", res.Decision)
	}
	if res.Severity != SeverityCritical {
		t.Errorf("want severity critical, got %s", res.Severity)
	}
	if len(res.Statements[0].Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(res.Statements[0].Findings))
	}
}

func TestAnalyze_MultipleStatements_StrictestWins(t *testing.T) {
	stmts := []parser.Statement{
		{Type: parser.StmtTypeSelect, Table: "a"},
		{Type: parser.StmtTypeDelete, Table: "users", HasWhere: false},
		{Type: parser.StmtTypeUpdate, Table: "orders", HasWhere: true},
	}
	res := Analyze(stmts, policy.DefaultPolicy())
	if res.Decision != policy.DecisionBlock {
		t.Errorf("want block (from DELETE without WHERE), got %s", res.Decision)
	}
	if res.Severity != SeverityCritical {
		t.Errorf("want critical, got %s", res.Severity)
	}
	if len(res.Statements) != 3 {
		t.Fatalf("want 3 statement results, got %d", len(res.Statements))
	}
}

func TestAnalyze_StatementIndices(t *testing.T) {
	stmts := []parser.Statement{
		{Type: parser.StmtTypeDropTable, Table: "t1"},
		{Type: parser.StmtTypeDropTable, Table: "t2"},
	}
	res := Analyze(stmts, policy.DefaultPolicy())
	if res.Statements[0].Index != 1 || res.Statements[1].Index != 2 {
		t.Errorf("want indices 1 and 2, got %d and %d", res.Statements[0].Index, res.Statements[1].Index)
	}
}

func TestAnalyze_NilPolicy_UsesDefaults(t *testing.T) {
	stmts := []parser.Statement{
		{Type: parser.StmtTypeDelete, Table: "x", HasWhere: false},
	}
	res := Analyze(stmts, nil)
	if res.Decision != policy.DecisionBlock {
		t.Errorf("nil policy should use defaults (block), got %s", res.Decision)
	}
}
