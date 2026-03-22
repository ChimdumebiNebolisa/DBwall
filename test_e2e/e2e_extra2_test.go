package test_e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIE2EExtra2(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "dbguard_test_bin", "../cmd/dbguard")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build dbguard: %v", err)
	}
	defer os.Remove("dbguard_test_bin")

	// Create policy file
	policyContent := `
dialect: postgres
protected_tables:
  - public.users
rules:
  writes_to_protected_tables: warn
`
	err := os.WriteFile("test_policy_schema.yaml", []byte(policyContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write policy file: %v", err)
	}
	defer os.Remove("test_policy_schema.yaml")

	// Create SQL file
	sqlContent := `UPDATE public.users SET role = 'viewer' WHERE id = 1;`
	err = os.WriteFile("test_query_schema.sql", []byte(sqlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write sql file: %v", err)
	}
	defer os.Remove("test_query_schema.sql")

	tests := []struct {
		name     string
		args     []string
		exitCode int
		contains string
	}{
		{
			name:     "schema-qualified and protected-table cases",
			args:     []string{"review-file", "test_query_schema.sql", "--policy", "test_policy_schema.yaml"},
			exitCode: 2,
			contains: "Decision: WARN",
		},
		{
			name:     "malformed SQL",
			args:     []string{"review-sql", "THIS IS NOT SQL;"},
			exitCode: 1,
			contains: "syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./dbguard_test_bin", tt.args...)
			output, err := cmd.CombinedOutput()

			var exitCode int
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("Failed to run cmd: %v, output: %s", err, string(output))
				}
			}

			if exitCode != tt.exitCode {
				t.Errorf("Expected exit code %d, got %d. Output: %s", tt.exitCode, exitCode, string(output))
			}

			if !strings.Contains(string(output), tt.contains) {
				t.Errorf("Expected output to contain %q, but got: %s", tt.contains, string(output))
			}
		})
	}
}
