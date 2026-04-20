// Package policy handles default policy, YAML loading, validation,
// and rule action lookup.
package policy

import (
	"strings"

	"github.com/ChimdumebiNebolisa/DBwall/internal/rulemeta"
)

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
	RuleDeleteWithoutWhere          = rulemeta.RuleDeleteWithoutWhere
	RuleUpdateWithoutWhere          = rulemeta.RuleUpdateWithoutWhere
	RuleDropTable                   = rulemeta.RuleDropTable
	RuleDropColumn                  = rulemeta.RuleDropColumn
	RuleWritesToProtectedTable      = rulemeta.RuleWritesToProtectedTable
	RuleDeleteTrivialWhere          = rulemeta.RuleDeleteTrivialWhere
	RuleUpdateTrivialWhere          = rulemeta.RuleUpdateTrivialWhere
	RuleTruncateTable               = rulemeta.RuleTruncateTable
	RuleDropSchema                  = rulemeta.RuleDropSchema
	RuleDropDatabase                = rulemeta.RuleDropDatabase
	RuleAlterDropSafetyConstraint   = rulemeta.RuleAlterDropSafetyConstraint
	RuleGrantToPublicProtected      = rulemeta.RuleGrantToPublicProtected
	RuleAlterDefaultPrivileges      = rulemeta.RuleAlterDefaultPrivileges
	RuleGrantHighRiskRoleMembership = rulemeta.RuleGrantHighRiskRoleMembership
	RuleSelectAllProtectedTable     = rulemeta.RuleSelectAllProtectedTable
	RuleSelectWithoutLimitProtected = rulemeta.RuleSelectWithoutLimitProtected
	RuleCopyToStdoutOrProgram       = rulemeta.RuleCopyToStdoutOrProgram
)

// Policy holds dialect, protected tables, and per-rule decisions.
type Policy struct {
	Dialect          Dialect           `yaml:"dialect" json:"dialect"`
	ProtectedTables  []string          `yaml:"protected_tables" json:"protected_tables"`
	ProtectedSchemas []string          `yaml:"protected_schemas" json:"protected_schemas"`
	ProtectedRoles   []string          `yaml:"protected_roles" json:"protected_roles"`
	Rules            map[string]string `yaml:"rules" json:"rules"` // rule name -> "allow"|"warn"|"block"
}

// DefaultPolicy returns the built-in v1 default policy.
func DefaultPolicy() *Policy {
	defaults := map[string]string{}
	for _, ruleID := range rulemeta.IDs() {
		defaults[ruleID] = rulemeta.MustGet(ruleID).DefaultDecision
	}
	return &Policy{
		Dialect:          DialectPostgres,
		ProtectedTables:  nil,
		ProtectedSchemas: nil,
		ProtectedRoles:   nil,
		Rules:            defaults,
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
	if rule, ok := rulemeta.Get(ruleName); ok {
		return Decision(rule.DefaultDecision)
	}
	return DecisionWarn
}

// IsProtectedTable returns true if tableName is explicitly protected or its schema is protected.
func (p *Policy) IsProtectedTable(tableName string) bool {
	if p == nil {
		return false
	}
	name := normalizeName(tableName)
	if name == "" {
		return false
	}
	leaf := name
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		leaf = name[idx+1:]
	}
	for _, t := range p.ProtectedTables {
		protected := normalizeName(t)
		if protected == name || protected == leaf {
			return true
		}
	}
	return p.IsProtectedSchema(schemaFromName(name))
}

// IsProtectedSchema returns true when a schema is marked protected.
func (p *Policy) IsProtectedSchema(schema string) bool {
	if p == nil {
		return false
	}
	name := normalizeName(schema)
	if name == "" {
		return false
	}
	for _, s := range p.ProtectedSchemas {
		if normalizeName(s) == name {
			return true
		}
	}
	return false
}

// IsProtectedRole returns true when a role is marked protected.
func (p *Policy) IsProtectedRole(role string) bool {
	if p == nil {
		return false
	}
	name := normalizeName(role)
	if name == "" {
		return false
	}
	for _, r := range p.ProtectedRoles {
		if normalizeName(r) == name {
			return true
		}
	}
	return false
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	parts := strings.Split(name, ".")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		parts[i] = strings.ToLower(part)
	}
	return strings.Join(parts, ".")
}

func schemaFromName(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[:idx]
	}
	return ""
}
