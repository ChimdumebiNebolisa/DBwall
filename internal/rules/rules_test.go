package rules

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

func TestCheck_DeleteWithoutWhere_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDelete, Table: "users", HasWhere: false}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleDeleteWithoutWhere, policy.DecisionBlock)
}

func TestCheck_DeleteTrivialWhere_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDelete, Table: "users", HasWhere: true, WhereTrivial: true}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleDeleteTrivialWhere, policy.DecisionBlock)
}

func TestCheck_UpdateWithoutWhere_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "users", HasWhere: false}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleUpdateWithoutWhere, policy.DecisionBlock)
}

func TestCheck_UpdateTrivialWhere_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "users", HasWhere: true, WhereTrivial: true}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleUpdateTrivialWhere, policy.DecisionBlock)
}

func TestCheck_DropTable_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDropTable, Table: "users"}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleDropTable, policy.DecisionBlock)
}

func TestCheck_DropSchema_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDropSchema, Object: "reporting"}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleDropSchema, policy.DecisionBlock)
}

func TestCheck_DropDatabase_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeDropDatabase, Object: "analytics"}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleDropDatabase, policy.DecisionBlock)
}

func TestCheck_DropColumn_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeAlterTableDropCol, Table: "t1"}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleDropColumn, policy.DecisionBlock)
}

func TestCheck_DropConstraint_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeAlterTable, Table: "t1", DropConstraint: true}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleAlterDropSafetyConstraint, policy.DecisionBlock)
}

func TestCheck_TruncateTable_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeTruncate, Table: "users"}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleTruncateTable, policy.DecisionBlock)
}

func TestCheck_ProtectedTable_Warns(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"users"}}
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "users", HasWhere: true}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleWritesToProtectedTable, policy.DecisionWarn)
}

func TestCheck_GrantToPublicProtectedObject_Blocks(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"users"}}
	stmt := parser.Statement{Type: parser.StmtTypeGrant, Table: "users", IsGrantToPublic: true}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleGrantToPublicProtected, policy.DecisionBlock)
}

func TestCheck_AlterDefaultPrivilegesPublic_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeAlterDefaultPrivileges, IsGrantToPublic: true}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleAlterDefaultPrivileges, policy.DecisionBlock)
}

func TestCheck_HighRiskRoleGrant_Blocks(t *testing.T) {
	stmt := parser.Statement{Type: parser.StmtTypeGrant, IsRoleMembershipGrant: true, GrantedRoles: []string{"pg_read_all_data"}}
	findings := Check(stmt, policy.DefaultPolicy())
	requireRuleDecision(t, findings, policy.RuleGrantHighRiskRoleMembership, policy.DecisionBlock)
}

func TestCheck_SelectAllProtectedTable_Warns(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"users"}}
	stmt := parser.Statement{Type: parser.StmtTypeSelect, Table: "users", SelectAll: true}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleSelectAllProtectedTable, policy.DecisionWarn)
}

func TestCheck_SelectWithoutLimitProtectedTable_Warns(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"users"}}
	stmt := parser.Statement{Type: parser.StmtTypeSelect, Table: "users", HasLimit: false}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleSelectWithoutLimitProtected, policy.DecisionWarn)
}

func TestCheck_CopyToProgramProtectedSource_Blocks(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"users"}}
	stmt := parser.Statement{Type: parser.StmtTypeCopy, Table: "users", CopyToProgram: true}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleCopyToStdoutOrProgram, policy.DecisionBlock)
}

func TestCheck_PolicyOverride_Allow(t *testing.T) {
	p := &policy.Policy{
		Dialect: policy.DialectPostgres,
		Rules:   map[string]string{policy.RuleDropTable: "allow"},
	}
	stmt := parser.Statement{Type: parser.StmtTypeDropTable, Table: "tmp"}
	findings := Check(stmt, p)
	requireRuleDecision(t, findings, policy.RuleDropTable, policy.DecisionAllow)
}

func TestCheck_NonProtectedTable_NoProtectedFinding(t *testing.T) {
	p := &policy.Policy{Dialect: policy.DialectPostgres, ProtectedTables: []string{"users"}}
	stmt := parser.Statement{Type: parser.StmtTypeUpdate, Table: "orders", HasWhere: true}
	findings := Check(stmt, p)
	for _, f := range findings {
		if f.Rule == policy.RuleWritesToProtectedTable {
			t.Fatal("did not expect protected-table finding")
		}
	}
}

func requireRuleDecision(t *testing.T, findings []Finding, rule string, decision policy.Decision) {
	t.Helper()
	for _, finding := range findings {
		if finding.Rule == rule {
			if finding.Decision != decision {
				t.Fatalf("rule %s decision want %s, got %s", rule, decision, finding.Decision)
			}
			return
		}
	}
	t.Fatalf("expected finding for rule %s, got %#v", rule, findings)
}
