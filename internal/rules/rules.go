// Package rules implements safety rules that operate on parsed statements.
// Each rule is independently testable.
package rules

import (
	"slices"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
	"github.com/ChimdumebiNebolisa/DBwall/internal/rulemeta"
)

var highRiskBuiltinRoles = []string{
	"pg_execute_server_program",
	"pg_monitor",
	"pg_read_all_data",
	"pg_read_server_files",
	"pg_signal_backend",
	"pg_write_all_data",
	"pg_write_server_files",
}

// Check runs all security rules on the statement and returns any findings.
func Check(stmt parser.Statement, p *policy.Policy) []Finding {
	if p == nil {
		p = policy.DefaultPolicy()
	}
	var out []Finding
	add := func(f *Finding) {
		if f != nil {
			out = append(out, *f)
		}
	}

	add(checkDeleteWithoutWhere(stmt, p))
	add(checkDeleteTrivialWhere(stmt, p))
	add(checkUpdateWithoutWhere(stmt, p))
	add(checkUpdateTrivialWhere(stmt, p))
	add(checkDropTable(stmt, p))
	add(checkDropSchema(stmt, p))
	add(checkDropDatabase(stmt, p))
	add(checkDropColumn(stmt, p))
	add(checkAlterDropSafetyConstraint(stmt, p))
	add(checkTruncateTable(stmt, p))
	add(checkGrantToPublicOnProtectedObjects(stmt, p))
	add(checkAlterDefaultPrivilegesPublic(stmt, p))
	add(checkGrantHighRiskRoleMembership(stmt, p))
	add(checkSelectAllFromProtectedTable(stmt, p))
	add(checkSelectWithoutLimitFromProtectedTable(stmt, p))
	add(checkCopyToStdoutOrProgramFromProtectedSource(stmt, p))
	for _, f := range checkWritesToProtectedTables(stmt, p) {
		out = append(out, f)
	}
	return out
}

func checkDeleteWithoutWhere(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeDelete || stmt.HasWhere {
		return nil
	}
	return newFinding(policy.RuleDeleteWithoutWhere, p.RuleDecision(policy.RuleDeleteWithoutWhere), "DELETE statement has no WHERE clause")
}

func checkDeleteTrivialWhere(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeDelete || !stmt.WhereTrivial {
		return nil
	}
	return newFinding(policy.RuleDeleteTrivialWhere, p.RuleDecision(policy.RuleDeleteTrivialWhere), "DELETE statement uses a trivial WHERE predicate")
}

func checkUpdateWithoutWhere(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeUpdate || stmt.HasWhere {
		return nil
	}
	return newFinding(policy.RuleUpdateWithoutWhere, p.RuleDecision(policy.RuleUpdateWithoutWhere), "UPDATE statement has no WHERE clause")
}

func checkUpdateTrivialWhere(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeUpdate || !stmt.WhereTrivial {
		return nil
	}
	return newFinding(policy.RuleUpdateTrivialWhere, p.RuleDecision(policy.RuleUpdateTrivialWhere), "UPDATE statement uses a trivial WHERE predicate")
}

func checkDropTable(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeDropTable {
		return nil
	}
	return newFinding(policy.RuleDropTable, p.RuleDecision(policy.RuleDropTable), "DROP TABLE statement")
}

func checkDropSchema(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeDropSchema {
		return nil
	}
	return newFinding(policy.RuleDropSchema, p.RuleDecision(policy.RuleDropSchema), "DROP SCHEMA statement")
}

func checkDropDatabase(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeDropDatabase {
		return nil
	}
	return newFinding(policy.RuleDropDatabase, p.RuleDecision(policy.RuleDropDatabase), "DROP DATABASE statement")
}

func checkDropColumn(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeAlterTableDropCol && !stmt.DropColumn {
		return nil
	}
	return newFinding(policy.RuleDropColumn, p.RuleDecision(policy.RuleDropColumn), "ALTER TABLE DROP COLUMN statement")
}

func checkAlterDropSafetyConstraint(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeAlterTable || (!stmt.DropConstraint && !stmt.DropNotNull) {
		return nil
	}
	return newFinding(policy.RuleAlterDropSafetyConstraint, p.RuleDecision(policy.RuleAlterDropSafetyConstraint), "ALTER TABLE removes a NOT NULL or table constraint")
}

