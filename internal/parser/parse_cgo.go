//go:build cgo

package parser

import (
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// Parse validates SQL with pg_query and then derives the statement metadata DBwall needs.
func Parse(sql string) ([]Statement, error) {
	segments, err := splitSQLStatementsWithLines(sql)
	if err != nil {
		return nil, err
	}
	stmts := make([]Statement, 0, len(segments))
	for i, segment := range segments {
		if _, err := pg_query.ParseToJSON(segment.SQL); err != nil {
			return nil, fmt.Errorf("statement %d: parse SQL: %w", i+1, err)
		}
		stmt, err := parseStatementText(segment.SQL, segment.StartLine)
		if err != nil {
			return nil, fmt.Errorf("statement %d: %w", i+1, err)
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}
