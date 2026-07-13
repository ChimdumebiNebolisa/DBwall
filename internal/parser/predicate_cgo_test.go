//go:build cgo

package parser

import "testing"

func TestPredicate_AlwaysTrueCases(t *testing.T) {
	cases := []string{
		`DELETE FROM users WHERE TRUE;`,
		`DELETE FROM users WHERE (1 = 1);`,
		`UPDATE accounts SET disabled = true WHERE 'a' = 'a';`,
		`DELETE FROM users WHERE TRUE OR id = 5;`,
		`DELETE FROM users WHERE id = 5 OR TRUE;`,
		`DELETE FROM users WHERE NOT FALSE;`,
		`DELETE FROM users WHERE 2 > 1;`,
		`DELETE FROM users WHERE 1 IS NOT DISTINCT FROM 1;`,
	}
	for _, sql := range cases {
		stmts, err := Parse(sql)
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		if stmts[0].Predicate != PredicateAlwaysTrue || !stmts[0].WhereTrivial {
			t.Fatalf("%s: want always_true, got %s", sql, stmts[0].Predicate)
		}
	}
}

func TestPredicate_NotTrivialCases(t *testing.T) {
	cases := []string{
		`DELETE FROM users WHERE id = id;`,
		`DELETE FROM users WHERE now() = now();`,
		`DELETE FROM users WHERE nullable_column = nullable_column;`,
		`DELETE FROM users WHERE EXISTS (SELECT 1 FROM approved_users);`,
		`DELETE FROM users WHERE id = 5;`,
		`DELETE FROM users WHERE TRUE AND id = 5;`,
	}
	for _, sql := range cases {
		stmts, err := Parse(sql)
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		if stmts[0].Predicate == PredicateAlwaysTrue || stmts[0].WhereTrivial {
			t.Fatalf("%s: incorrectly classified as always_true", sql)
		}
		if stmts[0].Predicate == PredicateAbsent {
			t.Fatalf("%s: expected present predicate", sql)
		}
	}
}
