package parser

import "testing"

func TestTokenParserRemainsAvailable(t *testing.T) {
	stmt, err := parseStatementText("DELETE FROM users WHERE id = 1;", 3)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Type != StmtTypeDelete {
		t.Fatalf("want DELETE, got %s", stmt.Type)
	}
	if stmt.Table != "users" {
		t.Fatalf("want users, got %q", stmt.Table)
	}
	if stmt.StartLine != 3 {
		t.Fatalf("want start line 3, got %d", stmt.StartLine)
	}
	if !hasRole(stmt, "users", RelationWrite) {
		t.Fatalf("expected write relation, got %#v", stmt.Relations)
	}
	addRelationQualified(&stmt, "audit.logs", RelationRead)
	schema, name := splitQualified("public.users")
	if schema != "public" || name != "users" {
		t.Fatalf("splitQualified mismatch: %q %q", schema, name)
	}
}
