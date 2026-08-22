//go:build !cgo

package rules

import (
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

// Audit finding F-001 (core side): REVOKE parses to a revoke statement instead
// of failing the gate with a syntax error.
func TestCoreMode_RevokeParsesWithoutError(t *testing.T) {
	if parser.CoverageMode() != "core" {
		t.Fatal("requires core mode")
	}
	stmts, err := parser.Parse("REVOKE ALL ON TABLE users FROM PUBLIC;")
	if err != nil {
		t.Fatalf("revoke should parse in core mode: %v", err)
	}
	if stmts[0].Type != parser.StmtTypeRevoke {
		t.Fatalf("want REVOKE, got %s", stmts[0].Type)
	}
	p := policy.DefaultPolicy()
	p.ProtectedTables = []string{"users"}
	if findings := Check(stmts[0], p); len(findings) != 0 {
		t.Fatalf("revocation must not produce findings: %#v", findings)
	}
}

// Audit finding F-005 (core side): unsupported ALTER/DROP targets must surface
// the incomplete-analysis rule rather than silently allowing.
func TestCoreMode_UnsupportedAlterTargetWarns(t *testing.T) {
	if parser.CoverageMode() != "core" {
		t.Fatal("requires core mode")
	}
	for _, sql := range []string{
		"ALTER ROLE app WITH SUPERUSER;",
		"DROP TYPE custom_status;",
	} {
		stmts, err := parser.Parse(sql)
		if err != nil {
			t.Fatalf("%q: %v", sql, err)
		}
		findings := Check(stmts[0], policy.DefaultPolicy())
		found := false
		for _, f := range findings {
			if f.Rule == policy.RuleSemanticAnalysisIncomplete {
				found = true
			}
		}
		if !found {
			t.Fatalf("%q: expected semantic_analysis_incomplete finding, got %#v", sql, findings)
		}
	}
}

// Audit finding F-004 (core side): multi-object GRANT must keep the protected
// second relation so GRANT ... TO PUBLIC cannot silently pass.
func TestCoreMode_MultiObjectGrantBlocksProtectedSecondTarget(t *testing.T) {
	if parser.CoverageMode() != "core" {
		t.Fatal("requires core mode")
	}
	stmts, err := parser.Parse("GRANT SELECT ON TABLE orders, users TO PUBLIC;")
	if err != nil {
		t.Fatal(err)
	}
	p := policy.DefaultPolicy()
	p.ProtectedTables = []string{"users"}
	findings := Check(stmts[0], p)
	for _, f := range findings {
		if f.Rule == policy.RuleGrantToPublicProtected && f.Decision == policy.DecisionBlock {
			return
		}
	}
	t.Fatalf("expected grant_to_public_on_protected_objects block, got %#v", findings)
}
