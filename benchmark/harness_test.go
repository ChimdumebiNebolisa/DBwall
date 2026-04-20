package benchmark

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunProducesArtifacts(t *testing.T) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	tempDir := t.TempDir()
	jsonOut := filepath.Join(tempDir, "results.json")
	reportOut := filepath.Join(tempDir, "report.md")

	result, err := Run(context.Background(), Options{
		RepoRoot:  repoRoot,
		Manifest:  "benchmark/manifest.json",
		JSONOut:   jsonOut,
		ReportOut: reportOut,
	})
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if result.Metrics.TotalCases == 0 {
		t.Fatal("expected benchmark cases")
	}
	if _, err := os.Stat(jsonOut); err != nil {
		t.Fatalf("json output missing: %v", err)
	}
	if _, err := os.Stat(reportOut); err != nil {
		t.Fatalf("report output missing: %v", err)
	}

	data, err := os.ReadFile(jsonOut)
	if err != nil {
		t.Fatalf("read json output: %v", err)
	}
	var decoded RunResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode json output: %v", err)
	}
	if decoded.Metrics.TotalCases != result.Metrics.TotalCases {
		t.Fatalf("total cases mismatch: got %d want %d", decoded.Metrics.TotalCases, result.Metrics.TotalCases)
	}
}
