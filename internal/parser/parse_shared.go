package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type sqlSegment struct {
	SQL       string
	StartLine int
}

// stripUTF8BOM removes a leading UTF-8 byte-order mark so BOM-prefixed files
// parse instead of failing with an opaque syntax error (audit finding F-006).
func stripUTF8BOM(sql string) string {
	return strings.TrimPrefix(sql, "\uFEFF")
}

func splitSQLStatementsWithLines(sql string) ([]sqlSegment, error) {
	var out []sqlSegment
	var current strings.Builder

	line := 1
	segmentLine := 1
	segmentStarted := false
	// segmentHasContent tracks whether the current segment contains any real SQL
	// outside comments; a trailing comment must not become a phantom statement.
	segmentHasContent := false
	inSingle := false
	inDouble := false
	inLineComment := false
	inBlockComment := false
	dollarTag := ""

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if !segmentStarted && !unicode.IsSpace(rune(ch)) {
			segmentLine = line
			segmentStarted = true
		}

		if inLineComment {
			current.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
				line++
			}
			continue
		}
		if inBlockComment {
			current.WriteByte(ch)
			if ch == '\n' {
				line++
			}
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				current.WriteByte(sql[i+1])
				i++
				inBlockComment = false
			}
			continue
		}
		if dollarTag != "" {
			current.WriteByte(ch)
			if ch == '\n' {
				line++
			}
			if strings.HasPrefix(sql[i:], dollarTag) {
				for j := 1; j < len(dollarTag); j++ {
					current.WriteByte(sql[i+j])
					if sql[i+j] == '\n' {
						line++
					}
				}
				i += len(dollarTag) - 1
				dollarTag = ""
			}
			continue
		}
		if inSingle {
			current.WriteByte(ch)
			if ch == '\n' {
				line++
			}
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
			if ch == '\n' {
				line++
			}
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
			segmentHasContent = true
			inSingle = true
			continue
		}
		if ch == '"' {
			current.WriteByte(ch)
			segmentHasContent = true
			inDouble = true
			continue
		}
		if ch == '$' {
			if tag := readDollarTag(sql[i:]); tag != "" {
				current.WriteString(tag)
				segmentHasContent = true
				i += len(tag) - 1
				dollarTag = tag
				continue
			}
		}
		if ch == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && segmentHasContent {
				out = append(out, sqlSegment{SQL: stmt, StartLine: segmentLine})
			}
			current.Reset()
			segmentStarted = false
			segmentHasContent = false
			continue
		}

		current.WriteByte(ch)
		if !unicode.IsSpace(rune(ch)) {
			segmentHasContent = true
		}
		if ch == '\n' {
			line++
		}
	}

	if inSingle || inDouble || inBlockComment || dollarTag != "" {
		return nil, fmt.Errorf("syntax error: unterminated quoted string or comment")
	}

	stmt := strings.TrimSpace(current.String())
	if stmt != "" && segmentHasContent {
		out = append(out, sqlSegment{SQL: stmt, StartLine: segmentLine})
	}
	return out, nil
}

