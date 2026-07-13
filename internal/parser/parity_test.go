package parser

import "testing"

func TestCoreOrShared_BasicCompletenessMetadata(t *testing.T) {
	stmts, err := Parse(`DELETE FROM users WHERE id = 1;`)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 1 {
		t.Fatalf("want 1 statement, got %d", len(stmts))
	}
	if stmts[0].Table != "users" {
		t.Fatalf("table want users, got %q", stmts[0].Table)
	}
	if !hasRole(stmts[0], "users", RelationWrite) && CoverageMode() == "core" {
		// core path always populates Relations through setRelation
		t.Fatalf("expected write relation users, got %#v", stmts[0].Relations)
	}
	if CoverageMode() == "core" {
		if stmts[0].Completeness != AnalysisPartial {
			t.Fatalf("core mode should mark partial, got %s", stmts[0].Completeness)
		}
	}
	if CoverageMode() == "full" {
		if stmts[0].Completeness != AnalysisComplete {
			t.Fatalf("full mode should mark complete for simple delete, got %s (%v)", stmts[0].Completeness, stmts[0].IncompleteReasons)
		}
	}
}

func TestParity_SimpleStatements(t *testing.T) {
	sqls := []string{
		`DELETE FROM users;`,
		`UPDATE users SET x = 1 WHERE id = 1;`,
		`SELECT * FROM users;`,
		`INSERT INTO users (id) VALUES (1);`,
		`TRUNCATE users;`,
		`DROP TABLE users;`,
		`GRANT SELECT ON TABLE users TO PUBLIC;`,
		`COPY users TO PROGRAM 'cat';`,
	}
	for _, sql := range sqls {
		stmts, err := Parse(sql)
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		if len(stmts) != 1 {
			t.Fatalf("%s: want 1 stmt", sql)
		}
		if stmts[0].Type == "" || stmts[0].Type == StmtTypeOther {
			t.Fatalf("%s: unexpected type %s", sql, stmts[0].Type)
		}
		if stmts[0].Table == "" && stmts[0].Type != StmtTypeSelect {
			// select without from handled elsewhere; these all have tables
			if stmts[0].Type != StmtTypeGrant && stmts[0].Object == "" {
				t.Fatalf("%s: missing table/object", sql)
			}
		}
	}
}

func hasRole(stmt Statement, name string, role RelationRole) bool {
	for _, rel := range stmt.Relations {
		if rel.Role == role && (rel.Name == name || rel.QualifiedName() == name) {
			return true
		}
	}
	return false
}
