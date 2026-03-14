package rules

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

func TestCheck_DeleteWithoutWhere_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDelete, Table: "users", HasWhere: false}
	p := policy.DefaultPolicy()
	findings := Check(stmt, p)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Rule != policy.RuleDeleteWithoutWhere {
		t.Errorf("want rule delete_without_where, got %s", findings[0].Rule)
	}
	if findings[0].Decision != policy.DecisionBlock {
		t.Errorf("want block, got %s", findings[0].Decision)
	}
}

func TestCheck_DeleteWithWhere_NoFinding(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDelete, Table: "users", HasWhere: true}
	findings := Check(stmt, policy.DefaultPolicy())
	if len(findings) != 0 {
		t.Errorf("want 0 findings for DELETE with WHERE, got %d", len(findings))
	}
}

func TestCheck_UpdateWithoutWhere_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "users", HasWhere: false}
	p := policy.DefaultPolicy()
	findings := Check(stmt, p)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Rule != policy.RuleUpdateWithoutWhere {
		t.Errorf("want rule update_without_where, got %s", findings[0].Rule)
	}
	if findings[0].Decision != policy.DecisionBlock {
		t.Errorf("want block, got %s", findings[0].Decision)
	}
}

func TestCheck_UpdateWithWhere_NoFinding(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "users", HasWhere: true}
	findings := Check(stmt, policy.DefaultPolicy())
	if len(findings) != 0 {
		t.Errorf("want 0 findings for UPDATE with WHERE, got %d", len(findings))
	}
}

func TestCheck_DropTable_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDropTable, Table: "users"}
	p := policy.DefaultPolicy()
	findings := Check(stmt, p)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Rule != policy.RuleDropTable {
		t.Errorf("want rule drop_table, got %s", findings[0].Rule)
	}
	if findings[0].Decision != policy.DecisionBlock {
		t.Errorf("want block, got %s", findings[0].Decision)
	}
}

func TestCheck_DropColumn_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeAlterTableDropCol, Table: "t1"}
	p := policy.DefaultPolicy()
	findings := Check(stmt, p)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Rule != policy.RuleDropColumn {
		t.Errorf("want rule drop_column, got %s", findings[0].Rule)
	}
	if findings[0].Decision != policy.DecisionBlock {
		t.Errorf("want block, got %s", findings[0].Decision)
	}
}

func TestCheck_ProtectedTable_Warns(t *testing.T) {
	p := &policy.Policy{
		Dialect:         policy.DialectPostgres,
		ProtectedTables: []string{"users", "payments"},
		Rules:           nil, // uses default: writes_to_protected_tables = warn
	}
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "users", HasWhere: true}
	findings := Check(stmt, p)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding (protected table), got %d", len(findings))
	}
	if findings[0].Rule != policy.RuleWritesToProtectedTable {
		t.Errorf("want rule writes_to_protected_tables, got %s", findings[0].Rule)
	}
}

func TestCheck_ProtectedTable_Delete(t *testing.T) {
	p := &policy.Policy{
		Dialect:         policy.DialectPostgres,
		ProtectedTables: []string{"users"},
		Rules:           nil,
	}
	stmt := parser.Statement{Type: parser.StmtTypeDelete, Table: "users", HasWhere: true}
	findings := Check(stmt, p)
	var protectedFinding *Finding
	for i := range findings {
		if findings[i].Rule == policy.RuleWritesToProtectedTable {
			protectedFinding = &findings[i]
			break
		}
	}
	if protectedFinding == nil {
		t.Fatal("want writes_to_protected_tables finding for DELETE on protected table")
	}
}

func TestCheck_NonProtectedTable_NoProtectedFinding(t *testing.T) {
	p := &policy.Policy{
		Dialect:         policy.DialectPostgres,
		ProtectedTables: []string{"users"},
		Rules:           nil,
	}
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "orders", HasWhere: true}
	findings := Check(stmt, p)
	for _, f := range findings {
		if f.Rule == policy.RuleWritesToProtectedTable {
			t.Error("should not trigger protected table for non-protected table")
		}
	}
}

func TestCheck_PolicyOverride_Allow(t *testing.T) {
	p := &policy.Policy{
		Dialect: policy.DialectPostgres,
		Rules:   map[string]string{policy.RuleDropTable: "allow"},
	}
	stmt := parser.Statement{Type: parser.StmtTypeDropTable, Table: "tmp"}
	findings := Check(stmt, p)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Decision != policy.DecisionAllow {
		t.Errorf("policy should allow drop_table; got %s", findings[0].Decision)
	}
}

func TestCheck_NilPolicy_UsesDefaults(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDelete, Table: "x", HasWhere: false}
	findings := Check(stmt, nil)
	if len(findings) != 1 {
		t.Fatalf("want 1 finding with nil policy (defaults), got %d", len(findings))
	}
	if findings[0].Decision != policy.DecisionBlock {
		t.Errorf("default should block delete without where, got %s", findings[0].Decision)
	}
}