func parseStatementText(sql string, startLine int) (Statement, error) {
	tokens, err := tokenizeSQL(sql)
	if err != nil {
		return Statement{}, err
	}
	if len(tokens) == 0 {
		return Statement{}, fmt.Errorf("syntax error at end of input")
	}

	stmt := Statement{
		RawSQL:       strings.TrimSpace(sql),
		StartLine:    startLine,
		Completeness: AnalysisPartial,
		IncompleteReasons: []string{
			"core_mode_token_parser",
		},
	}

	var (
		out Statement
		parseErr error
	)
	switch upper(tokens[0]) {
	case "DELETE":
		out, parseErr = parseDeleteTokens(tokens, stmt)
	case "UPDATE":
		out, parseErr = parseUpdateTokens(tokens, stmt)
	case "DROP":
		out, parseErr = parseDropTokens(tokens, stmt)
	case "ALTER":
		out, parseErr = parseAlterTokens(tokens, stmt)
	case "SELECT":
		out, parseErr = parseSelectTokens(tokens, stmt), nil
	case "INSERT":
		out, parseErr = parseInsertTokens(tokens, stmt)
	case "TRUNCATE":
		out, parseErr = parseTruncateTokens(tokens, stmt)
	case "GRANT":
		out, parseErr = parseGrantTokens(tokens, stmt)
	case "REVOKE":
		out, parseErr = parseRevokeTokens(tokens, stmt)
	case "COPY":
		out, parseErr = parseCopyTokens(tokens, stmt)
	default:
		return Statement{}, fmt.Errorf("syntax error at or near %q", tokens[0])
	}
	if parseErr != nil {
		return Statement{}, parseErr
	}
	if out.Type == StmtTypeOther {
		// Actionable reason: unsupported statement shapes must surface as
		// semantic_analysis_incomplete in core mode too, matching full mode
		// (audit finding F-005). Previously this was silently filtered.
		markIncomplete(&out, AnalysisUnsupported, "unsupported_statement_in_core_mode")
	}
	syncConvenienceFields(&out)
	out.IncompleteReasons = ensureReasonsBounded(out.IncompleteReasons, 5)
	return out, nil
}

func parseDeleteTokens(tokens []string, stmt Statement) (Statement, error) {
	pos := 1
	if !tokenIs(tokens, pos, "FROM") {
		return Statement{}, syntaxNear(tokens, pos)
	}
	pos++
	if tokenIs(tokens, pos, "ONLY") {
		pos++
	}
	table, _, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	stmt.Type = StmtTypeDelete
	stmt.HasWhere = containsKeywordTopLevel(tokens, "WHERE")
	stmt.WhereTrivial = trivialWhereTokens(tokens)
	setRelation(&stmt, table)
	return stmt, nil
}

func parseUpdateTokens(tokens []string, stmt Statement) (Statement, error) {
	pos := 1
	if tokenIs(tokens, pos, "ONLY") {
		pos++
	}
	table, next, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	setIndex := indexKeywordTopLevelFrom(tokens, next, "SET")
	if setIndex < 0 {
		return Statement{}, fmt.Errorf("syntax error: UPDATE missing SET clause")
	}
	stmt.Type = StmtTypeUpdate
	stmt.HasWhere = containsKeywordTopLevel(tokens, "WHERE")
	stmt.WhereTrivial = trivialWhereTokens(tokens)
	setRelation(&stmt, table)
	return stmt, nil
}

func parseDropTokens(tokens []string, stmt Statement) (Statement, error) {
	pos := 1
	if pos >= len(tokens) {
		return Statement{}, fmt.Errorf("syntax error at end of input")
	}
	switch upper(tokens[pos]) {
	case "TABLE":
		pos++
		if tokenIs(tokens, pos, "IF") && tokenIs(tokens, pos+1, "EXISTS") {
			pos += 2
		}
		stmt.Type = StmtTypeDropTable
		first := ""
		for pos < len(tokens) {
			name, next, err := parseQualifiedIdentifier(tokens, pos)
			if err != nil {
				break
			}
			if first == "" {
				first = name
			}
			setRelation(&stmt, name)
			pos = next
			if pos < len(tokens) && tokens[pos] == "," {
				pos++
				continue
			}
			break
		}
		if first == "" {
			return Statement{}, syntaxNear(tokens, pos)
		}
		return stmt, nil
	case "SCHEMA":
		pos++
		if tokenIs(tokens, pos, "IF") && tokenIs(tokens, pos+1, "EXISTS") {
			pos += 2
		}
		name, _, err := parseQualifiedIdentifier(tokens, pos)
		if err != nil {
			return Statement{}, err
		}
		stmt.Type = StmtTypeDropSchema
		stmt.Object = name
		stmt.Schema = name
		return stmt, nil
	case "DATABASE":
		pos++
		if tokenIs(tokens, pos, "IF") && tokenIs(tokens, pos+1, "EXISTS") {
			pos += 2
		}
		name, _, err := parseQualifiedIdentifier(tokens, pos)
		if err != nil {
			return Statement{}, err
		}
		stmt.Type = StmtTypeDropDatabase
		stmt.Object = name
		return stmt, nil
	default:
		stmt.Type = StmtTypeOther
		return stmt, nil
	}
}

