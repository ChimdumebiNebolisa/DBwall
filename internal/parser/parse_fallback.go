//go:build !cgo

package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// Parse parses the supported SQL subset without CGO.
// When CGO is enabled, the pg_query-backed implementation is used instead.
func Parse(sql string) ([]Statement, error) {
	parts, err := splitStatements(sql)
	if err != nil {
		return nil, err
	}
	stmts := make([]Statement, 0, len(parts))
	for i, part := range parts {
		stmt, err := parseFallbackStatement(part)
		if err != nil {
			return nil, fmt.Errorf("statement %d: %w", i+1, err)
		}
		stmts = append(stmts, stmt)
	}
	return stmts, nil
}

func parseFallbackStatement(sql string) (Statement, error) {
	tokens, err := tokenize(sql)
	if err != nil {
		return Statement{}, err
	}
	if len(tokens) == 0 {
		return Statement{}, fmt.Errorf("syntax error at end of input")
	}

	switch upper(tokens[0]) {
	case "DELETE":
		return parseDelete(tokens)
	case "UPDATE":
		return parseUpdate(tokens)
	case "DROP":
		return parseDrop(tokens)
	case "ALTER":
		return parseAlter(tokens)
	case "SELECT":
		return parseSelect(tokens), nil
	case "INSERT":
		return parseInsert(tokens)
	default:
		return Statement{}, fmt.Errorf("syntax error at or near %q", tokens[0])
	}
}

func parseDelete(tokens []string) (Statement, error) {
	pos := 1
	if !tokenIs(tokens, pos, "FROM") {
		return Statement{}, syntaxNear(tokens, pos)
	}
	pos++
	if tokenIs(tokens, pos, "ONLY") {
		pos++
	}
	table, next, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	return Statement{
		Type:     StmtTypeDelete,
		Table:    table,
		HasWhere: containsKeyword(tokens[next:], "WHERE"),
	}, nil
}

func parseUpdate(tokens []string) (Statement, error) {
	pos := 1
	if tokenIs(tokens, pos, "ONLY") {
		pos++
	}
	table, next, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	if !containsKeyword(tokens[next:], "SET") {
		return Statement{}, fmt.Errorf("syntax error: UPDATE missing SET clause")
	}
	return Statement{
		Type:     StmtTypeUpdate,
		Table:    table,
		HasWhere: containsKeyword(tokens[next:], "WHERE"),
	}, nil
}

func parseDrop(tokens []string) (Statement, error) {
	pos := 1
	if !tokenIs(tokens, pos, "TABLE") {
		return Statement{Type: StmtTypeOther}, nil
	}
	pos++
	if tokenIs(tokens, pos, "IF") && tokenIs(tokens, pos+1, "EXISTS") {
		pos += 2
	}
	table, _, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	return Statement{Type: StmtTypeDropTable, Table: table}, nil
}

func parseAlter(tokens []string) (Statement, error) {
	if !tokenIs(tokens, 1, "TABLE") {
		return Statement{Type: StmtTypeOther}, nil
	}
	pos := 2
	if tokenIs(tokens, pos, "ONLY") {
		pos++
	}
	table, next, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	for i := next; i < len(tokens); i++ {
		if upper(tokens[i]) == "DROP" && tokenIs(tokens, i+1, "COLUMN") {
			return Statement{Type: StmtTypeAlterTableDropCol, Table: table}, nil
		}
	}
	return Statement{Type: StmtTypeOther, Table: table}, nil
}

func parseSelect(tokens []string) Statement {
	table := ""
	if idx := indexKeyword(tokens, "FROM"); idx >= 0 && idx+1 < len(tokens) {
		if name, _, err := parseQualifiedIdentifier(tokens, idx+1); err == nil {
			table = name
		}
	}
	return Statement{
		Type:     StmtTypeSelect,
		Table:    table,
		HasWhere: containsKeyword(tokens, "WHERE"),
	}
}

func parseInsert(tokens []string) (Statement, error) {
	pos := 1
	if tokenIs(tokens, pos, "INTO") {
		pos++
	}
	table, _, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	return Statement{Type: StmtTypeInsert, Table: table}, nil
}

func splitStatements(sql string) ([]string, error) {
	var out []string
	var current strings.Builder

	inSingle := false
	inDouble := false
	inLineComment := false
	inBlockComment := false
	dollarTag := ""

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		if inLineComment {
			current.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			current.WriteByte(ch)
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				current.WriteByte(sql[i+1])
				i++
				inBlockComment = false
			}
			continue
		}
		if dollarTag != "" {
			current.WriteByte(ch)
			if strings.HasPrefix(sql[i:], dollarTag) {
				for j := 1; j < len(dollarTag); j++ {
					current.WriteByte(sql[i+j])
				}
				i += len(dollarTag) - 1
				dollarTag = ""
			}
			continue
		}
		if inSingle {
			current.WriteByte(ch)
			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					current.WriteByte(sql[i+1])
					i++
				} else {
					inSingle = false
				}
			}
			continue
		}
		if inDouble {
			current.WriteByte(ch)
			if ch == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					current.WriteByte(sql[i+1])
					i++
				} else {
					inDouble = false
				}
			}
			continue
		}

		if ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			current.WriteByte(ch)
			current.WriteByte(sql[i+1])
			i++
			inLineComment = true
			continue
		}
		if ch == '/' && i+1 < len(sql) && sql[i+1] == '*' {
			current.WriteByte(ch)
			current.WriteByte(sql[i+1])
			i++
			inBlockComment = true
			continue
		}
		if ch == '\'' {
			current.WriteByte(ch)
			inSingle = true
			continue
		}
		if ch == '"' {
			current.WriteByte(ch)
			inDouble = true
			continue
		}
		if ch == '$' {
			if tag := readDollarTag(sql[i:]); tag != "" {
				current.WriteString(tag)
				i += len(tag) - 1
				dollarTag = tag
				continue
			}
		}
		if ch == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				out = append(out, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	if inSingle || inDouble || inBlockComment || dollarTag != "" {
		return nil, fmt.Errorf("syntax error: unterminated quoted string or comment")
	}

	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		out = append(out, stmt)
	}
	return out, nil
}

