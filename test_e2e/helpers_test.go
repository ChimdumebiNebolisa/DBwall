package test_e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()

	name := "dbguard_test_bin"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", name, "../cmd/dbguard")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build dbguard: %v", err)
	}

	absName, err := filepath.Abs(name)
	if err != nil {
		t.Fatalf("Failed to resolve binary path: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Remove(absName)
	})

	return absName
}
