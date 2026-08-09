package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/report"
)

func TestReviewFiles_AggregatesAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	allowPath := filepath.Join(dir, "allow.sql")
	blockPath := filepath.Join(dir, "block.sql")
	if err := os.WriteFile(allowPath, []byte("SELECT 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blockPath, []byte("DELETE FROM users;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := captureStdout(t, func() {
		code := ReviewFiles([]string{allowPath, blockPath}, "", "json")
		if code != ExitBlock {
			t.Fatalf("want exit %d, got %d", ExitBlock, code)
		}
	})

	var out report.JSONOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout)
	}
	if out.Decision != "block" {
		t.Fatalf("want block, got %s", out.Decision)
	}
	if out.Summary.Statements != 2 {
		t.Fatalf("want 2 statements, got %d", out.Summary.Statements)
	}
	if len(out.Statements) != 2 {
		t.Fatalf("want 2 statement results, got %d", len(out.Statements))
	}
	if out.Statements[0].Location == nil || out.Statements[0].Location.Path != allowPath {
		t.Fatalf("unexpected first location: %#v", out.Statements[0].Location)
	}
	if out.Statements[1].Location == nil || out.Statements[1].Location.Path != blockPath {
		t.Fatalf("unexpected second location: %#v", out.Statements[1].Location)
	}
}

func TestReviewFiles_EmptyPathsAllow(t *testing.T) {
	stdout := captureStdout(t, func() {
		code := ReviewFiles(nil, "", "json")
		if code != ExitAllow {
			t.Fatalf("want exit %d, got %d", ExitAllow, code)
		}
	})
	if !strings.Contains(stdout, `"decision": "allow"`) {
		t.Fatalf("want allow decision, got %s", stdout)
	}
}

func TestReviewFiles_SARIFIncludesEachFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.sql")
	b := filepath.Join(dir, "b.sql")
	if err := os.WriteFile(a, []byte("DELETE FROM t1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("TRUNCATE t2;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout := captureStdout(t, func() {
		code := ReviewFiles([]string{a, b}, "", "sarif")
		if code != ExitBlock {
			t.Fatalf("want exit %d, got %d", ExitBlock, code)
		}
	})
	if !strings.Contains(stdout, a) || !strings.Contains(stdout, b) {
		t.Fatalf("SARIF should include both file paths, got %s", stdout)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if readErr != nil {
				done <- b.String()
				return
			}
		}
	}()

	fn()
	_ = w.Close()
	return <-done
}
