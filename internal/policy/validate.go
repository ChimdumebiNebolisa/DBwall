package policy

import (
	"errors"
	"fmt"
)

// ErrUnsupportedDialect is returned when dialect is not postgres.
var ErrUnsupportedDialect = errors.New("unsupported dialect: only postgres is supported in v1")

// ValidRuleNames is the set of rule names that may appear in policy.
var ValidRuleNames = []string{
	RuleDeleteWithoutWhere,
	RuleUpdateWithoutWhere,
	RuleDropTable,
	RuleDropColumn,
	RuleWritesToProtectedTable,
}

var validRuleNamesMap = map[string]bool{
	RuleDeleteWithoutWhere:     true,
	RuleUpdateWithoutWhere:     true,
	RuleDropTable:              true,
	RuleDropColumn:             true,
	RuleWritesToProtectedTable: true,
}

// ValidDecisions are allowed values for rule actions.
var ValidDecisions = map[string]bool{
	"allow": true,
	"warn":  true,
	"block": true,
}

// Validate checks the policy and returns an error if invalid.
// If dialect is empty, it is treated as postgres. If dialect is set and not postgres, returns ErrUnsupportedDialect.
func Validate(p *Policy) error {
	if p == nil {
		return errors.New("policy is nil")
	}
	d := p.Dialect
	if d == "" {
		d = DialectPostgres
	}
	if d != DialectPostgres {
		return fmt.Errorf("%w (got %q)", ErrUnsupportedDialect, d)
	}
	if p.Rules != nil {
		for k, v := range p.Rules {
			if !ValidDecisions[v] {
				return fmt.Errorf("invalid rule decision for %q: %q (must be allow, warn, or block)", k, v)
			}
			if !validRuleNamesMap[k] {
				return fmt.Errorf("unknown rule name in policy: %q", k)
			}
		}
	}
	return nil
}
