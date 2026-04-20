package rulemeta

// Rule names shared across policy validation, rule evaluation, and reporting.
const (
	RuleDeleteWithoutWhere          = "delete_without_where"
	RuleUpdateWithoutWhere          = "update_without_where"
	RuleDropTable                   = "drop_table"
	RuleDropColumn                  = "drop_column"
	RuleWritesToProtectedTable      = "writes_to_protected_tables"
	RuleDeleteTrivialWhere          = "delete_trivial_where"
	RuleUpdateTrivialWhere          = "update_trivial_where"
	RuleTruncateTable               = "truncate_table"
	RuleDropSchema                  = "drop_schema"
	RuleDropDatabase                = "drop_database"
	RuleAlterDropSafetyConstraint   = "alter_table_drop_not_null_or_constraint"
	RuleGrantToPublicProtected      = "grant_to_public_on_protected_objects"
	RuleAlterDefaultPrivileges      = "alter_default_privileges_public"
	RuleGrantHighRiskRoleMembership = "grant_high_risk_role_membership"
	RuleSelectAllProtectedTable     = "select_all_from_protected_table"
	RuleSelectWithoutLimitProtected = "select_without_limit_from_protected_table"
	RuleCopyToStdoutOrProgram       = "copy_to_stdout_or_program_from_protected_source"
)

// Rule describes one stable rule and its reporting metadata.
type Rule struct {
	ID              string
	Title           string
	Category        string
	Severity        string
	DefaultDecision string
	Rationale       string
	Remediation     string
}

