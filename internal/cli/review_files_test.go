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

func TestReviewFiles_EmptyPathsSARIFHasOneRunZeroResults(t *testing.T) {
	stdout := captureStdout(t, func() {
		code := ReviewFiles(nil, "", "sarif")
		if code != ExitAllow {
			t.Fatalf("want exit %d, got %d", ExitAllow, code)
		}
	})

	uris, resultsLen, runsLen := decodeSARIFArtifactURIs(t, stdout)
	if runsLen != 1 {
		t.Fatalf("want exactly 1 SARIF run for empty input, got %d\n%s", runsLen, stdout)
	}
	if resultsLen != 0 {
		t.Fatalf("want 0 SARIF results for empty input, got %d uris=%v\n%s", resultsLen, uris, stdout)
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

	uris, _, runsLen := decodeSARIFArtifactURIs(t, stdout)
	if runsLen != 1 {
		t.Fatalf("want 1 SARIF run, got %d", runsLen)
	}
	if !sarifURIsContainPath(uris, a) || !sarifURIsContainPath(uris, b) {
		t.Fatalf("SARIF artifactLocation.uri values missing expected paths\nwant: %q and %q\ngot:  %#v", a, b, uris)
	}
}

func TestReviewFiles_SARIFPathWithSpaces(t *testing.T) {
	dir := t.TempDir()
	spaced := filepath.Join(dir, "my migrations", "risk file.sql")
	if err := os.MkdirAll(filepath.Dir(spaced), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spaced, []byte("DELETE FROM users;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := captureStdout(t, func() {
		code := ReviewFiles([]string{spaced}, "", "sarif")
		if code != ExitBlock {
			t.Fatalf("want exit %d, got %d", ExitBlock, code)
		}
	})

	uris, _, _ := decodeSARIFArtifactURIs(t, stdout)
	if !sarifURIsContainPath(uris, spaced) {
		t.Fatalf("SARIF missing path-with-spaces uri\nwant: %q\ngot:  %#v", spaced, uris)
	}
}

func TestReviewFiles_SARIFRelativeSlashPaths(t *testing.T) {
	dir := t.TempDir()
	relDir := filepath.Join(dir, "migrations")
	if err := os.MkdirAll(relDir, 0o755); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(relDir, "001.sql")
	if err := os.WriteFile(abs, []byte("TRUNCATE t;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	// Forward-slash relative path is the portable cross-platform form used in CI.
	relSlash := "migrations/001.sql"
	stdout := captureStdout(t, func() {
		code := ReviewFiles([]string{relSlash}, "", "sarif")
		if code != ExitBlock {
			t.Fatalf("want exit %d, got %d", ExitBlock, code)
		}
	})
	uris, _, _ := decodeSARIFArtifactURIs(t, stdout)
	if !sarifURIsContainPath(uris, relSlash) {
		t.Fatalf("SARIF missing relative slash path\nwant: %q\ngot:  %#v", relSlash, uris)
	}
}

type sarifLogView struct {
	Runs []struct {
		Results []struct {
			Locations []struct {
				PhysicalLocation struct {
					ArtifactLocation struct {
						URI string `json:"uri"`
					} `json:"artifactLocation"`
				} `json:"physicalLocation"`
			} `json:"locations"`
		} `json:"results"`
	} `json:"runs"`
}

func decodeSARIFArtifactURIs(t *testing.T, raw string) (uris []string, resultsLen int, runsLen int) {
	t.Helper()
	var log sarifLogView
	if err := json.Unmarshal([]byte(raw), &log); err != nil {
		t.Fatalf("parse SARIF JSON: %v\n%s", err, raw)
	}
	runsLen = len(log.Runs)
	for _, run := range log.Runs {
		resultsLen += len(run.Results)
		for _, result := range run.Results {
			for _, loc := range result.Locations {
				uris = append(uris, loc.PhysicalLocation.ArtifactLocation.URI)
			}
		}
	}
	return uris, resultsLen, runsLen
}

func sarifURIsContainPath(uris []string, want string) bool {
	for _, uri := range uris {
		if uri == want {
			return true
		}
		// Compare cleaned forms so Windows short-path vs long-path differences
		// and slash normalization do not create brittle string matches.
		if filepath.Clean(uri) == filepath.Clean(want) {
			return true
		}
	}
	return false
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
