//go:build cgo

package parser

import (
	"testing"
)

func TestAST_MultiTableJoinReads(t *testing.T) {
	stmts, err := Parse(`SELECT * FROM users u JOIN payments p ON u.id = p.user_id;`)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 stmt, got %d", len(stmts))
	}
	names := stmts[0].AllRelationNames()
	if !containsAll(names, "users", "payments") {
		t.Fatalf("expected users and payments, got %#v", names)
	}
	if stmts[0].Completeness != AnalysisComplete {
		t.Fatalf("expected complete analysis, got %s (%v)", stmts[0].Completeness, stmts[0].IncompleteReasons)
	}
}

func TestAST_DeleteUsingAndUpdateFrom(t *testing.T) {
	del, err := Parse(`DELETE FROM users USING audit_logs WHERE users.id = audit_logs.user_id;`)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRelation(del[0], "users", RelationWrite) || !hasRelation(del[0], "audit_logs", RelationRead) {
		t.Fatalf("unexpected relations: %#v", del[0].Relations)
	}

	upd, err := Parse(`UPDATE accounts SET disabled = true FROM users WHERE accounts.user_id = users.id;`)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRelation(upd[0], "accounts", RelationWrite) || !hasRelation(upd[0], "users", RelationRead) {
		t.Fatalf("unexpected relations: %#v", upd[0].Relations)
	}
}

func TestAST_InsertSelectAndCopySelect(t *testing.T) {
	ins, err := Parse(`INSERT INTO archive SELECT * FROM users;`)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRelation(ins[0], "archive", RelationWrite) || !hasRelation(ins[0], "users", RelationRead) {
		t.Fatalf("unexpected relations: %#v", ins[0].Relations)
	}

	cp, err := Parse(`COPY (SELECT * FROM public.users) TO STDOUT;`)
	if err != nil {
		t.Fatal(err)
	}
	if !cp[0].CopyToStdout {
		t.Fatal("expected CopyToStdout")
	}
	if !hasRelation(cp[0], "users", RelationRead) {
		t.Fatalf("expected users read, got %#v", cp[0].Relations)
	}
	found := false
	for _, rel := range cp[0].Relations {
		if rel.Name == "users" && rel.Schema == "public" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected schema-qualified public.users, got %#v", cp[0].Relations)
	}
}

func TestAST_MultiObjectDropTruncateGrant(t *testing.T) {
	drop, err := Parse(`DROP TABLE a, public.b;`)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(drop[0].AllRelationNames(), "a", "public.b") {
		t.Fatalf("unexpected drop relations: %#v", drop[0].Relations)
	}

	tr, err := Parse(`TRUNCATE a, b;`)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(tr[0].AllRelationNames(), "a", "b") {
		t.Fatalf("unexpected truncate relations: %#v", tr[0].Relations)
	}

	gr, err := Parse(`GRANT SELECT ON TABLE a, b TO PUBLIC;`)
	if err != nil {
		t.Fatal(err)
	}
	if !gr[0].IsGrantToPublic {
		t.Fatal("expected grant to public")
	}
	if !containsAll(gr[0].AllRelationNames(), "a", "b") {
		t.Fatalf("unexpected grant relations: %#v", gr[0].Relations)
	}
}

func TestAST_CTENestedDelete(t *testing.T) {
	stmts, err := Parse(`WITH x AS (DELETE FROM users RETURNING *) SELECT * FROM x;`)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRelation(stmts[0], "users", RelationWrite) {
		t.Fatalf("expected nested write to users, got %#v", stmts[0].Relations)
	}
	if len(stmts[0].Nested) == 0 || stmts[0].Nested[0].Type != StmtTypeDelete {
		t.Fatalf("expected nested DELETE, got %#v", stmts[0].Nested)
	}
}

func TestAST_UnsupportedStatementMarksIncomplete(t *testing.T) {
	stmts, err := Parse(`CREATE INDEX ON users (id);`)
	if err != nil {
		t.Fatal(err)
	}
	if stmts[0].Type != StmtTypeOther {
		t.Fatalf("want OTHER, got %s", stmts[0].Type)
	}
	if stmts[0].Completeness != AnalysisUnsupported {
		t.Fatalf("want unsupported, got %s", stmts[0].Completeness)
	}
}

func TestAST_CopyFromProgramNotExport(t *testing.T) {
	stmts, err := Parse(`COPY users FROM PROGRAM 'cat';`)
	if err != nil {
		t.Fatal(err)
	}
	if stmts[0].CopyToProgram || stmts[0].CopyToStdout {
		t.Fatalf("COPY FROM PROGRAM must not look like export: %#v", stmts[0])
	}
	if !hasRelation(stmts[0], "users", RelationWrite) {
		t.Fatalf("expected write relation for COPY FROM, got %#v", stmts[0].Relations)
	}
}

func TestAST_DropSchemaDoesNotSetTable(t *testing.T) {
	stmts, err := Parse(`DROP SCHEMA reporting;`)
	if err != nil {
		t.Fatal(err)
	}
	if stmts[0].Table != "" {
		t.Fatalf("DROP SCHEMA must not invent Table, got %q", stmts[0].Table)
	}
	if stmts[0].Object != "reporting" || stmts[0].Schema != "reporting" {
		t.Fatalf("unexpected object/schema: %#v", stmts[0])
	}
}

func TestAST_NestedCTEInsideCopy(t *testing.T) {
	stmts, err := Parse(`COPY (WITH x AS (DELETE FROM users RETURNING id) SELECT * FROM x) TO STDOUT;`)
	if err != nil {
		t.Fatal(err)
	}
	foundDelete := false
	var walk func(Statement)
	walk = func(s Statement) {
		if s.Type == StmtTypeDelete && hasRelation(s, "users", RelationWrite) && !s.HasWhere {
			foundDelete = true
		}
		for _, n := range s.Nested {
			walk(n)
		}
	}
	walk(stmts[0])
	if !foundDelete {
		t.Fatalf("expected nested DELETE without WHERE under COPY, got %#v", stmts[0].Nested)
	}
}

func TestAST_ParentDoesNotInheritNestedLimit(t *testing.T) {
	stmts, err := Parse(`WITH c AS (SELECT * FROM users LIMIT 10) SELECT * FROM c;`)
	if err != nil {
		t.Fatal(err)
	}
	if stmts[0].HasLimit {
		t.Fatal("parent SELECT must not inherit CTE HasLimit")
	}
}

func TestAST_SetOpOuterLimit(t *testing.T) {
	stmts, err := Parse(`SELECT * FROM users UNION ALL SELECT * FROM orders LIMIT 10;`)
	if err != nil {
		t.Fatal(err)
	}
	if !stmts[0].HasLimit {
		t.Fatal("set-op outer LIMIT must be recognized")
	}
	if !containsAll(stmts[0].AllRelationNames(), "users", "orders") {
		t.Fatalf("unexpected relations: %#v", stmts[0].Relations)
	}
}

func TestAST_QuotedAndSchemaQualified(t *testing.T) {
	stmts, err := Parse(`SELECT id FROM public."Users";`)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, rel := range stmts[0].Relations {
		if rel.Schema == "public" && rel.Name == "Users" && rel.Role == RelationRead {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected public.Users read, got %#v", stmts[0].Relations)
	}
}


func hasRelation(stmt Statement, name string, role RelationRole) bool {
	for _, rel := range stmt.Relations {
		if rel.Role == role && (rel.Name == name || rel.QualifiedName() == name) {
			return true
		}
	}
	return false
}

func containsAll(have []string, want ...string) bool {
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
