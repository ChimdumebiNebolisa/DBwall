package test_e2e

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCLIE2EBasic(t *testing.T) {
	bin := buildBinary(t)

	tests := []struct {
		name     string
		args     []string
		exitCode int
		contains string
	}{
		{
			name:     "SELECT 1",
			args:     []string{"review-sql", "SELECT 1;"},
			exitCode: 0,
			contains: "Decision: ALLOW",
		},
		{
			name:     "DELETE without WHERE",
			args:     []string{"review-sql", "DELETE FROM users;"},
			exitCode: 3,
			contains: "Decision: BLOCK",
		},
		{
			name:     "DROP TABLE",
			args:     []string{"review-sql", "DROP TABLE users;"},
			exitCode: 3,
			contains: "Decision: BLOCK",
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
		})
	}
}
