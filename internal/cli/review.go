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
	return printAndExit(res, format)
}

// ReviewFile runs the full review pipeline on a SQL file and prints output.
func ReviewFile(filePath string, policyPath string, format string) int {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read file:", err)
		return ExitError
	}
	return ReviewSQL(string(data), policyPath, format)
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

func printAndExit(res *analyzer.Result, format string) int {
	var out string
	var err error
	if format == "json" {
		out, err = report.JSON(res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return ExitError
		}
	} else {
		out = report.Human(res)
	}
	fmt.Println(out)
	return decisionToExit(res.Decision)
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
