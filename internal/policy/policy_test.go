package policy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPolicy(t *testing.T) {
	p := DefaultPolicy()
	if p.Dialect != DialectPostgres {
		t.Errorf("default dialect want postgres, got %q", p.Dialect)
	}
	if p.RuleDecision(RuleDeleteWithoutWhere) != DecisionBlock {
		t.Errorf("delete_without_where want block, got %s", p.RuleDecision(RuleDeleteWithoutWhere))
	}
	if p.RuleDecision(RuleWritesToProtectedTable) != DecisionWarn {
		t.Errorf("writes_to_protected_tables want warn, got %s", p.RuleDecision(RuleWritesToProtectedTable))
	}
}

func TestPolicy_RuleDecision(t *testing.T) {
	p := &Policy{
		Rules: map[string]string{
			RuleDeleteWithoutWhere: "warn",
			RuleDropTable:          "allow",
		},
	}
	if p.RuleDecision(RuleDeleteWithoutWhere) != DecisionWarn {
		t.Errorf("delete_without_where want warn, got %s", p.RuleDecision(RuleDeleteWithoutWhere))
	}
	if p.RuleDecision(RuleDropTable) != DecisionAllow {
		t.Errorf("drop_table want allow, got %s", p.RuleDecision(RuleDropTable))
	}
	// unset rule uses default
	if p.RuleDecision(RuleUpdateWithoutWhere) != DecisionBlock {
		t.Errorf("update_without_where (unset) want block, got %s", p.RuleDecision(RuleUpdateWithoutWhere))
	}
}

func TestPolicy_IsProtectedTable(t *testing.T) {
	p := &Policy{ProtectedTables: []string{"users", "payments"}}
	if !p.IsProtectedTable("users") {
		t.Error("users should be protected")
	}
	if p.IsProtectedTable("orders") {
		t.Error("orders should not be protected")
	}
	pNil := (*Policy)(nil)
	if pNil.IsProtectedTable("users") {
		t.Error("nil policy should not protect any table")
	}
}

func TestLoadFromBytes(t *testing.T) {
	yaml := `
dialect: postgres
protected_tables:
  - users
  - audit_logs
rules:
  delete_without_where: block
  writes_to_protected_tables: warn
`
	p, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}
	if p.Dialect != DialectPostgres {
		t.Errorf("dialect want postgres, got %q", p.Dialect)
	}
	if len(p.ProtectedTables) != 2 {
		t.Errorf("protected_tables want 2, got %d", len(p.ProtectedTables))
	}
	if !p.IsProtectedTable("users") {
		t.Error("users should be protected")
	}
}

func TestLoadFromBytes_InvalidYAML(t *testing.T) {
	_, err := LoadFromBytes([]byte("dialect: [ broken"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Need to work within CWD for current LoadFromFile implementation
	cwd, _ := os.Getwd()
	f := filepath.Join(cwd, "test_policy.yaml")
	if err := os.WriteFile(f, []byte("dialect: postgres\nprotected_tables: []\n"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	defer os.Remove(f)

	p, err := LoadFromFile(f)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if p.Dialect != DialectPostgres {
		t.Errorf("dialect want postgres, got %q", p.Dialect)
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("nonexistent_dbguard.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFromFile_PathTraversal(t *testing.T) {
	// Test that an absolute path outside CWD is blocked.
	_, err := LoadFromFile("/etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt (absolute path outside CWD)")
	}

	// Test that a relative path escaping CWD is blocked.
	_, err = LoadFromFile("../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt (relative path escaping CWD)")
	}
}

func TestValidate_Nil(t *testing.T) {
	if err := Validate(nil); err == nil {
		t.Fatal("expected error for nil policy")
	}
}

func TestValidate_UnsupportedDialect(t *testing.T) {
	p := &Policy{Dialect: "mysql"}
	err := Validate(p)
	if err == nil {
		t.Fatal("expected error for unsupported dialect")
	}
	if !errors.Is(err, ErrUnsupportedDialect) {
		t.Errorf("expected ErrUnsupportedDialect, got %v", err)
	}
}

func TestValidate_InvalidRuleDecision(t *testing.T) {
	p := &Policy{
		Dialect: DialectPostgres,
		Rules:   map[string]string{RuleDeleteWithoutWhere: "deny"},
	}
	err := Validate(p)
	if err == nil {
		t.Fatal("expected error for invalid decision")
	}
}

func TestValidate_UnknownRuleName(t *testing.T) {
	p := &Policy{
		Dialect: DialectPostgres,
		Rules:   map[string]string{"unknown_rule": "block"},
	}
	err := Validate(p)
	if err == nil {
		t.Fatal("expected error for unknown rule name")
	}
}

func TestValidate_Valid(t *testing.T) {
	p := DefaultPolicy()
	if err := Validate(p); err != nil {
		t.Fatalf("default policy should validate: %v", err)
	}
	p2 := &Policy{
		Dialect:         DialectPostgres,
		ProtectedTables: []string{"users"},
		Rules: map[string]string{
			RuleDeleteWithoutWhere: "block",
			RuleDropTable:          "warn",
		},
	}
	if err := Validate(p2); err != nil {
		t.Fatalf("valid policy should validate: %v", err)
	}
}
