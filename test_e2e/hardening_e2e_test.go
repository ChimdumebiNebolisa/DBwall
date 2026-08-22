package test_e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Regression coverage for adversarial audit fixes, exercised through the CLI in
// whatever mode the test binary was built with (CI covers both modes).

func writeHardeningPolicy(dir string) (string, error) {
	path := filepath.Join(dir, "hardening_policy.yaml")
	content := "dialect: postgres\nprotected_tables:\n  - users\nrules:\n  delete_without_where: block\n  grant_to_public_on_protected_objects: block\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func TestCLIE2E_HardeningRegressions(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	if _, err := writeHardeningPolicy(dir); err != nil {
		t.Fatal(err)
	}

	bomFile := filepath.Join(dir, "bom.sql")
	if err := os.WriteFile(bomFile, append([]byte{0xEF, 0xBB, 0xBF}, []byte("DELETE FROM users WHERE id = 1;\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	multiGrantFile := filepath.Join(dir, "multi_grant.sql")
	if err := os.WriteFile(multiGrantFile, []byte("GRANT SELECT ON TABLE orders, users TO PUBLIC;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	revokeFile := filepath.Join(dir, "revoke.sql")
	if err := os.WriteFile(revokeFile, []byte("REVOKE ALL ON TABLE users FROM PUBLIC;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	subqueryWhereFile := filepath.Join(dir, "subquery_where.sql")
	if err := os.WriteFile(subqueryWhereFile, []byte("UPDATE accounts SET balance = (SELECT 0 WHERE TRUE);\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	commentOnlyFile := filepath.Join(dir, "comments.sql")
	if err := os.WriteFile(commentOnlyFile, []byte("-- header comment\n/* block */\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		args      []string
		wantExit  int
		wantSub   string
		checkJSON func(t *testing.T, body []byte)
	}{
		{
			name:     "BOM-prefixed file parses instead of failing",
			args:     []string{"review-file", bomFile},
			wantExit: 0,
			wantSub:  "Decision: ALLOW",
		},
		{
			name:     "multi-object GRANT to PUBLIC blocks even when only second object is protected",
			args:     []string{"review-file", multiGrantFile, "--policy", "hardening_policy.yaml"},
			wantExit: 3,
			wantSub:  "Decision: BLOCK",
		},
		{
			name:     "revocation of protected access is allowed",
			args:     []string{"review-file", revokeFile, "--policy", "hardening_policy.yaml"},
			wantExit: 0,
			wantSub:  "Decision: ALLOW",
		},
		{
			name:     "subquery WHERE does not bound an unbounded UPDATE",
			args:     []string{"review-file", subqueryWhereFile},
			wantExit: 3,
			wantSub:  "Decision: BLOCK",
		},
		{
			name:     "comment-only file is allow, not a tool error",
			args:     []string{"review-file", commentOnlyFile},
			wantExit: 0,
			wantSub:  "Decision: ALLOW",
		},
		{
			name:     "invalid format is rejected instead of silently using human output",
			args:     []string{"review-sql", "SELECT 1;", "--format", "xml"},
			wantExit: 1,
			wantSub:  "invalid --format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(bin, tt.args...)
			// Run from the fixture dir so the CWD-relative policy containment
			// check sees hardening_policy.yaml.
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			exitCode := 0
			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatalf("run: %v output: %s", err, output)
				}
				exitCode = exitErr.ExitCode()
			}
			if exitCode != tt.wantExit {
				t.Fatalf("exit code: want %d got %d; output: %s", tt.wantExit, exitCode, output)
			}
			if !strings.Contains(string(output), tt.wantSub) {
				t.Fatalf("output missing %q: %s", tt.wantSub, output)
			}
			if tt.checkJSON != nil {
				var decoded map[string]any
				if jerr := json.Unmarshal(output, &decoded); jerr != nil {
					t.Fatalf("invalid JSON: %v", jerr)
				}
				tt.checkJSON(t, output)
			}
		})
	}
}
