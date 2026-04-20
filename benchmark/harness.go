package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"
)

type Manifest struct {
	PositiveDecision string         `json:"positive_decision"`
	Cases            []ManifestCase `json:"cases"`
}

type ManifestCase struct {
	ID               string `json:"id"`
	Category         string `json:"category"`
	SQLFile          string `json:"sql_file"`
	PolicyFile       string `json:"policy_file,omitempty"`
	ExpectedDecision string `json:"expected_decision"`
}

type Options struct {
	RepoRoot  string
	Manifest  string
	Binary    string
	JSONOut   string
	ReportOut string
}

type RunResult struct {
	RepoRoot         string       `json:"repo_root"`
	ManifestPath     string       `json:"manifest_path"`
	BinaryPath       string       `json:"binary_path"`
	BuildCommand     []string     `json:"build_command"`
	WarmupCommand    []string     `json:"warmup_command"`
	Cases            []CaseResult `json:"cases"`
	Metrics          Metrics      `json:"metrics"`
	PositiveDecision string       `json:"positive_decision"`
	Definitions      Definitions  `json:"definitions"`
}

type CaseResult struct {
	ID               string   `json:"id"`
	Category         string   `json:"category"`
	SQLFile          string   `json:"sql_file"`
	PolicyFile       string   `json:"policy_file,omitempty"`
	ExpectedDecision string   `json:"expected_decision"`
	ActualDecision   string   `json:"actual_decision"`
	ExactMatch       bool     `json:"exact_match"`
	FalsePositive    bool     `json:"false_positive"`
	FalseNegative    bool     `json:"false_negative"`
	ExitCode         int      `json:"exit_code"`
	RuntimeNanos     int64    `json:"runtime_nanos"`
	Command          []string `json:"command"`
	Stdout           string   `json:"stdout"`
	Stderr           string   `json:"stderr"`
}

type Metrics struct {
	TotalCases           int     `json:"total_cases"`
	CorrectBlocks        int     `json:"correct_blocks"`
	CorrectAllows        int     `json:"correct_allows"`
	CorrectWarns         int     `json:"correct_warns"`
	FalsePositives       int     `json:"false_positives"`
	FalseNegatives       int     `json:"false_negatives"`
	Precision            float64 `json:"precision"`
	Recall               float64 `json:"recall"`
	Accuracy             float64 `json:"accuracy"`
	AverageRuntimeNanos  int64   `json:"average_runtime_nanos"`
	AverageRuntimeMillis float64 `json:"average_runtime_ms"`
}

type Definitions struct {
	PrecisionPositiveClass string `json:"precision_positive_class"`
	AccuracyDefinition     string `json:"accuracy_definition"`
	RuntimeDefinition      string `json:"runtime_definition"`
}

type dbguardJSON struct {
	Decision string `json:"decision"`
}

