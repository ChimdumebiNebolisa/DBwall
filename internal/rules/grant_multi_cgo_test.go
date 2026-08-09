//go:build cgo

package rules

import (
	"strings"
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

func TestFullMode_GrantOrdersAndUsersToPublic(t *testing.T) {
	const sql = `GRANT SELECT ON TABLE orders, users TO PUBLIC;`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	names := stmts[0].AllRelationNames()
	if !containsAllNames(names, "orders", "users") {
		t.Fatalf("full mode must see both grant targets, got %#v", stmts[0].Relations)
	}
	if !stmts[0].IsGrantToPublic {
		t.Fatal("expected grant to public")
	}

	p := &policy.Policy{
		Dialect:         policy.DialectPostgres,
		ProtectedTables: []string{"orders", "users"},
	}
	findings := Check(stmts[0], p)
	var hitOrders, hitUsers bool
	for _, f := range findings {
		if f.Rule != policy.RuleGrantToPublicProtected {
			continue
		}
		if strings.Contains(f.Message, "orders") {
			hitOrders = true
		}
		if strings.Contains(f.Message, "users") {
			hitUsers = true
		}
	}
	if !hitOrders || !hitUsers {
		t.Fatalf("want protected-object findings for orders and users, got %#v", findings)
	}
}

func TestFullMode_GrantMultiOnlyUsersProtected(t *testing.T) {
	const sql = `GRANT SELECT ON TABLE orders, users TO PUBLIC;`
	stmts, err := parser.Parse(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	names := stmts[0].AllRelationNames()
	if !containsAllNames(names, "orders", "users") {
		t.Fatalf("full mode must see both grant targets, got %#v", stmts[0].Relations)
	}

	// Only the second relation is protected — core/first-target-only parsers miss this.
	p := &policy.Policy{
		Dialect:         policy.DialectPostgres,
		ProtectedTables: []string{"users"},
	}
	findings := Check(stmts[0], p)
	var hitUsers bool
	for _, f := range findings {
		if f.Rule == policy.RuleGrantToPublicProtected && strings.Contains(f.Message, "users") {
			hitUsers = true
		}
		if f.Rule == policy.RuleGrantToPublicProtected && strings.Contains(f.Message, "orders") {
			t.Fatalf("orders is not protected; unexpected finding: %#v", f)
		}
	}
	if !hitUsers {
		t.Fatalf("want finding for protected users (second GRANT target), got %#v", findings)
	}
}

func containsAllNames(have []string, want ...string) bool {
	set := map[string]struct{}{}
	for _, h := range have {
		set[h] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}
