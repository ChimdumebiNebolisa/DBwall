package parser

import "testing"

// FuzzParse asserts the parser never panics on arbitrary input and that
// successful parses satisfy basic structural invariants.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"DELETE FROM users;",
		"UPDATE t SET a = 1 WHERE b = 2;",
		"SELECT * FROM finance.payments LIMIT 5;",
		"WITH del AS (DELETE FROM users RETURNING *) SELECT * FROM del;",
		"COPY (SELECT * FROM users) TO STDOUT;",
		"GRANT SELECT ON TABLE a, b TO PUBLIC;",
		"REVOKE ALL ON TABLE users FROM PUBLIC;",
		"DO $$ BEGIN EXECUTE 'x'; END $$;",
		"$tag$body; $tag$",
		"'unterminated",
		"/* unterminated",
		"\xff\xfe garbage",
		";", "-- c", "", "🚀🚀 DELETE FROM 🚀;",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, sql string) {
		stmts, err := Parse(sql)
		if err != nil {
			return
		}
		for _, st := range stmts {
			if st.StartLine < 1 {
				t.Fatalf("statement start line %d below 1 for input %q", st.StartLine, sql)
			}
			if st.Completeness == "" {
				t.Fatalf("completeness unset for %q", sql)
			}
			for _, rel := range st.Relations {
				if rel.QualifiedName() == "" {
					t.Fatalf("empty relation reference for %q", sql)
				}
			}
		}
	})
}
