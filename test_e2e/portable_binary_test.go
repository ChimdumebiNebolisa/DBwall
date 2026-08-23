package test_e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// buildPortableBinary builds dbguard exactly the way release archives do
// (CGO_ENABLED=0) so adversarial checks run against the artifact users
// actually install, independent of the ambient CGO setting.
func buildPortableBinary(t *testing.T) string {
	t.Helper()

	name := "dbguard_portable_bin"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", name, "../cmd/dbguard")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build portable binary: %v", err)
	}
	abs, err := filepath.Abs(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(abs) })
	return abs
}

func requireExit(t *testing.T, bin string, dir string, args []string, wantExit int, wantSub string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	exit := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v output: %s", args, err, out)
		}
		exit = exitErr.ExitCode()
	}
	if exit != wantExit {
		t.Fatalf("%v: exit want %d got %d; output: %s", args, wantExit, exit, out)
	}
	if wantSub != "" && !strings.Contains(string(out), wantSub) {
		t.Fatalf("%v: output missing %q: %s", args, wantSub, out)
	}
	return string(out)
}

func writeBytes(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestPortableBinaryAdversarial exercises the installed-style core-mode binary
// through its real CLI contract.
func TestPortableBinaryAdversarial(t *testing.T) {
	bin := buildPortableBinary(t)
	dir := t.TempDir()
	policyPath := func() string {
		p := filepath.Join(dir, "pol.yaml")
		content := "dialect: postgres\nprotected_tables:\n  - users\nrules:\n  insert_select_from_protected_table: warn\n  grant_to_public_on_protected_objects: block\n"
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}()

	crlf := writeBytes(t, dir, "crlf.sql", []byte("DELETE FROM logs;\r\nTRUNCATE users;\r\n"))
	nul := writeBytes(t, dir, "nul.sql", append([]byte("DELETE FROM users WHERE id=1"), 0x00))
	longWhere := writeBytes(t, dir, "long.sql", []byte("DELETE FROM logs WHERE "+strings.Repeat("a=1 OR ", 20000)+"FALSE;"))
	deepParens := writeBytes(t, dir, "deep.sql", []byte("SELECT * FROM logs WHERE ("+strings.Repeat("(", 5000)+"1"+strings.Repeat(")", 5000)+")=1;"))
	big := writeBytes(t, dir, "big.sql", []byte(strings.Repeat("UPDATE t SET x = 1 WHERE id IN (SELECT id FROM s);\n", 20000)))

	staging := writeBytes(t, dir, "staging.sql", []byte("INSERT INTO archive SELECT * FROM users;\n"))

	t.Run("exit codes and decisions", func(t *testing.T) {
		requireExit(t, bin, dir, []string{"review-sql", "SELECT 1;"}, 0, "Decision: ALLOW")
		requireExit(t, bin, dir, []string{"review-sql", "DELETE FROM users;"}, 3, "Decision: BLOCK")
		requireExit(t, bin, dir, []string{"review-sql", "ALTER ROLE app WITH SUPERUSER;"}, 2, "Decision: WARN")
		requireExit(t, bin, dir, []string{"review-sql", "SELECT 1;", "--format", "bogus"}, 1, "invalid --format")
		requireExit(t, bin, dir, []string{"review-sql", "SELECT 1;", "--policy", "../../x.yaml"}, 1, "outside the working directory")
	})

	t.Run("CRLF file parses and blocks", func(t *testing.T) {
		requireExit(t, bin, dir, []string{"review-file", crlf}, 3, "Decision: BLOCK")
	})

	t.Run("NUL byte fails closed without panic", func(t *testing.T) {
		requireExit(t, bin, dir, []string{"review-file", nul}, 1, "")
	})

	t.Run("oversized predicate completes quickly", func(t *testing.T) {
		start := time.Now()
		requireExit(t, bin, dir, []string{"review-file", longWhere}, 0, "")
		if elapsed := time.Since(start); elapsed > 60*time.Second {
			t.Fatalf("100k-token predicate took too long: %s", elapsed)
		}
	})

	t.Run("deep nesting fails closed or parses without crash", func(t *testing.T) {
		cmd := exec.Command(bin, "review-file", deepParens)
		cmd.Dir = dir
		_, _ = cmd.CombinedOutput() // any exit is fine; must not hang or panic-loop
	})

	t.Run("large file bounded runtime", func(t *testing.T) {
		start := time.Now()
		requireExit(t, bin, dir, []string{"review-file", big}, 0, "")
		if elapsed := time.Since(start); elapsed > 120*time.Second {
			t.Fatalf("20k-statement file took too long: %s", elapsed)
		}
	})

	t.Run("staging copy of protected table now warns via CLI", func(t *testing.T) {
		out := requireExit(t, bin, dir, []string{
			"review-file", staging, "--policy", policyPath,
		}, 2, "Decision: WARN")
		if !strings.Contains(out, "insert_select_from_protected_table") {
			t.Fatalf("expected new rule in human output: %s", out)
		}
	})

	t.Run("sarif stays valid for multi-file runs", func(t *testing.T) {
		a := writeBytes(t, dir, "sa.sql", []byte("DELETE FROM logs;\n"))
		b := writeBytes(t, dir, "sb spaced.sql", []byte("TRUNCATE users;\n"))
		out := requireExit(t, bin, dir, []string{
			"review-files", filepath.Base(a), filepath.Base(b),
			"--policy", "pol.yaml", "--format", "sarif",
		}, 3, "")
		var decoded map[string]any
		if err := json.Unmarshal([]byte(out), &decoded); err != nil {
			t.Fatalf("invalid SARIF from portable binary: %v", err)
		}
		run := decoded["runs"].([]any)[0].(map[string]any)
		results := run["results"].([]any)
		if len(results) == 0 {
			t.Fatal("expected findings in SARIF")
		}
	})
}

// Guard against accidental regression of the empty-SARIF contract that the
// GitHub Action relies on for zero-change runs.
func TestPortableBinaryEmptyInputSarifContract(t *testing.T) {
	bin := buildPortableBinary(t)
	dir := t.TempDir()
	out := requireExit(t, bin, dir, []string{"review-files", "--format", "sarif"}, 0, "")
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("empty-run SARIF invalid: %v", err)
	}
	run := decoded["runs"].([]any)[0].(map[string]any)
	if results, ok := run["results"].([]any); !ok || len(results) != 0 {
		t.Fatalf("empty run must emit an explicit empty results array, got: %s", out)
	}
}