var catalog = []Rule{
	{
		ID:              RuleDeleteWithoutWhere,
		Title:           "DELETE without WHERE",
		Category:        "destructive_dml",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "An unbounded DELETE can remove every row from the target relation.",
		Remediation:     "Add a selective WHERE clause or split the operation into a reviewed migration.",
	},
	{
		ID:              RuleUpdateWithoutWhere,
		Title:           "UPDATE without WHERE",
		Category:        "destructive_dml",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "An unbounded UPDATE can overwrite every row in the target relation.",
		Remediation:     "Add a selective WHERE clause or run the change through a reviewed migration.",
	},
	{
		ID:              RuleDropTable,
		Title:           "DROP TABLE",
		Category:        "destructive_ddl",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "Dropping a table removes the relation definition and can cascade into data loss.",
		Remediation:     "Use a reviewed migration plan and verify the table is intentionally being retired.",
	},
	{
		ID:              RuleDropColumn,
		Title:           "ALTER TABLE DROP COLUMN",
		Category:        "destructive_ddl",
		Severity:        "high",
		DefaultDecision: "block",
		Rationale:       "Dropping a column permanently removes data and can break dependent code paths.",
		Remediation:     "Prefer phased deprecation and verified backfills before removing a column.",
	},
	{
		ID:              RuleWritesToProtectedTable,
		Title:           "Write to protected table",
		Category:        "destructive_dml",
		Severity:        "medium",
		DefaultDecision: "warn",
		Rationale:       "Protected relations hold high-value data and deserve an explicit review boundary.",
		Remediation:     "Confirm the target relation is intended and escalate to manual approval when needed.",
	},
	{
		ID:              RuleDeleteTrivialWhere,
		Title:           "DELETE with trivial WHERE",
		Category:        "destructive_dml",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "A trivial predicate such as WHERE TRUE behaves like an unbounded DELETE in practice.",
		Remediation:     "Replace the predicate with a selective filter tied to the intended records.",
	},
	{
		ID:              RuleUpdateTrivialWhere,
		Title:           "UPDATE with trivial WHERE",
		Category:        "destructive_dml",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "A trivial predicate such as WHERE 1=1 behaves like an unbounded UPDATE in practice.",
		Remediation:     "Replace the predicate with a selective filter tied to the intended rows.",
	},
	{
		ID:              RuleTruncateTable,
		Title:           "TRUNCATE TABLE",
		Category:        "destructive_dml",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "TRUNCATE removes all rows immediately and bypasses row-by-row safety checks.",
		Remediation:     "Use an explicit reviewed maintenance workflow before truncating data.",
	},
	{
		ID:              RuleDropSchema,
		Title:           "DROP SCHEMA",
		Category:        "destructive_ddl",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "Dropping a schema can remove many relations at once and frequently cascades.",
		Remediation:     "Review schema retirement explicitly and confirm every dependent object is accounted for.",
	},
	{
		ID:              RuleDropDatabase,
		Title:           "DROP DATABASE",
		Category:        "destructive_ddl",
		Severity:        "critical",
		DefaultDecision: "block",
		Rationale:       "Dropping a database is irreversible destructive administration.",
		Remediation:     "Handle database retirement through a separately approved operational runbook.",
	},
	{
		ID:              RuleAlterDropSafetyConstraint,
		Title:           "ALTER TABLE drops safety constraint",
		Category:        "destructive_ddl",
		Severity:        "high",
		DefaultDecision: "block",
		Rationale:       "Removing NOT NULL or table constraints weakens data integrity controls.",
		Remediation:     "Validate downstream invariants first and use a reviewed migration with rollback planning.",
	},
	{
		ID:              RuleGrantToPublicProtected,
		Title:           "GRANT to PUBLIC on protected object",
		Category:        "permissions",
		Severity:        "high",
		DefaultDecision: "block",
		Rationale:       "Granting access to PUBLIC broadens visibility to every role in the database.",
		Remediation:     "Grant only to explicit least-privilege roles and keep protected objects out of PUBLIC access.",
	},
	{
		ID:              RuleAlterDefaultPrivileges,
		Title:           "ALTER DEFAULT PRIVILEGES to PUBLIC",
		Category:        "permissions",
		Severity:        "high",
		DefaultDecision: "block",
		Rationale:       "Broad default privileges create a persistent exposure for future objects.",
		Remediation:     "Default privileges should target explicit least-privilege roles rather than PUBLIC.",
	},
	{
		ID:              RuleGrantHighRiskRoleMembership,
		Title:           "Grant high-risk role membership",
		Category:        "permissions",
		Severity:        "high",
		DefaultDecision: "block",
		Rationale:       "Granting high-risk built-in or protected roles can expand privileged access sharply.",
		Remediation:     "Review role membership manually and prefer narrowly scoped application roles.",
	},
	{
		ID:              RuleSelectAllProtectedTable,
		Title:           "SELECT * from protected table",
		Category:        "bulk_access",
		Severity:        "medium",
		DefaultDecision: "warn",
		Rationale:       "Selecting every column from a protected relation increases accidental data exposure risk.",
		Remediation:     "Project only the required columns and document the access need for protected data.",
	},
	{
		ID:              RuleSelectWithoutLimitProtected,
		Title:           "SELECT without LIMIT from protected table",
		Category:        "bulk_access",
		Severity:        "medium",
		DefaultDecision: "warn",
		Rationale:       "Unbounded reads from protected relations look like bulk extraction and are harder to review safely.",
		Remediation:     "Add a LIMIT or another narrow predicate to make the read scope explicit.",
	},
	{
		ID:              RuleCopyToStdoutOrProgram,
		Title:           "COPY to STDOUT or PROGRAM from protected source",
		Category:        "bulk_access",
		Severity:        "high",
		DefaultDecision: "block",
		Rationale:       "COPY TO STDOUT or PROGRAM is a direct bulk-exfiltration path for protected data.",
		Remediation:     "Use a reviewed export workflow with explicit approval, destination, and audit trail.",
	},
}

var catalogByID = func() map[string]Rule {
	out := make(map[string]Rule, len(catalog))
	for _, rule := range catalog {
		out[rule.ID] = rule
	}
	return out
}()

// All returns the full stable rule catalog.
func All() []Rule {
	out := make([]Rule, len(catalog))
	copy(out, catalog)
	return out
}

// Get returns the rule metadata for an id.
func Get(id string) (Rule, bool) {
	rule, ok := catalogByID[id]
	return rule, ok
}

// MustGet returns rule metadata or panics if the catalog is broken.
func MustGet(id string) Rule {
	rule, ok := Get(id)
	if !ok {
		panic("unknown rule id: " + id)
	}
	return rule
}

// IDs returns the valid rule ids for policy validation.
func IDs() []string {
	out := make([]string, 0, len(catalog))
	for _, rule := range catalog {
		out = append(out, rule.ID)
	}
	return out
}
