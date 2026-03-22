package parser

import (
	"encoding/json"
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// Parse parses PostgreSQL SQL and returns a list of analyzer-friendly statements.
// Multi-statement input is supported; each statement is returned in order.
func Parse(sql string) ([]Statement, error) {
	jsonStr, err := pg_query.ParseToJSON(sql)
	if err != nil {
		return nil, fmt.Errorf("parse SQL: %w", err)
	}
	return parseJSONToStatements(jsonStr)
}

// parseJSONToStatements decodes the pg_query JSON output and extracts statement list.
// The JSON shape is: {"version": N, "stmts": [{"stmt": {"DeleteStmt": {...}}, ...}]}
func parseJSONToStatements(jsonStr string) ([]Statement, error) {
	var root struct {
		Stmts []struct {
			Stmt map[string]json.RawMessage `json:"stmt"`
		} `json:"stmts"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, fmt.Errorf("decode parse tree JSON: %w", err)
	}
	var out []Statement
	for i, s := range root.Stmts {
		if s.Stmt == nil {
			continue
		}
		for key, raw := range s.Stmt {
			stmt, err := extractStatement(key, raw)
			if err != nil {
				return nil, fmt.Errorf("statement %d: %w", i+1, err)
			}
			out = append(out, stmt)
			break // one statement type per wrapper
		}
	}
	return out, nil
}

func extractStatement(typeKey string, raw json.RawMessage) (Statement, error) {
	switch typeKey {
	case "DeleteStmt":
		return extractTableAndWhereStmt(raw, StmtTypeDelete)
	case "UpdateStmt":
		return extractTableAndWhereStmt(raw, StmtTypeUpdate)
	case "DropStmt":
		return extractDropStmt(raw)
	case "AlterTableStmt":
		return extractAlterTableStmt(raw)
	case "SelectStmt", "InsertStmt":
		// Pass through for reporting; we may need table for InsertStmt later
		return Statement{Type: stmtTypeFromKey(typeKey), Table: tableFromRaw(raw)}, nil
	default:
		return Statement{Type: StmtTypeOther}, nil
	}
}

func stmtTypeFromKey(k string) StmtType {
	switch k {
	case "SelectStmt":
		return StmtTypeSelect
	case "InsertStmt":
		return StmtTypeInsert
	default:
		return StmtTypeOther
	}
}

func extractTableAndWhereStmt(raw json.RawMessage, stype StmtType) (Statement, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return Statement{}, err
	}
	table := extractRelation(m)
	hasWhere := hasWhereClause(m)
	return Statement{Type: stype, Table: table, HasWhere: hasWhere}, nil
}

func extractDropStmt(raw json.RawMessage) (Statement, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return Statement{}, err
	}
	// removeType: 1 = OBJECT_TABLE, or string "OBJECT_TABLE" (pg_query_go v5 JSON)
	removeType, _ := m["remove_type"].(float64)
	if removeType == 0 {
		removeType, _ = m["removeType"].(float64)
	}
	isTable := removeType == 1
	if !isTable {
		if s, _ := m["removeType"].(string); s == "OBJECT_TABLE" {
			isTable = true
		}
		if s, _ := m["remove_type"].(string); s == "OBJECT_TABLE" {
			isTable = true
		}
	}
	if !isTable {
		return Statement{Type: StmtTypeOther}, nil
	}
	table := extractDropObjectsTable(m)
	return Statement{Type: StmtTypeDropTable, Table: table}, nil
}

func extractAlterTableStmt(raw json.RawMessage) (Statement, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return Statement{}, err
	}
	table := extractRelation(m)
	cmds, _ := m["cmds"].([]interface{})
	for _, c := range cmds {
		cmd, _ := c.(map[string]interface{})
		if cmd == nil {
			continue
		}
		// pg_query_go wraps cmd as {"AlterTableCmd": {"subtype": "AT_DropColumn", ...}}
		if atc, ok := cmd["AlterTableCmd"].(map[string]interface{}); ok {
			if isDropColumnCmd(atc) {
				return Statement{Type: StmtTypeAlterTableDropCol, Table: table}, nil
			}
			continue
		}
		if isDropColumnCmd(cmd) {
			return Statement{Type: StmtTypeAlterTableDropCol, Table: table}, nil
		}
	}
	// ALTER TABLE but no DROP COLUMN in this statement
	return Statement{Type: StmtTypeOther, Table: table}, nil
}

func isDropColumnCmd(cmd map[string]interface{}) bool {
	subtype, _ := cmd["subtype"].(float64)
	if subtype == 11 {
		return true
	}
	if s, _ := cmd["subtype"].(string); s == "AT_DropColumn" {
		return true
	}
	return false
}

func extractRelation(m map[string]interface{}) string {
	rel, ok := m["relation"]
	if !ok {
		return ""
	}
	return relationRelname(rel)
}

func relationRelname(rel interface{}) string {
	m, ok := rel.(map[string]interface{})
	if !ok {
		return ""
	}
	// RangeVar can have "relname" directly
	if n, ok := m["relname"].(string); ok {
		return n
	}
	// or nested under "RangeVar" key (protobuf oneof)
	if rv, ok := m["RangeVar"].(map[string]interface{}); ok {
		if n, ok := rv["relname"].(string); ok {
			return n
		}
	}
	for _, v := range m {
		if s := relationRelname(v); s != "" {
			return s
		}
	}
	return ""
}

func hasWhereClause(m map[string]interface{}) bool {
	// protobuf JSON may use where_clause or whereClause
	for _, key := range []string{"where_clause", "whereClause"} {
		if v, ok := m[key]; ok && v != nil {
			// empty array or null means no clause
			if arr, ok := v.([]interface{}); ok && len(arr) == 0 {
				return false
			}
			return true
		}
	}
	return false
}

func extractDropObjectsTable(m map[string]interface{}) string {
	objs, _ := m["objects"].([]interface{})
	if len(objs) == 0 {
		return ""
	}
	// pg_query_go: objects is [{"List":{"items":[{"String":{"sval":"users"}}]}}] for DROP TABLE users
	first := objs[0]
	if listNode, ok := first.(map[string]interface{}); ok {
		if list, ok := listNode["List"].(map[string]interface{}); ok {
			items, _ := list["items"].([]interface{})
			if len(items) > 0 {
				last := items[len(items)-1]
				if strNode, ok := last.(map[string]interface{}); ok {
					if str, ok := strNode["String"].(map[string]interface{}); ok {
						if s, ok := str["sval"].(string); ok {
							return s
						}
					}
					if s, ok := strNode["sval"].(string); ok {
						return s
					}
				}
			}
		}
		if rv := relnameFromNode(listNode); rv != "" {
			return rv
		}
	}
	if list, ok := first.([]interface{}); ok && len(list) > 0 {
		last := list[len(list)-1]
		if m, ok := last.(map[string]interface{}); ok {
			if rv := relnameFromNode(m); rv != "" {
				return rv
			}
			if s, ok := m["sval"].(string); ok {
				return s
			}
		}
	}
	return ""
}

func relnameFromNode(m map[string]interface{}) string {
	if rv, ok := m["RangeVar"].(map[string]interface{}); ok {
		if n, ok := rv["relname"].(string); ok {
			return n
		}
	}
	if n, ok := m["relname"].(string); ok {
		return n
	}
	for _, v := range m {
		if vm, ok := v.(map[string]interface{}); ok {
			if s := relnameFromNode(vm); s != "" {
				return s
			}
		}
	}
	return ""
}

func tableFromRaw(raw json.RawMessage) string {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return extractRelation(m)
}