func tokenize(sql string) ([]string, error) {
	var tokens []string

	for i := 0; i < len(sql); {
		ch := rune(sql[i])
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		if sql[i] == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			i += 2
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			continue
		}
		if sql[i] == '/' && i+1 < len(sql) && sql[i+1] == '*' {
			i += 2
			closed := false
			for i+1 < len(sql) {
				if sql[i] == '*' && sql[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("syntax error: unterminated block comment")
			}
			continue
		}
		if sql[i] == '\'' {
			token, next, err := readSingleQuoted(sql, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			i = next
			continue
		}
		if sql[i] == '"' {
			token, next, err := readDoubleQuoted(sql, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			i = next
			continue
		}
		if sql[i] == '$' {
			if tag := readDollarTag(sql[i:]); tag != "" {
				next := strings.Index(sql[i+len(tag):], tag)
				if next < 0 {
					return nil, fmt.Errorf("syntax error: unterminated dollar-quoted string")
				}
				token := sql[i : i+len(tag)+next+len(tag)]
				tokens = append(tokens, token)
				i += len(token)
				continue
			}
		}
		if isIdentifierStart(ch) {
			start := i
			i++
			for i < len(sql) && isIdentifierPart(rune(sql[i])) {
				i++
			}
			tokens = append(tokens, sql[start:i])
			continue
		}
		if unicode.IsDigit(ch) {
			start := i
			i++
			for i < len(sql) {
				r := rune(sql[i])
				if !(unicode.IsDigit(r) || r == '.') {
					break
				}
				i++
			}
			tokens = append(tokens, sql[start:i])
			continue
		}
		if strings.ContainsRune("(),.*=", ch) {
			tokens = append(tokens, sql[i:i+1])
			i++
			continue
		}
		if unicode.IsPunct(ch) || unicode.IsSymbol(ch) {
			tokens = append(tokens, sql[i:i+1])
			i++
			continue
		}
		return nil, fmt.Errorf("syntax error at or near %q", string(ch))
	}

	return tokens, nil
}

func parseQualifiedIdentifier(tokens []string, pos int) (string, int, error) {
	if pos >= len(tokens) {
		return "", pos, fmt.Errorf("syntax error at end of input")
	}

	part, ok := normalizeIdentifier(tokens[pos])
	if !ok {
		return "", pos, syntaxNear(tokens, pos)
	}
	parts := []string{part}
	pos++

	for pos < len(tokens) && tokens[pos] == "." {
		if pos+1 >= len(tokens) {
			return "", pos, fmt.Errorf("syntax error at end of input")
		}
		next, ok := normalizeIdentifier(tokens[pos+1])
		if !ok {
			return "", pos + 1, syntaxNear(tokens, pos+1)
		}
		parts = append(parts, next)
		pos += 2
	}

	return strings.Join(parts, "."), pos, nil
}

func normalizeIdentifier(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	if token[0] == '"' {
		if len(token) < 2 || token[len(token)-1] != '"' {
			return "", false
		}
		return strings.ReplaceAll(token[1:len(token)-1], `""`, `"`), true
	}
	r := rune(token[0])
	if !isIdentifierStart(r) {
		return "", false
	}
	return strings.ToLower(token), true
}

func containsKeyword(tokens []string, keyword string) bool {
	return indexKeyword(tokens, keyword) >= 0
}

func indexKeyword(tokens []string, keyword string) int {
	for i, token := range tokens {
		if upper(token) == keyword {
			return i
		}
	}
	return -1
}

func tokenIs(tokens []string, pos int, want string) bool {
	return pos >= 0 && pos < len(tokens) && upper(tokens[pos]) == want
}

func syntaxNear(tokens []string, pos int) error {
	if pos >= 0 && pos < len(tokens) {
		return fmt.Errorf("syntax error at or near %q", tokens[pos])
	}
	return fmt.Errorf("syntax error at end of input")
}

func upper(s string) string {
	return strings.ToUpper(s)
}

func readSingleQuoted(sql string, start int) (string, int, error) {
	i := start + 1
	for i < len(sql) {
		if sql[i] == '\'' {
			if i+1 < len(sql) && sql[i+1] == '\'' {
				i += 2
				continue
			}
			return sql[start : i+1], i + 1, nil
		}
		i++
	}
	return "", 0, fmt.Errorf("syntax error: unterminated quoted string")
}

func readDoubleQuoted(sql string, start int) (string, int, error) {
	i := start + 1
	for i < len(sql) {
		if sql[i] == '"' {
			if i+1 < len(sql) && sql[i+1] == '"' {
				i += 2
				continue
			}
			return sql[start : i+1], i + 1, nil
		}
		i++
	}
	return "", 0, fmt.Errorf("syntax error: unterminated quoted identifier")
}

func readDollarTag(sql string) string {
	if !strings.HasPrefix(sql, "$") {
		return ""
	}
	for i := 1; i < len(sql); i++ {
		ch := rune(sql[i])
		if ch == '$' {
			return sql[:i+1]
		}
		if !(unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_') {
			return ""
		}
	}
	return ""
}

func isIdentifierStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentifierPart(ch rune) bool {
	return isIdentifierStart(ch) || unicode.IsDigit(ch) || ch == '$'
}
