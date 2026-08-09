package test_e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIReviewFilesMultiFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	a := filepath.Join(dir, "safe.sql")
	b := filepath.Join(dir, "unsafe.sql")
	if err := os.WriteFile(a, []byte("SELECT 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("DELETE FROM users;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "review-files", a, b, "--format", "json")
	out, err := cmd.CombinedOutput()
	var code int
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run: %v (%s)", err, out)
		}
		code = exitErr.ExitCode()
	}
	if code != 3 {
		t.Fatalf("want exit 3, got %d output=%s", code, out)
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("json: %v (%s)", err, out)
	}
	if payload["decision"] != "block" {
		t.Fatalf("want block, got %#v", payload["decision"])
	}
}

func TestCLIReviewFilesGrantMultiProtected(t *testing.T) {
	bin := buildBinary(t)
	modeCmd := exec.Command(bin, "review-sql", "SELECT 1;", "--format", "json")
	modeOut, err := modeCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("coverage probe failed: %v (%s)", err, modeOut)
	}
	var modePayload map[string]any
	if err := json.Unmarshal(modeOut, &modePayload); err != nil {
		t.Fatalf("coverage json: %v (%s)", err, modeOut)
	}
	if modePayload["coverage_mode"] != "full" {
		t.Skip("grant multi-object regression requires full coverage mode")
	}

	cmd := exec.Command(bin, "review-files", "testdata/grant_multi.sql", "--policy", "testdata/grant_multi_policy.yaml", "--format", "json")
	out, err := cmd.CombinedOutput()
	var code int
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run: %v (%s)", err, out)
		}
		code = exitErr.ExitCode()
	}
	if code != 3 {
		t.Fatalf("want exit 3, got %d output=%s", code, out)
	}
	text := string(out)
	// Policy protects only users. If full mode only saw the first GRANT target
	// (orders), this would incorrectly allow — so users must appear in findings.
	if !strings.Contains(text, "users") {
		t.Fatalf("expected finding mentioning protected users, got %s", text)
	}
	if !strings.Contains(text, "grant_to_public_on_protected_objects") {
		t.Fatalf("expected grant rule, got %s", text)
	}
	if !strings.Contains(text, "GRANT exposes a protected object to PUBLIC: users") {
		t.Fatalf("expected users protected-object finding, got %s", text)
	}
}
