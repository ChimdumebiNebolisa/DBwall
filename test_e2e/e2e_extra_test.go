package test_e2e

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIE2EExtra(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name      string
		args      []string
		exitCode  int
		contains  string
		checkJson bool
	}{
		{
			name:     "UPDATE without WHERE",
			args:     []string{"review-sql", "UPDATE users SET name='test';"},
			exitCode: 3,
			contains: "Decision: BLOCK",
		},
		{
			name:     "ALTER TABLE DROP COLUMN",
			args:     []string{"review-sql", "ALTER TABLE users DROP COLUMN name;"},
			exitCode: 3,
			contains: "Decision: BLOCK",
		},
		{
			name:     "missing policy file",
			args:     []string{"review-sql", "SELECT 1;", "--policy", "non_existent_policy.yaml"},
			exitCode: 1,
			contains: "no such file or directory",
		},
		{
			name:     "multi-statement input where at least one statement is unsafe",
			args:     []string{"review-sql", "SELECT 1; DELETE FROM users;"},
			exitCode: 3,
			contains: "Decision: BLOCK",
		},
		{
			name:      "JSON output format",
			args:      []string{"review-sql", "SELECT 1;", "--format", "json"},
			exitCode:  0,
			contains:  "allow",
			checkJson: true,
		},
		{
			name:      "JSON output format with block",
			args:      []string{"review-sql", "DELETE FROM users;", "--format", "json"},
			exitCode:  3,
			contains:  "block",
			checkJson: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(bin, tt.args...)
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

			if tt.checkJson {
				var js map[string]interface{}
				if err := json.Unmarshal(output, &js); err != nil {
					t.Errorf("Expected valid JSON output, but got error: %v, output: %s", err, string(output))
				}
			}
		})
	}
}