func parseAlterTokens(tokens []string, stmt Statement) (Statement, error) {
	if tokenIs(tokens, 1, "DEFAULT") && tokenIs(tokens, 2, "PRIVILEGES") {
		stmt.Type = StmtTypeAlterDefaultPrivileges
		stmt.IsGrantToPublic = containsKeyword(tokens, "PUBLIC")
		stmt.Grantees = parseIdentifiersAfterKeyword(tokens, "TO")
		return stmt, nil
	}
	if !tokenIs(tokens, 1, "TABLE") {
		stmt.Type = StmtTypeOther
		return stmt, nil
	}
	pos := 2
	if tokenIs(tokens, pos, "IF") && tokenIs(tokens, pos+1, "EXISTS") {
		pos += 2
	}
	if tokenIs(tokens, pos, "ONLY") {
		pos++
	}
	table, next, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	stmt.Type = StmtTypeAlterTable
	setRelation(&stmt, table)
	for i := next; i < len(tokens); i++ {
		switch {
		case upper(tokens[i]) == "DROP" && tokenIs(tokens, i+1, "COLUMN"):
			stmt.DropColumn = true
			stmt.Type = StmtTypeAlterTableDropCol
		case upper(tokens[i]) == "DROP" && tokenIs(tokens, i+1, "CONSTRAINT"):
			stmt.DropConstraint = true
		case upper(tokens[i]) == "DROP" && tokenIs(tokens, i+1, "NOT") && tokenIs(tokens, i+2, "NULL"):
			stmt.DropNotNull = true
		}
	}
	return stmt, nil
}

func parseSelectTokens(tokens []string, stmt Statement) Statement {
	stmt.Type = StmtTypeSelect
	stmt.HasWhere = containsKeywordTopLevel(tokens, "WHERE")
	stmt.HasLimit = hasLimitClauseTopLevel(tokens)
	if from := indexKeywordTopLevel(tokens, "FROM"); from >= 0 {
		stmt.SelectAll = containsStarProjection(tokens[1:from])
		if from+1 < len(tokens) {
			if name, _, err := parseQualifiedIdentifier(tokens, from+1); err == nil {
				setRelation(&stmt, name)
			}
		}
	} else {
		stmt.SelectAll = containsStarProjection(tokens[1:])
	}
	return stmt
}

func parseInsertTokens(tokens []string, stmt Statement) (Statement, error) {
	pos := 1
	if tokenIs(tokens, pos, "INTO") {
		pos++
	}
	table, next, err := parseQualifiedIdentifier(tokens, pos)
	if err != nil {
		return Statement{}, err
	}
	stmt.Type = StmtTypeInsert
	setRelation(&stmt, table)
	captureInsertSelectSources(tokens, next, &stmt)
	return stmt, nil
}

// captureInsertSelectSources records read relations feeding INSERT ... SELECT.
// The full AST path already merges these; without this the portable parser
// silently missed staging copies of protected tables (audit exclusion F-014).
// As with other core FROM handling only comma-separated plain sources are
// captured; joins and subqueries remain documented full-mode coverage.
func captureInsertSelectSources(tokens []string, start int, stmt *Statement) {
	selIdx := indexKeywordTopLevelFrom(tokens, start, "SELECT")
	if selIdx < 0 {
		return
	}
	fromIdx := indexKeywordTopLevelFrom(tokens, selIdx+1, "FROM")
	if fromIdx < 0 || fromIdx+1 >= len(tokens) {
		return
	}
	pos := fromIdx + 1
	for pos < len(tokens) {
		name, next, err := parseQualifiedIdentifier(tokens, pos)
		if err != nil {
			return
		}
		addRelationQualified(stmt, name, RelationRead)
		pos = next
		if pos < len(tokens) && tokens[pos] == "," {
			pos++
			continue
		}
		return
	}
}

