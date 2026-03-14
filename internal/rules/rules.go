// Package rules implements safety rules that operate on parsed statements.
// Each rule is independently testable.
package rules

import (
	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

// Check runs all v1 rules on the statement and returns any findings.
func Check(stmt parser.Statement, p *policy.Policy) []Finding {
	if p == nil {
		p = policy.DefaultPolicy()
	}
	var out []Finding
	if f := checkDeleteWithoutWhere(stmt, p); f != nil {
		out = append(out, *f)
	}
	if f := checkUpdateWithoutWhere(stmt, p); f != nil {
		out = append(out, *f)
	}
	if f := checkDropTable(stmt, p); f != nil {
		out = append(out, *f)
	}
	if f := checkDropColumn(stmt, p); f != nil {
		out = append(out, *f)
	}
	for _, f := range checkWritesToProtectedTables(stmt, p) {
		out = append(out, f)
	}
	return out
}

// checkDeleteWithoutWhere: DELETE with no WHERE clause.
func checkDeleteWithoutWhere(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeDelete || stmt.HasWhere {
		return nil
	}
	decision := p.RuleDecision(policy.RuleDeleteWithoutWhere)
	return &Finding{
		Rule:     policy.RuleDeleteWithoutWhere,
		Severity: "critical",
		Decision: decision,
		Message:  "DELETE statement has no WHERE clause",
	}
}

// checkUpdateWithoutWhere: UPDATE with no WHERE clause.
func checkUpdateWithoutWhere(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeUpdate || stmt.HasWhere {
		return nil
	}
	decision := p.RuleDecision(policy.RuleUpdateWithoutWhere)
	return &Finding{
		Rule:     policy.RuleUpdateWithoutWhere,
		Severity: "critical",
		Decision: decision,
		Message:  "UPDATE statement has no WHERE clause",
	}
}

// checkDropTable: DROP TABLE.
func checkDropTable(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeDropTable {
		return nil
	}
	decision := p.RuleDecision(policy.RuleDropTable)
	return &Finding{
		Rule:     policy.RuleDropTable,
		Severity: "critical",
		Decision: decision,
		Message:  "DROP TABLE statement",
	}
}

// checkDropColumn: ALTER TABLE ... DROP COLUMN.
func checkDropColumn(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeAlterTableDropCol {
		return nil
	}
	decision := p.RuleDecision(policy.RuleDropColumn)
	return &Finding{
		Rule:     policy.RuleDropColumn,
		Severity: "high",
		Decision: decision,
		Message:  "ALTER TABLE DROP COLUMN statement",
	}
}

// checkWritesToProtectedTables: write against a protected table.
func checkWritesToProtectedTables(stmt parser.Statement, p *policy.Policy) []Finding {
	if stmt.Table == "" {
		return nil
	}
	if !p.IsProtectedTable(stmt.Table) {
		return nil
	}
	// Write-affecting types
	switch stmt.Type {
	case parser.StmtTypeDelete, parser.StmtTypeUpdate, parser.StmtTypeDropTable,
		parser.StmtTypeAlterTableDropCol, parser.StmtTypeInsert:
		decision := p.RuleDecision(policy.RuleWritesToProtectedTable)
		return []Finding{{
			Rule:     policy.RuleWritesToProtectedTable,
			Severity: "medium",
			Decision: decision,
			Message:  "Write to protected table: " + stmt.Table,
		}}
	default:
		return nil
	}
}
