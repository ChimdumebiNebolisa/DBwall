//go:build !cgo

package parser

import "fmt"

// Parse parses the supported SQL subset without CGO.
// The fallback path intentionally focuses on core coverage for DBwall's rule set.
func Parse(sql string) ([]Statement, error) {
	segments, err := splitSQLStatementsWithLines(sql)
	if err != nil {
		return nil, err
	}
	stmts := make([]Statement, 0, len(segments))
	for i, segment := range segments {
		stmt, err := parseStatementText(segment.SQL, segment.StartLine)
		if err != nil {
			return nil, fmt.Errorf("statement %d: %w", i+1, err)
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}
