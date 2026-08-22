//go:build cgo

package parser

import (
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// Parse validates SQL with the structured PostgreSQL AST and derives semantic metadata.
func Parse(sql string) ([]Statement, error) {
	segments, err := splitSQLStatementsWithLines(stripUTF8BOM(sql))
	if err != nil {
		return nil, err
	}
	stmts := make([]Statement, 0, len(segments))
	for i, segment := range segments {
		tree, err := pg_query.Parse(segment.SQL)
		if err != nil {
			return nil, fmt.Errorf("statement %d: parse SQL: %w", i+1, err)
		}
		if tree == nil || len(tree.Stmts) == 0 {
			return nil, fmt.Errorf("statement %d: empty parse tree", i+1)
		}
		if len(tree.Stmts) != 1 {
			return nil, fmt.Errorf("statement %d: expected single statement segment, got %d", i+1, len(tree.Stmts))
		}
		raw := tree.Stmts[0]
		if raw == nil || raw.Stmt == nil {
			return nil, fmt.Errorf("statement %d: missing statement node", i+1)
		}
		stmt, err := extractStatementFromNode(raw.Stmt, segment.SQL, segment.StartLine)
		if err != nil {
			return nil, fmt.Errorf("statement %d: %w", i+1, err)
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}
