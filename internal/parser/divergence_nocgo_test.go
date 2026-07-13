//go:build !cgo

package parser

import "testing"

func TestDivergence_CoreMissesJoinSecondTable(t *testing.T) {
	stmts, err := Parse(`SELECT * FROM orders o JOIN users u ON o.user_id = u.id;`)
	if err != nil {
		t.Fatal(err)
	}
	if CoverageMode() != "core" {
		t.Fatal("this test requires core mode")
	}
	// Token parser keeps only the first FROM identifier.
	if stmts[0].Table != "orders" {
		t.Fatalf("core mode table want orders, got %q", stmts[0].Table)
	}
	if containsName(stmts[0].AllRelationNames(), "users") {
		t.Fatalf("core mode should not claim join coverage for users, got %#v", stmts[0].Relations)
	}
}

func TestDivergence_CoreMissesCopySelectSourceDepth(t *testing.T) {
	stmts, err := Parse(`COPY (SELECT id FROM users) TO STDOUT;`)
	if err != nil {
		t.Fatal(err)
	}
	// Shared token path finds FROM users inside the parentheses, which is acceptable
	// parity for the simple shape, but completeness remains partial.
	if stmts[0].Completeness != AnalysisPartial {
		t.Fatalf("core mode should remain partial, got %s", stmts[0].Completeness)
	}
}

func containsName(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}