func parseTruncateTokens(tokens []string, stmt Statement) (Statement, error) {
	pos := 1
	if tokenIs(tokens, pos, "TABLE") {
		pos++
	}
	if tokenIs(tokens, pos, "ONLY") {
		pos++
	}
	stmt.Type = StmtTypeTruncate
	first := ""
	for pos < len(tokens) {
		name, next, err := parseQualifiedIdentifier(tokens, pos)
		if err != nil {
			break
		}
		if first == "" {
			first = name
		}
		setRelation(&stmt, name)
		pos = next
		if pos < len(tokens) && tokens[pos] == "," {
			pos++
			continue
		}
		break
	}
	if first == "" {
		return Statement{}, syntaxNear(tokens, pos)
	}
	return stmt, nil
}

func parseGrantTokens(tokens []string, stmt Statement) (Statement, error) {
	toIdx := indexKeywordTopLevel(tokens, "TO")
	if toIdx < 0 {
		return Statement{}, fmt.Errorf("syntax error at or near %q", "GRANT")
	}
	stmt.Type = StmtTypeGrant
	stmt.Grantees = parseIdentifiersUntil(tokens, toIdx+1, "WITH", "GRANTED", "BY")
	stmt.IsGrantToPublic = containsIdentifier(stmt.Grantees, "public")
	onIdx := indexKeywordTopLevel(tokens, "ON")
	if onIdx >= 0 && onIdx < toIdx {
		parseGrantObject(tokens, onIdx+1, toIdx, &stmt)
		return stmt, nil
	}
	stmt.IsRoleMembershipGrant = true
	stmt.GrantedRoles = parseIdentifiersUntil(tokens, 1, "TO")
	return stmt, nil
}

func parseRevokeTokens(tokens []string, stmt Statement) (Statement, error) {
	// REVOKE is recognized so privilege revocations do not fail the gate or get
	// misread as grants (audit finding F-001). No rules act on revokes yet.
	fromIdx := indexKeywordTopLevel(tokens, "FROM")
	if fromIdx < 0 {
		return Statement{}, fmt.Errorf("syntax error at or near %q", "REVOKE")
	}
	stmt.Type = StmtTypeRevoke
	stmt.IsRoleMembershipRevoke = !containsKeywordTopLevel(tokens[:fromIdx], "ON")
	stmt.Grantees = parseIdentifiersUntil(tokens, fromIdx+1, "CASCADE", "RESTRICT")
	return stmt, nil
}

func parseGrantObject(tokens []string, start, end int, stmt *Statement) {
	i := start
	sawObject := false
	for i < end {
		switch upper(tokens[i]) {
		case "TABLE", "TABLES", "SEQUENCE", "SEQUENCES", "FUNCTION", "FUNCTIONS", "ON", "ALL", "PRIVILEGES", "IN":
			i++
			continue
		case "SCHEMA":
			i++
			for i < end {
				name, next, err := parseQualifiedIdentifier(tokens, i)
				if err != nil {
					return
				}
				addRelationQualified(stmt, name, RelationTarget)
				if stmt.Schema == "" {
					stmt.Schema = name
					stmt.Object = name
				}
				i = next
				if i < end && tokens[i] == "," {
					i++
					continue
				}
				break
			}
			return
		case "DATABASE":
			// Database-level grants are not protected-object checks today; record
			// the object name without inventing a relation (audit finding F-018).
			if i+1 < end {
				if name, _, err := parseQualifiedIdentifier(tokens, i+1); err == nil {
					stmt.Object = name
				}
			}
			return
		case "TO":
			return
		default:
			name, next, err := parseQualifiedIdentifier(tokens, i)
			if err != nil {
				i++
				continue
			}
			setRelation(stmt, name)
			sawObject = true
			i = next
			if i < end && tokens[i] == "," {
				i++
				continue
			}
			continue
		}
	}
	if !sawObject && len(stmt.Relations) == 0 && stmt.Schema != "" {
		stmt.Object = stmt.Schema
	}
}

