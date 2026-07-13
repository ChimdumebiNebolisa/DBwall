// Package rules implements safety rules that operate on parsed statements.
// Each rule is independently testable.
package rules

import (
	"fmt"
	"slices"
	"sort"
	"strings"

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
	out := checkOne(stmt, p)
	for _, nested := range stmt.Nested {
		out = append(out, Check(nested, p)...)
	}
	return dedupeFindings(out)
}

func checkOne(stmt parser.Statement, p *policy.Policy) []Finding {
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
	add(checkAlterDefaultPrivilegesPublic(stmt, p))
	add(checkGrantHighRiskRoleMembership(stmt, p))
	out = append(out, checkGrantToPublicOnProtectedObjects(stmt, p)...)
	out = append(out, checkSelectAllFromProtectedTable(stmt, p)...)
	out = append(out, checkSelectWithoutLimitFromProtectedTable(stmt, p)...)
	out = append(out, checkCopyToStdoutOrProgramFromProtectedSource(stmt, p)...)
	out = append(out, checkWritesToProtectedTables(stmt, p)...)
	add(checkSemanticAnalysisIncomplete(stmt, p))
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
	if !stmt.DropConstraint && !stmt.DropNotNull {
		return nil
	}
	switch stmt.Type {
	case parser.StmtTypeAlterTable, parser.StmtTypeAlterTableDropCol:
	default:
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
	switch stmt.Type {
	case parser.StmtTypeDelete, parser.StmtTypeUpdate, parser.StmtTypeDropTable,
		parser.StmtTypeAlterTable, parser.StmtTypeAlterTableDropCol,
		parser.StmtTypeInsert, parser.StmtTypeTruncate:
	default:
		return nil
	}
	names := protectedMutationNames(stmt, p)
	if len(names) == 0 {
		return nil
	}
	decision := p.RuleDecision(policy.RuleWritesToProtectedTable)
	out := make([]Finding, 0, len(names))
	for _, name := range names {
		out = append(out, *newFinding(policy.RuleWritesToProtectedTable, decision, "Write to protected table: "+name))
	}
	return out
}

func checkGrantToPublicOnProtectedObjects(stmt parser.Statement, p *policy.Policy) []Finding {
	if stmt.Type != parser.StmtTypeGrant || !stmt.IsGrantToPublic || stmt.IsRoleMembershipGrant {
		return nil
	}
	decision := p.RuleDecision(policy.RuleGrantToPublicProtected)
	var out []Finding
	seen := map[string]struct{}{}
	add := func(name string) {
		if name == "" {
			return
		}
		if !(p.IsProtectedTable(name) || p.IsProtectedSchema(name)) {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, *newFinding(policy.RuleGrantToPublicProtected, decision, "GRANT exposes a protected object to PUBLIC: "+name))
	}
	for _, rel := range stmt.TargetRelations() {
		add(rel.QualifiedName())
		if rel.Schema != "" {
			if p.IsProtectedSchema(rel.Schema) {
				add(rel.Schema)
			}
		}
	}
	if len(stmt.Relations) == 0 {
		add(stmt.Table)
	}
	if stmt.Schema != "" {
		add(stmt.Schema)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Message < out[j].Message })
	return out
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
	roles := append([]string{}, stmt.GrantedRoles...)
	sort.Strings(roles)
	for _, role := range roles {
		if p.IsProtectedRole(role) || slices.Contains(highRiskBuiltinRoles, role) {
			return newFinding(policy.RuleGrantHighRiskRoleMembership, p.RuleDecision(policy.RuleGrantHighRiskRoleMembership), "GRANT assigns a high-risk role membership: "+role)
		}
	}
	return nil
}

func checkSelectAllFromProtectedTable(stmt parser.Statement, p *policy.Policy) []Finding {
	if stmt.Type != parser.StmtTypeSelect || !stmt.SelectAll {
		return nil
	}
	names := protectedReadNames(stmt, p)
	if len(names) == 0 {
		return nil
	}
	decision := p.RuleDecision(policy.RuleSelectAllProtectedTable)
	out := make([]Finding, 0, len(names))
	for _, name := range names {
		out = append(out, *newFinding(policy.RuleSelectAllProtectedTable, decision, "SELECT * reads every column from a protected table: "+name))
	}
	return out
}

func checkSelectWithoutLimitFromProtectedTable(stmt parser.Statement, p *policy.Policy) []Finding {
	if stmt.Type != parser.StmtTypeSelect || stmt.HasLimit {
		return nil
	}
	names := protectedReadNames(stmt, p)
	if len(names) == 0 {
		return nil
	}
	decision := p.RuleDecision(policy.RuleSelectWithoutLimitProtected)
	out := make([]Finding, 0, len(names))
	for _, name := range names {
		out = append(out, *newFinding(policy.RuleSelectWithoutLimitProtected, decision, "SELECT reads from a protected table without a LIMIT: "+name))
	}
	return out
}

