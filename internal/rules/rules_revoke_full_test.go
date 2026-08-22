//go:build cgo

package rules

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

// Audit finding F-001: full-mode REVOKE must be extracted as a revoke, not a
// grant, so privilege revocations are not blocked as expansions.
func TestFullMode_RevokeObjectNotGrant(t *testing.T) {
	if parser.CoverageMode() != "full" {
		t.Fatal("requires full mode")
	}
	stmts, err := parser.Parse("REVOKE ALL ON TABLE users FROM PUBLIC;")
	if err != nil {
		t.Fatal(err)
	}
	st := stmts[0]
	if st.Type != parser.StmtTypeRevoke {
		t.Fatalf("want REVOKE, got %s", st.Type)
	}
	if st.Completeness != parser.AnalysisComplete {
		t.Fatalf("want complete analysis, got %s (%v)", st.Completeness, st.IncompleteReasons)
	}
	p := policy.DefaultPolicy()
	p.ProtectedTables = []string{"users"}
	if findings := Check(st, p); len(findings) != 0 {
		t.Fatalf("revocation of protected table access must not produce findings: %#v", findings)
	}
}

func TestFullMode_RevokeRoleMembershipNotGrant(t *testing.T) {
	if parser.CoverageMode() != "full" {
		t.Fatal("requires full mode")
	}
	stmts, err := parser.Parse("REVOKE pg_read_all_data FROM analyst;")
	if err != nil {
		t.Fatal(err)
	}
	st := stmts[0]
	if st.Type != parser.StmtTypeRevoke || !st.IsRoleMembershipRevoke {
		t.Fatalf("want role-membership revoke, got %s membershipRevoke=%v", st.Type, st.IsRoleMembershipRevoke)
	}
	findings := Check(st, policy.DefaultPolicy())
	for _, f := range findings {
		if f.Rule == policy.RuleGrantHighRiskRoleMembership {
			t.Fatalf("revoking a high-risk role was reported as granting it: %#v", f)
		}
	}
}

// Audit finding F-002: an arm's own LIMIT bounds only that arm.
func TestFullMode_UnionArmLimitDoesNotBoundOtherArm(t *testing.T) {
	if parser.CoverageMode() != "full" {
		t.Fatal("requires full mode")
	}
	stmts, err := parser.Parse("(SELECT * FROM logs LIMIT 1) UNION ALL (SELECT * FROM users);")
	if err != nil {
		t.Fatal(err)
	}
	if stmts[0].HasLimit {
		t.Fatal("one bounded arm must not mark the whole union bounded")
	}
	p := policy.DefaultPolicy()
	p.ProtectedTables = []string{"users"}
	findings := Check(stmts[0], p)
	for _, f := range findings {
		if f.Rule == policy.RuleSelectWithoutLimitProtected {
			return // expected warn-level finding present
		}
	}
	t.Fatalf("expected select_without_limit_from_protected_table finding for unbounded arm, got %#v", findings)
}

func TestFullMode_AllArmsBoundedMeansBounded(t *testing.T) {
	if parser.CoverageMode() != "full" {
		t.Fatal("requires full mode")
	}
	stmts, err := parser.Parse("(SELECT * FROM logs LIMIT 1) UNION ALL (SELECT * FROM users LIMIT 1);")
	if err != nil {
		t.Fatal(err)
	}
	if !stmts[0].HasLimit {
		t.Fatal("all arms bounded should mark the union bounded")
	}
}
