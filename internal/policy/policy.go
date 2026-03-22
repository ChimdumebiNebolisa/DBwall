// Package policy handles default policy, YAML loading, validation,
// and rule action lookup.
package policy

// Dialect is the SQL dialect. v1 supports only postgres.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
)

// Decision is the action for a rule: allow, warn, or block.
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionWarn  Decision = "warn"
	DecisionBlock Decision = "block"
)

// Rule names used in policy and rules package.
const (
	RuleDeleteWithoutWhere     = "delete_without_where"
	RuleUpdateWithoutWhere     = "update_without_where"
	RuleDropTable              = "drop_table"
	RuleDropColumn             = "drop_column"
	RuleWritesToProtectedTable = "writes_to_protected_tables"
)

// Policy holds dialect, protected tables, and per-rule decisions.
type Policy struct {
	Dialect         Dialect           `yaml:"dialect"`
	ProtectedTables []string          `yaml:"protected_tables"`
	Rules           map[string]string `yaml:"rules"` // rule name -> "allow"|"warn"|"block"
}

// DefaultPolicy returns the built-in v1 default policy.
func DefaultPolicy() *Policy {
	return &Policy{
		Dialect:         DialectPostgres,
		ProtectedTables: nil,
		Rules: map[string]string{
			RuleDeleteWithoutWhere:     string(DecisionBlock),
			RuleUpdateWithoutWhere:     string(DecisionBlock),
			RuleDropTable:              string(DecisionBlock),
			RuleDropColumn:             string(DecisionBlock),
			RuleWritesToProtectedTable: string(DecisionWarn),
		},
	}
}

// RuleDecision returns the decision for a rule name. Uses default if not set in policy.
func (p *Policy) RuleDecision(ruleName string) Decision {
	if p == nil || p.Rules == nil {
		return defaultDecisionForRule(ruleName)
	}
	s, ok := p.Rules[ruleName]
	if !ok {
		return defaultDecisionForRule(ruleName)
	}
	switch s {
	case "allow":
		return DecisionAllow
	case "warn":
		return DecisionWarn
	case "block":
		return DecisionBlock
	default:
		return defaultDecisionForRule(ruleName)
	}
}

func defaultDecisionForRule(ruleName string) Decision {
	switch ruleName {
	case RuleDeleteWithoutWhere, RuleUpdateWithoutWhere, RuleDropTable, RuleDropColumn:
		return DecisionBlock
	case RuleWritesToProtectedTable:
		return DecisionWarn
	default:
		return DecisionWarn
	}
}

// IsProtectedTable returns true if tableName is in the protected list (exact match, case-sensitive in v1).
func (p *Policy) IsProtectedTable(tableName string) bool {
	if p == nil || len(p.ProtectedTables) == 0 {
		return false
	}
	for _, t := range p.ProtectedTables {
		if t == tableName {
			return true
		}
	}
	return false
}