func parseCopyTokens(tokens []string, stmt Statement) (Statement, error) {
	stmt.Type = StmtTypeCopy
	if len(tokens) < 2 {
		return Statement{}, fmt.Errorf("syntax error at end of input")
	}
	if tokens[1] == "(" {
		if fromIdx := indexKeyword(tokens, "FROM"); fromIdx >= 0 && fromIdx+1 < len(tokens) {
			if name, _, err := parseQualifiedIdentifier(tokens, fromIdx+1); err == nil {
				setRelation(&stmt, name)
			}
		}
	} else {
		name, _, err := parseQualifiedIdentifier(tokens, 1)
		if err == nil {
			setRelation(&stmt, name)
		}
	}
	if toIdx := indexKeyword(tokens, "TO"); toIdx >= 0 && toIdx+1 < len(tokens) {
		stmt.CopyToStdout = tokenIs(tokens, toIdx+1, "STDOUT")
		stmt.CopyToProgram = tokenIs(tokens, toIdx+1, "PROGRAM")
	}
	return stmt, nil
}

func tokenizeSQL(sql string) ([]string, error) {
	var tokens []string

	for i := 0; i < len(sql); {
		ch, width := utf8.DecodeRuneInString(sql[i:])
		if ch == utf8.RuneError && width == 1 {
			// Invalid UTF-8 byte: treat as an opaque single-byte token so the
			// statement fails closed downstream instead of corrupting names.
			tokens = append(tokens, sql[i:i+1])
			i++
			continue
		}
		if unicode.IsSpace(ch) {
			i += width
			continue
		}
		if ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			i += 2
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			continue
		}
		if ch == '/' && i+1 < len(sql) && sql[i+1] == '*' {
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
		if ch == '\'' {
			token, next, err := readSingleQuoted(sql, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			i = next
			continue
		}
		if ch == '"' {
			token, next, err := readDoubleQuoted(sql, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			i = next
			continue
		}
		if ch == '$' {
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
			i += width
			for i < len(sql) {
				r, w := utf8.DecodeRuneInString(sql[i:])
				if !isIdentifierPart(r) {
					break
				}
				i += w
			}
			tokens = append(tokens, sql[start:i])
			continue
		}
		if unicode.IsDigit(ch) {
			start := i
			i += width
			for i < len(sql) {
				r, w := utf8.DecodeRuneInString(sql[i:])
				if !(unicode.IsDigit(r) || r == '.') {
					break
				}
				i += w
			}
			tokens = append(tokens, sql[start:i])
			continue
		}
		if strings.ContainsRune("(),.*=;", ch) {
			tokens = append(tokens, sql[i:i+width])
			i += width
			continue
		}
		if unicode.IsPunct(ch) || unicode.IsSymbol(ch) {
			tokens = append(tokens, sql[i:i+width])
			i += width
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
	if !isIdentifierStart(rune(token[0])) {
		return "", false
	}
	return strings.ToLower(token), true
}

func parseIdentifiersAfterKeyword(tokens []string, keyword string) []string {
	if idx := indexKeyword(tokens, keyword); idx >= 0 {
		return parseIdentifiersUntil(tokens, idx+1, "WITH", "GRANTED", "BY")
	}
	return nil
}

func parseIdentifiersUntil(tokens []string, start int, stopKeywords ...string) []string {
	stop := make(map[string]bool, len(stopKeywords))
	for _, s := range stopKeywords {
		stop[s] = true
	}
	var out []string
	for i := start; i < len(tokens); i++ {
		up := upper(tokens[i])
		if stop[up] {
			break
		}
		if name, ok := normalizeIdentifier(tokens[i]); ok {
			out = append(out, name)
		}
	}
	return out
}

func containsIdentifier(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsStarProjection(tokens []string) bool {
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "*" {
			return true
		}
		if i+2 < len(tokens) && tokens[i+1] == "." && tokens[i+2] == "*" {
			return true
		}
	}
	return false
}

func trivialWhereTokens(tokens []string) bool {
	idx := indexKeywordTopLevel(tokens, "WHERE")
	if idx < 0 {
		return false
	}
	var clause []string
	depth := 0
	for _, token := range tokens[idx+1:] {
		switch token {
		case "(":
			depth++
		case ")":
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				up := upper(token)
				switch up {
				case "RETURNING", "LIMIT", "FETCH", "ORDER", "FOR":
					clause = stripOuterParens(clause)
					return clauseIsTrivial(clause)
				case ";":
					continue
				}
				clause = append(clause, up)
			} else {
				clause = append(clause, upper(token))
			}
		}
	}
	clause = stripOuterParens(clause)
	return clauseIsTrivial(clause)
}

func clauseIsTrivial(clause []string) bool {
	switch {
	case len(clause) == 1 && clause[0] == "TRUE":
		return true
	case len(clause) == 3 && clause[0] == "1" && clause[1] == "=" && clause[2] == "1":
		return true
	default:
		return false
	}
}

func stripOuterParens(tokens []string) []string {
	for len(tokens) >= 2 && tokens[0] == "(" && tokens[len(tokens)-1] == ")" {
		tokens = tokens[1 : len(tokens)-1]
	}
	return tokens
}

func containsKeyword(tokens []string, keyword string) bool {
	return indexKeyword(tokens, keyword) >= 0
}

// indexKeywordTopLevel finds a keyword outside any parenthesis group. Keywords
// inside subqueries or CTE bodies must not satisfy outer-clause checks such as
// WHERE presence or LIMIT bounding (audit findings F-003 and F-008).
func indexKeywordTopLevel(tokens []string, keyword string) int {
	return indexKeywordTopLevelFrom(tokens, 0, keyword)
}

func indexKeywordTopLevelFrom(tokens []string, start int, keyword string) int {
	depth := 0
	for i := start; i < len(tokens); i++ {
		switch tokens[i] {
		case "(":
			depth++
		case ")":
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && upper(tokens[i]) == keyword {
				return i
			}
		}
	}
	return -1
}

func containsKeywordTopLevel(tokens []string, keyword string) bool {
	return indexKeywordTopLevel(tokens, keyword) >= 0
}

// hasLimitClauseTopLevel reports whether the statement is bounded by a
// top-level LIMIT clause or SQL-standard FETCH FIRST/NEXT ... ROW[S] form
// (audit finding F-010).
func hasLimitClauseTopLevel(tokens []string) bool {
	if containsKeywordTopLevel(tokens, "LIMIT") {
		return true
	}
	fetch := indexKeywordTopLevel(tokens, "FETCH")
	if fetch < 0 || fetch+1 >= len(tokens) {
		return false
	}
	first := upper(tokens[fetch+1])
	if first != "FIRST" && first != "NEXT" && first != "ROW" && first != "ROWS" {
		return false
	}
	for i := fetch + 1; i < len(tokens); i++ {
		up := upper(tokens[i])
		if up == "ROW" || up == "ROWS" {
			return true
		}
		switch up {
		case "ONLY":
			return false
		case "WITH", "TIES":
			return true // WITH TIES still bounds via FETCH
		}
	}
	return false
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
	i := 1
	for i < len(sql) {
		ch, width := utf8.DecodeRuneInString(sql[i:])
		if ch == '$' {
			return sql[:i+width]
		}
		if !(unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_') {
			return ""
		}
		i += width
	}
	return ""
}

func isIdentifierStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentifierPart(ch rune) bool {
	return isIdentifierStart(ch) || unicode.IsDigit(ch) || ch == '$'
}

func setRelation(stmt *Statement, name string) {
	setRelationWithRole(stmt, name, defaultRoleForStmt(stmt.Type))
}

func setRelationWithRole(stmt *Statement, name string, role RelationRole) {
	schema, rel := splitQualified(name)
	addRelation(stmt, schema, rel, role)
	stmt.Table = name
	stmt.Object = name
	stmt.Schema = schema
	if stmt.Schema == "" {
		stmt.Schema = relationSchema(name)
	}
}

func defaultRoleForStmt(t StmtType) RelationRole {
	switch t {
	case StmtTypeSelect, StmtTypeCopy:
		return RelationRead
	case StmtTypeGrant, StmtTypeRevoke, StmtTypeAlterDefaultPrivileges:
		return RelationTarget
	default:
		return RelationWrite
	}
}

func relationSchema(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		return parts[0]
	}
	return ""
}
