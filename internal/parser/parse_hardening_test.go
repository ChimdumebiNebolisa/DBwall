package parser

import (
	"strings"
	"testing"
)

// Regression tests for adversarial audit fixes. These run in BOTH build modes
// (core and full); mode-specific divergence assertions live in the
// divergence_*_test.go files.

func TestParse_UTF8BOMStripped(t *testing.T) {
	stmts, err := Parse("\uFEFFDELETE FROM users;")
	if err != nil {
		t.Fatalf("BOM-prefixed input should parse: %v", err)
	}
	if len(stmts) != 1 || stmts[0].Table != "users" || stmts[0].Type != StmtTypeDelete {
		t.Fatalf("unexpected statements: %#v", stmts)
	}
}

func TestParse_NonASCIIIdentifierPreserved(t *testing.T) {
	for _, sql := range []string{"DELETE FROM café;", `DELETE FROM "café";`} {
		stmts, err := Parse(sql)
		if err != nil {
			t.Fatalf("%q: %v", sql, err)
		}
		if got := stmts[0].Table; got != "café" && got != `"café"` {
			t.Fatalf("%q: table mangled: %q", sql, got)
		}
		if stmts[0].Table == "" || strings.ContainsRune(stmts[0].Table, '\uFFFD') {
			t.Fatalf("%q: replacement character in table name %q", sql, stmts[0].Table)
		}
	}
}

func TestParse_SubqueryWhereDoesNotBoundUpdate(t *testing.T) {
	// Audit finding F-003: a WHERE inside a SET subquery must not satisfy the
	// outer UPDATE bounding check.
	stmts, err := Parse("UPDATE accounts SET balance = (SELECT 0 WHERE TRUE);")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if stmts[0].HasWhere {
		t.Fatal("subquery WHERE must not count as outer UPDATE WHERE")
	}
	if stmts[0].Predicate != PredicateAbsent && CoverageMode() == "full" {
		t.Fatalf("full mode predicate want absent, got %s", stmts[0].Predicate)
	}
}

func TestParse_SubqueryWhereDoesNotBoundDelete(t *testing.T) {
	stmts, err := Parse("DELETE FROM logs WHERE id IN (SELECT id FROM keep_ids);")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !stmts[0].HasWhere {
		t.Fatal("top-level WHERE IN subquery must still count as bounded")
	}
}

func TestParse_SubqueryLimitDoesNotBoundOuterSelect(t *testing.T) {
	// Audit finding F-008: an inner LIMIT belongs to the subquery only.
	sql := "SELECT * FROM payments WHERE id IN (SELECT id FROM allowed_ids LIMIT 100);"
	stmts, err := Parse(sql)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if stmts[0].HasLimit {
		t.Fatal("inner subquery LIMIT must not mark the outer SELECT bounded")
	}
}

func TestParse_FetchFirstCountsAsLimit(t *testing.T) {
	// Audit finding F-010.
	for _, sql := range []string{
		"SELECT * FROM payments FETCH FIRST 10 ROWS ONLY;",
		"SELECT * FROM payments FETCH NEXT 5 ROWS ONLY;",
		"SELECT * FROM payments ORDER BY id FETCH FIRST 10 ROWS WITH TIES;",
	} {
		stmts, err := Parse(sql)
		if err != nil {
			t.Fatalf("%q: %v", sql, err)
		}
		if !stmts[0].HasLimit {
			t.Fatalf("%q: FETCH form should count as a limit", sql)
		}
	}
}

func TestParse_MultiObjectDropKeepsAllRelations(t *testing.T) {
	// Audit finding F-004: core parsed only the first object.
	stmts, err := Parse("DROP TABLE IF EXISTS finance.payments, public.audit_log CASCADE;")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	names := map[string]bool{}
	for _, rel := range stmts[0].Relations {
		names[rel.QualifiedName()] = true
	}
	if !names["finance.payments"] || !names["public.audit_log"] {
		t.Fatalf("want both drop targets, got %#v", stmts[0].Relations)
	}
}

func TestParse_MultiObjectTruncateKeepsAllRelations(t *testing.T) {
	stmts, err := Parse("TRUNCATE finance.payments, audit_log;")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found := map[string]int{}
	for _, rel := range stmts[0].Relations {
		found[rel.QualifiedName()]++
	}
	if found["finance.payments"] != 1 || found["audit_log"] != 1 {
		t.Fatalf("want both truncate targets exactly once each, got %#v", stmts[0].Relations)
	}
}

func TestParse_MultiObjectGrantKeepsAllTargets(t *testing.T) {
	// Audit finding F-004: the worst shape was GRANT ... ON TABLE a, b TO PUBLIC
	// silently losing the protected second relation in core builds.
	stmts, err := Parse("GRANT SELECT ON TABLE orders, users TO PUBLIC;")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	targets := map[string]bool{}
	for _, rel := range stmts[0].TargetRelations() {
		targets[rel.QualifiedName()] = true
	}
	if !targets["orders"] || !targets["users"] {
		t.Fatalf("want grant targets orders and users, got %#v (%#v)", stmts[0].Relations, stmts[0].Grantees)
	}
	if !stmts[0].IsGrantToPublic {
		t.Fatal("expected grant to PUBLIC detection")
	}
}