func Run(ctx context.Context, opts Options) (RunResult, error) {
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return RunResult{}, fmt.Errorf("resolve repo root: %w", err)
	}
	manifestPath := resolvePath(repoRoot, opts.Manifest)
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return RunResult{}, err
	}
	if manifest.PositiveDecision == "" {
		manifest.PositiveDecision = "block"
	}
	sortedCases := append([]ManifestCase(nil), manifest.Cases...)
	sort.Slice(sortedCases, func(i, j int) bool { return sortedCases[i].ID < sortedCases[j].ID })

	binaryPath := opts.Binary
	if binaryPath == "" {
		name := "dbguard-bench"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		tempDir, err := os.MkdirTemp("", "dbwall-benchmark-*")
		if err != nil {
			return RunResult{}, fmt.Errorf("create temp benchmark dir: %w", err)
		}
		binaryPath = filepath.Join(tempDir, name)
	}
	if !filepath.IsAbs(binaryPath) {
		binaryPath = resolvePath(repoRoot, binaryPath)
	}
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return RunResult{}, fmt.Errorf("create binary dir: %w", err)
	}
	buildCmd := []string{"go", "build", "-o", binaryPath, "./cmd/dbguard"}
	if err := runCommand(ctx, repoRoot, buildCmd...); err != nil {
		return RunResult{}, err
	}

	result := RunResult{
		RepoRoot:         repoRoot,
		ManifestPath:     manifestPath,
		BinaryPath:       binaryPath,
		BuildCommand:     buildCmd,
		WarmupCommand:    []string{binaryPath, "version"},
		PositiveDecision: manifest.PositiveDecision,
		Definitions: Definitions{
			PrecisionPositiveClass: manifest.PositiveDecision,
			AccuracyDefinition:     "accuracy is exact decision match rate across allow, warn, and block",
			RuntimeDefinition:      "average runtime per case is the arithmetic mean wall-clock runtime of one sequential CLI execution per case after one uncaptured warmup command",
		},
	}
	if err := runCommand(ctx, repoRoot, result.WarmupCommand...); err != nil {
		return RunResult{}, err
	}

	for _, c := range sortedCases {
		caseResult, err := runCase(ctx, repoRoot, binaryPath, c, manifest.PositiveDecision)
		if err != nil {
			return RunResult{}, err
		}
		result.Cases = append(result.Cases, caseResult)
	}
	result.Metrics = computeMetrics(result.Cases, manifest.PositiveDecision)

	if opts.JSONOut != "" {
		jsonPath := resolvePath(repoRoot, opts.JSONOut)
		if err := writeJSON(jsonPath, result); err != nil {
			return RunResult{}, err
		}
	}
	if opts.ReportOut != "" {
		reportPath := resolvePath(repoRoot, opts.ReportOut)
		if err := writeReport(reportPath, result); err != nil {
			return RunResult{}, err
		}
	}
	return result, nil
}

func loadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	if len(manifest.Cases) == 0 {
		return Manifest{}, fmt.Errorf("manifest has no cases: %s", path)
	}
	return manifest, nil
}

func runCase(ctx context.Context, repoRoot, binaryPath string, c ManifestCase, positiveDecision string) (CaseResult, error) {
	sqlPath := filepath.Join(repoRoot, "benchmark", filepath.FromSlash(c.SQLFile))
	args := []string{binaryPath, "review-file", sqlPath, "--format", "json"}
	if c.PolicyFile != "" {
		policyPath := filepath.Join(repoRoot, "benchmark", filepath.FromSlash(c.PolicyFile))
		args = append(args, "--policy", policyPath)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	runtimeNanos := time.Since(start).Nanoseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return CaseResult{}, fmt.Errorf("run case %s: %w", c.ID, err)
		}
	}

	var parsed dbguardJSON
	if err := json.Unmarshal(output, &parsed); err != nil {
		return CaseResult{}, fmt.Errorf("parse dbguard json for case %s: %w\noutput: %s", c.ID, err, string(output))
	}

	actual := parsed.Decision
	expectedPositive := c.ExpectedDecision == positiveDecision
	actualPositive := actual == positiveDecision
	return CaseResult{
		ID:               c.ID,
		Category:         c.Category,
		SQLFile:          filepath.ToSlash(c.SQLFile),
		PolicyFile:       filepath.ToSlash(c.PolicyFile),
		ExpectedDecision: c.ExpectedDecision,
		ActualDecision:   actual,
		ExactMatch:       actual == c.ExpectedDecision,
		FalsePositive:    actualPositive && !expectedPositive,
		FalseNegative:    !actualPositive && expectedPositive,
		ExitCode:         exitCode,
		RuntimeNanos:     runtimeNanos,
		Command:          normalizeArgs(args, repoRoot),
		Stdout:           string(output),
		Stderr:           "",
	}, nil
}