func checkTruncateTable(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeTruncate {
		return nil
	}
	return newFinding(policy.RuleTruncateTable, p.RuleDecision(policy.RuleTruncateTable), "TRUNCATE statement")
}

func checkWritesToProtectedTables(stmt parser.Statement, p *policy.Policy) []Finding {
	if stmt.Table == "" || !p.IsProtectedTable(stmt.Table) {
		return nil
	}
	switch stmt.Type {
	case parser.StmtTypeDelete, parser.StmtTypeUpdate, parser.StmtTypeDropTable,
		parser.StmtTypeAlterTable, parser.StmtTypeAlterTableDropCol,
		parser.StmtTypeInsert, parser.StmtTypeTruncate:
		decision := p.RuleDecision(policy.RuleWritesToProtectedTable)
		return []Finding{*newFinding(policy.RuleWritesToProtectedTable, decision, "Write to protected table: "+stmt.Table)}
	default:
		return nil
	}
}

func checkGrantToPublicOnProtectedObjects(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeGrant || !stmt.IsGrantToPublic {
		return nil
	}
	if stmt.Table == "" && stmt.Schema == "" {
		return nil
	}
	if !p.IsProtectedTable(stmt.Table) && !p.IsProtectedSchema(stmt.Schema) {
		return nil
	}
	return newFinding(policy.RuleGrantToPublicProtected, p.RuleDecision(policy.RuleGrantToPublicProtected), "GRANT exposes a protected object to PUBLIC")
}

func checkAlterDefaultPrivilegesPublic(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeAlterDefaultPrivileges || !stmt.IsGrantToPublic {
		return nil
	}
	return newFinding(policy.RuleAlterDefaultPrivileges, p.RuleDecision(policy.RuleAlterDefaultPrivileges), "ALTER DEFAULT PRIVILEGES grants access to PUBLIC")
}

func checkGrantHighRiskRoleMembership(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeGrant || !stmt.IsRoleMembershipGrant {
		return nil
	}
	for _, role := range stmt.GrantedRoles {
		if p.IsProtectedRole(role) || slices.Contains(highRiskBuiltinRoles, role) {
			return newFinding(policy.RuleGrantHighRiskRoleMembership, p.RuleDecision(policy.RuleGrantHighRiskRoleMembership), "GRANT assigns a high-risk role membership: "+role)
		}
	}
	return nil
}

func checkSelectAllFromProtectedTable(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeSelect || !stmt.SelectAll || !p.IsProtectedTable(stmt.Table) {
		return nil
	}
	return newFinding(policy.RuleSelectAllProtectedTable, p.RuleDecision(policy.RuleSelectAllProtectedTable), "SELECT * reads every column from a protected table")
}

func checkSelectWithoutLimitFromProtectedTable(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeSelect || stmt.HasLimit || !p.IsProtectedTable(stmt.Table) {
		return nil
	}
	return newFinding(policy.RuleSelectWithoutLimitProtected, p.RuleDecision(policy.RuleSelectWithoutLimitProtected), "SELECT reads from a protected table without a LIMIT")
}

func checkCopyToStdoutOrProgramFromProtectedSource(stmt parser.Statement, p *policy.Policy) *Finding {
	if stmt.Type != parser.StmtTypeCopy || (!stmt.CopyToStdout && !stmt.CopyToProgram) || !p.IsProtectedTable(stmt.Table) {
		return nil
	}
	return newFinding(policy.RuleCopyToStdoutOrProgram, p.RuleDecision(policy.RuleCopyToStdoutOrProgram), "COPY exports data from a protected source to STDOUT or PROGRAM")
}

func newFinding(ruleID string, decision policy.Decision, message string) *Finding {
	meta := rulemeta.MustGet(ruleID)
	return &Finding{
		Rule:        ruleID,
		Title:       meta.Title,
		Category:    meta.Category,
		Severity:    meta.Severity,
		Decision:    decision,
		Message:     message,
		Rationale:   meta.Rationale,
		Remediation: meta.Remediation,
	}
}
