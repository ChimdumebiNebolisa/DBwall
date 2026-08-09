package cli

import (
	"fmt"
	"os"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
	"github.com/ChimdumebiNebolisa/DBwall/internal/report"
)

// Exit codes for CI (documented in README and SPEC).
const (
	ExitAllow = 0
	ExitError = 1
	ExitWarn  = 2
	ExitBlock = 3
)

// ReviewSQL runs the full review pipeline on SQL string and prints output.
// Returns the exit code to use.
func ReviewSQL(sql string, policyPath string, format string) int {
	p, err := loadPolicy(policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "policy:", err)
		return ExitError
	}
	stmts, err := parser.Parse(sql)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		return ExitError
	}
	res := analyzer.Analyze(stmts, p)
	return printAndExit(res, format, report.Options{CoverageMode: parser.CoverageMode()})
}

// ReviewFile runs the full review pipeline on a SQL file and prints output.
func ReviewFile(filePath string, policyPath string, format string) int {
	return ReviewFiles([]string{filePath}, policyPath, format)
}

// ReviewFiles runs the review pipeline across one or more SQL files, aggregates
// findings into a single decision (strictest wins), and prints output once.
func ReviewFiles(filePaths []string, policyPath string, format string) int {
	p, err := loadPolicy(policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "policy:", err)
		return ExitError
	}

	aggregated := &analyzer.Result{
		Decision: policy.DecisionAllow,
		Severity: analyzer.SeverityLow,
	}
	if len(filePaths) == 0 {
		return printAndExit(aggregated, format, report.Options{CoverageMode: parser.CoverageMode()})
	}

	for _, filePath := range filePaths {
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read file:", err)
			return ExitError
		}
		stmts, err := parser.Parse(string(data))
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse:", filePath+":", err)
			return ExitError
		}
		res := analyzer.Analyze(stmts, p)
		for i := range res.Statements {
			st := res.Statements[i]
			st.Location = &analyzer.SourceLocation{
				Path:      filePath,
				StartLine: st.StartLine,
			}
			st.Index = len(aggregated.Statements) + 1
			for j := range st.Findings {
				st.Findings[j].StatementIndex = st.Index
			}
			aggregated.Statements = append(aggregated.Statements, st)
		}
	}

	aggregated.Decision, aggregated.Severity, aggregated.Summary = analyzer.Aggregate(aggregated.Statements)
	sourcePath := ""
	if len(filePaths) == 1 {
		sourcePath = filePaths[0]
	}
	return printAndExit(aggregated, format, report.Options{SourcePath: sourcePath, CoverageMode: parser.CoverageMode()})
}

func loadPolicy(path string) (*policy.Policy, error) {
	if path == "" {
		return policy.DefaultPolicy(), nil
	}
	p, err := policy.LoadFromFile(path)
	if err != nil {
		return nil, err
	}
	if err := policy.Validate(p); err != nil {
		return nil, err
	}
	return p, nil
}

func printAndExit(res *analyzer.Result, format string, opts report.Options) int {
	var out string
	var err error
	switch format {
	case "json":
		out, err = report.JSON(res, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return ExitError
		}
	case "sarif":
		out, err = report.SARIF(res, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return ExitError
		}
	default:
		out = report.Human(res, opts)
	}
	fmt.Println(out)
	return decisionToExit(res.Decision)
}

// ExitCodeForDecision maps decision to exit code (for testing and documentation).
func ExitCodeForDecision(d policy.Decision) int {
	return decisionToExit(d)
}

func decisionToExit(d policy.Decision) int {
	switch d {
	case policy.DecisionBlock:
		return ExitBlock
	case policy.DecisionWarn:
		return ExitWarn
	case policy.DecisionAllow:
		return ExitAllow
	default:
		return ExitAllow
	}
}