func TestParse_GrantOnDatabaseDoesNotFabricateRelation(t *testing.T) {
	// Audit finding F-018: core used to invent a target relation named "database".
	stmts, err := Parse("GRANT CONNECT ON DATABASE appdb TO PUBLIC;")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, rel := range stmts[0].Relations {
		name := rel.QualifiedName()
		if name == "database" || name == "appdb" {
			t.Fatalf("database grants must not become table-like relations, got %#v", stmts[0].Relations)
		}
	}
	if stmts[0].Object != "appdb" {
		t.Fatalf("object want appdb, got %q", stmts[0].Object)
	}
}

func TestParse_RevokeIsRevoke(t *testing.T) {
	// Audit finding F-001: REVOKE must never be typed as GRANT.
	cases := []string{
		"REVOKE ALL ON TABLE users FROM PUBLIC;",
		"REVOKE pg_read_all_data FROM analyst;",
	}
	for _, sql := range cases {
		stmts, err := Parse(sql)
		if err != nil {
			t.Fatalf("%q: parse: %v", sql, err)
		}
		if stmts[0].Type != StmtTypeRevoke {
			t.Fatalf("%q: want type REVOKE, got %s", sql, stmts[0].Type)
		}
		if stmts[0].IsGrantToPublic {
			t.Fatalf("%q: revoke must not set IsGrantToPublic", sql)
		}
	}
}

func TestParse_CommentOnlyInputYieldsNoStatements(t *testing.T) {
	for _, sql := range []string{"", "   ", ";", "-- only a comment", "/* block */", "\n-- x\n;/* y */"} {
		stmts, err := Parse(sql)
		if err != nil {
			t.Fatalf("%q: comment-only input must not error: %v", sql, err)
		}
		if len(stmts) != 0 {
			t.Fatalf("%q: want zero statements, got %d", sql, len(stmts))
		}
	}
}

func TestParse_TrailingCommentAfterStatement(t *testing.T) {
	stmts, err := Parse("DELETE FROM logs WHERE id = 1;\n-- migration footer\n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("trailing comment created phantom statement: %d statements", len(stmts))
	}
}

func TestWhitespaceAndCommentInvariance(t *testing.T) {
	// Property: adding whitespace/comments must not change extracted semantics.
	cases := []string{
		"DELETE FROM users WHERE id = 1;",
		"DELETE FROM users;",
		"UPDATE t SET a = 1 WHERE b = 2;",
		"SELECT * FROM finance.payments LIMIT 5;",
		"SELECT count(*) FROM payments;",
		"INSERT INTO logs (id) VALUES (1);",
		"TRUNCATE audit_logs;",
		"DROP TABLE old_events;",
		"GRANT SELECT ON TABLE orders TO app_role;",
		"COPY staging TO STDOUT;",
	}
	type summary struct {
		Type      string
		Tables    []string
		Predicate PredicateAssessment
		HasLimit  bool
		SelectAll bool
	}
	summarize := func(st Statement) summary {
		return summary{
			Type:      string(st.Type),
			Tables:    st.AllRelationNames(),
			Predicate: st.Predicate,
			HasLimit:  st.HasLimit,
			SelectAll: st.SelectAll,
		}
	}
	for _, sql := range cases {
		base, err := Parse(sql)
		if err != nil {
			t.Fatalf("%q: base parse: %v", sql, err)
		}
		variants := []string{
			strings.ReplaceAll(strings.ReplaceAll(sql, " ", "\n\t"), ";", ""),
			strings.ReplaceAll(sql, " ", " /* c */ "),
			"-- lead\n" + strings.ReplaceAll(sql, ";", ";\n-- tail\n"),
		}
		for vi, variant := range variants {
			got, err := Parse(variant)
			if err != nil {
				t.Fatalf("%q variant %d (%q): %v", sql, vi, variant, err)
			}
			if len(got) != len(base) {
				t.Fatalf("%q variant %d: statement count changed", sql, vi)
			}
			for i := range base {
				want, gotSum := summarize(base[i]), summarize(got[i])
				if want.Type != gotSum.Type || strings.Join(want.Tables, ",") != strings.Join(gotSum.Tables, ",") ||
					want.Predicate != gotSum.Predicate || want.HasLimit != gotSum.HasLimit || want.SelectAll != gotSum.SelectAll {
					t.Fatalf("%q variant %d stmt %d: semantics drifted: %#v -> %#v", sql, vi, i, want, gotSum)
				}
			}
		}
	}
}