func checkCopyToStdoutOrProgramFromProtectedSource(stmt parser.Statement, p *policy.Policy) []Finding {
	if stmt.Type != parser.StmtTypeCopy || (!stmt.CopyToStdout && !stmt.CopyToProgram) {
		return nil
	}
	names := protectedReadNames(stmt, p)
	if len(names) == 0 {
		return nil
	}
	decision := p.RuleDecision(policy.RuleCopyToStdoutOrProgram)
	out := make([]Finding, 0, len(names))
	for _, name := range names {
		out = append(out, *newFinding(policy.RuleCopyToStdoutOrProgram, decision, "COPY exports data from a protected source to STDOUT or PROGRAM: "+name))
	}
	return out
}

func checkSemanticAnalysisIncomplete(stmt parser.Statement, p *policy.Policy) *Finding {
	switch stmt.Completeness {
	case parser.AnalysisPartial, parser.AnalysisUnsupported:
	default:
		return nil
	}
	reasons := actionableIncompleteReasons(stmt.IncompleteReasons)
	if len(reasons) == 0 {
		// Core-mode reduced coverage is disclosed via coverage_mode, not this rule.
		return nil
	}
	decision := p.RuleDecision(policy.RuleSemanticAnalysisIncomplete)
	if decision == policy.DecisionWarn && shouldEscalateIncomplete(stmt) {
		if p.Rules == nil || p.Rules[policy.RuleSemanticAnalysisIncomplete] == "" || p.Rules[policy.RuleSemanticAnalysisIncomplete] == "warn" {
			decision = policy.DecisionBlock
		}
	}
	summary := strings.Join(reasons, "; ")
	if summary == "" {
		summary = stmt.IncompleteSummary()
	}
	return newFinding(policy.RuleSemanticAnalysisIncomplete, decision, fmt.Sprintf("Semantic analysis is %s: %s", stmt.Completeness, summary))
}

func actionableIncompleteReasons(reasons []string) []string {
	var out []string
	for _, reason := range reasons {
		switch reason {
		case "core_mode_token_parser", "unsupported_statement_in_core_mode":
			continue
		default:
			out = append(out, reason)
		}
	}
	return out
}

func shouldEscalateIncomplete(stmt parser.Statement) bool {
	if stmt.Completeness != parser.AnalysisPartial && stmt.Completeness != parser.AnalysisUnsupported {
		return false
	}
	switch stmt.Type {
	case parser.StmtTypeDelete, parser.StmtTypeUpdate, parser.StmtTypeInsert, parser.StmtTypeTruncate,
		parser.StmtTypeDropTable, parser.StmtTypeAlterTable, parser.StmtTypeAlterTableDropCol:
		for _, reason := range stmt.IncompleteReasons {
			if strings.Contains(reason, "relation") ||
				strings.Contains(reason, "from_item") ||
				strings.Contains(reason, "unsupported") ||
				strings.Contains(reason, "predicate_classification_unknown") {
				return true
			}
		}
		return stmt.Completeness == parser.AnalysisUnsupported
	case parser.StmtTypeOther:
		return false
	default:
		return false
	}
}

func protectedMutationNames(stmt parser.Statement, p *policy.Policy) []string {
	seen := map[string]struct{}{}
	var names []string
	add := func(name string) {
		if name == "" || !p.IsProtectedTable(name) {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for _, rel := range stmt.MutationRelations() {
		add(rel.QualifiedName())
	}
	// Backward-compatible fallback when Relations is empty but Table is populated.
	if len(stmt.Relations) == 0 {
		add(stmt.Table)
	}
	sort.Strings(names)
	return names
}

func protectedReadNames(stmt parser.Statement, p *policy.Policy) []string {
	seen := map[string]struct{}{}
	var names []string
	add := func(name string) {
		if name == "" || !p.IsProtectedTable(name) {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for _, rel := range stmt.ReadRelations() {
		add(rel.QualifiedName())
	}
	if len(stmt.Relations) == 0 {
		add(stmt.Table)
	}
	sort.Strings(names)
	return names
}

func dedupeFindings(in []Finding) []Finding {
	if len(in) <= 1 {
		return in
	}
	type key struct{ rule, message string }
	seen := map[key]struct{}{}
	var out []Finding
	for _, f := range in {
		k := key{rule: f.Rule, message: f.Message}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, f)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Rule != out[j].Rule {
			return out[i].Rule < out[j].Rule
		}
		return out[i].Message < out[j].Message
	})
	return out
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
