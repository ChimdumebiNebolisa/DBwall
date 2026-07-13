//go:build cgo

package parser

import "testing"

func TestDivergence_JoinOnlyFullMode(t *testing.T) {
	stmts, err := Parse(`SELECT * FROM orders o JOIN users u ON o.user_id = u.id;`)
	if err != nil {
		t.Fatal(err)
	}
	if CoverageMode() != "full" {
		t.Fatal("this test requires full mode")
	}
	if !containsAll(stmts[0].AllRelationNames(), "orders", "users") {
		t.Fatalf("full mode should see both join relations, got %#v", stmts[0].Relations)
	}
}

func TestDivergence_CopySelectOnlyFullMode(t *testing.T) {
	stmts, err := Parse(`COPY (SELECT id FROM users) TO STDOUT;`)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRelation(stmts[0], "users", RelationRead) {
		t.Fatalf("full mode should resolve COPY subquery source, got %#v", stmts[0].Relations)
	}
}