func resolvePath(repoRoot, path string) string {
	path = filepath.FromSlash(path)
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func computeMetrics(cases []CaseResult, positiveDecision string) Metrics {
	var m Metrics
	m.TotalCases = len(cases)
	var exactMatches int
	var totalRuntime int64
	for _, c := range cases {
		totalRuntime += c.RuntimeNanos
		if c.ExactMatch {
			exactMatches++
		}
		switch {
		case c.ExpectedDecision == positiveDecision && c.ActualDecision == positiveDecision:
			m.CorrectBlocks++
		case c.ExpectedDecision == "allow" && c.ActualDecision == "allow":
			m.CorrectAllows++
		case c.ExpectedDecision == "warn" && c.ActualDecision == "warn":
			m.CorrectWarns++
		}
		if c.FalsePositive {
			m.FalsePositives++
		}
		if c.FalseNegative {
			m.FalseNegatives++
		}
	}
	positivePredictions := m.CorrectBlocks + m.FalsePositives
	positiveTruth := m.CorrectBlocks + m.FalseNegatives
	if positivePredictions > 0 {
		m.Precision = float64(m.CorrectBlocks) / float64(positivePredictions)
	}
	if positiveTruth > 0 {
		m.Recall = float64(m.CorrectBlocks) / float64(positiveTruth)
	}
	if m.TotalCases > 0 {
		m.Accuracy = float64(exactMatches) / float64(m.TotalCases)
		m.AverageRuntimeNanos = totalRuntime / int64(m.TotalCases)
		m.AverageRuntimeMillis = float64(m.AverageRuntimeNanos) / float64(time.Millisecond)
	}
	return m
}

func writeJSON(path string, result RunResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create results dir: %w", err)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write result json: %w", err)
	}
	return nil
}

func writeReport(path string, result RunResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create report dir: %w", err)
	}
	var b strings.Builder
	b.WriteString("# DBwall Benchmark Report\n\n")
	b.WriteString("## Measured Results\n\n")
	b.WriteString(fmt.Sprintf("- Total cases: `%d`\n", result.Metrics.TotalCases))
	b.WriteString(fmt.Sprintf("- Correct blocks: `%d`\n", result.Metrics.CorrectBlocks))
	b.WriteString(fmt.Sprintf("- Correct allows: `%d`\n", result.Metrics.CorrectAllows))
	b.WriteString(fmt.Sprintf("- Correct warns: `%d`\n", result.Metrics.CorrectWarns))
	b.WriteString(fmt.Sprintf("- False positives: `%d`\n", result.Metrics.FalsePositives))
	b.WriteString(fmt.Sprintf("- False negatives: `%d`\n", result.Metrics.FalseNegatives))
	b.WriteString(fmt.Sprintf("- Precision (`%s` as positive class): `%.4f`\n", result.PositiveDecision, result.Metrics.Precision))
	b.WriteString(fmt.Sprintf("- Recall (`%s` as positive class): `%.4f`\n", result.PositiveDecision, result.Metrics.Recall))
	b.WriteString(fmt.Sprintf("- Accuracy (exact decision match): `%.4f`\n", result.Metrics.Accuracy))
	b.WriteString(fmt.Sprintf("- Average runtime per case: `%.3f ms`\n\n", result.Metrics.AverageRuntimeMillis))

	b.WriteString("## Assumptions and Definitions\n\n")
	b.WriteString(fmt.Sprintf("- Positive class for precision/recall: `%s`\n", result.Definitions.PrecisionPositiveClass))
	b.WriteString(fmt.Sprintf("- %s\n", result.Definitions.AccuracyDefinition))
	b.WriteString(fmt.Sprintf("- %s\n\n", result.Definitions.RuntimeDefinition))

	b.WriteString("## Case Results\n\n")
	b.WriteString("| ID | Category | Expected | Actual | Exact Match | Runtime (ms) |\n")
	b.WriteString("| --- | --- | --- | --- | --- | ---: |\n")
	for _, c := range result.Cases {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %t | %.3f |\n",
			c.ID,
			c.Category,
			c.ExpectedDecision,
			c.ActualDecision,
			c.ExactMatch,
			float64(c.RuntimeNanos)/float64(time.Millisecond),
		))
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write benchmark report: %w", err)
	}
	return nil
}

func runCommand(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run %q: %w\noutput: %s", strings.Join(args, " "), err, string(output))
	}
	return nil
}

func normalizeArgs(args []string, repoRoot string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, repoRoot) {
			rel, err := filepath.Rel(repoRoot, arg)
			if err == nil {
				out[i] = filepath.ToSlash(rel)
				continue
			}
		}
		out[i] = filepath.ToSlash(arg)
	}
	return out
}

func RequiredPaths(result RunResult) []string {
	paths := []string{result.ManifestPath, result.BinaryPath}
	for _, c := range result.Cases {
		paths = append(paths, c.SQLFile)
		if c.PolicyFile != "" {
			paths = append(paths, c.PolicyFile)
		}
	}
	sort.Strings(paths)
	return slices.Compact(paths)
}
