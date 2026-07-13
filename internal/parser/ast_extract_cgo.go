//go:build cgo

package parser

import (
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

func extractStatementFromNode(node *pg_query.Node, rawSQL string, startLine int) (Statement, error) {
	stmt := Statement{
		RawSQL:       strings.TrimSpace(rawSQL),
		StartLine:    startLine,
		Completeness: AnalysisComplete,
		Predicate:    PredicateAbsent,
	}
	if node == nil {
		markIncomplete(&stmt, AnalysisUnsupported, "missing_ast_node")
		stmt.Type = StmtTypeOther
		syncConvenienceFields(&stmt)
		return stmt, nil
	}

	switch n := node.Node.(type) {
	case *pg_query.Node_DeleteStmt:
		extractDelete(&stmt, n.DeleteStmt)
	case *pg_query.Node_UpdateStmt:
		extractUpdate(&stmt, n.UpdateStmt)
	case *pg_query.Node_SelectStmt:
		extractSelect(&stmt, n.SelectStmt)
	case *pg_query.Node_InsertStmt:
		extractInsert(&stmt, n.InsertStmt)
	case *pg_query.Node_TruncateStmt:
		extractTruncate(&stmt, n.TruncateStmt)
	case *pg_query.Node_DropStmt:
		extractDrop(&stmt, n.DropStmt)
	case *pg_query.Node_DropdbStmt:
		extractDropDatabase(&stmt, n.DropdbStmt)
	case *pg_query.Node_AlterTableStmt:
		extractAlterTable(&stmt, n.AlterTableStmt)
	case *pg_query.Node_AlterDefaultPrivilegesStmt:
		extractAlterDefaultPrivileges(&stmt, n.AlterDefaultPrivilegesStmt)
	case *pg_query.Node_GrantStmt:
		extractGrant(&stmt, n.GrantStmt)
	case *pg_query.Node_GrantRoleStmt:
		extractGrantRole(&stmt, n.GrantRoleStmt)
	case *pg_query.Node_CopyStmt:
		extractCopy(&stmt, n.CopyStmt)
	default:
		stmt.Type = StmtTypeOther
		markIncomplete(&stmt, AnalysisUnsupported, "unsupported_statement_type")
	}

	stmt.IncompleteReasons = ensureReasonsBounded(stmt.IncompleteReasons, 5)
	syncConvenienceFields(&stmt)
	return stmt, nil
}

func extractDelete(stmt *Statement, node *pg_query.DeleteStmt) {
	stmt.Type = StmtTypeDelete
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_delete_node")
		return
	}
	addRangeVar(stmt, node.Relation, RelationWrite)
	walkFromList(stmt, node.UsingClause, RelationRead, nil)
	applyPredicate(stmt, node.WhereClause)
	walkWithClause(stmt, node.WithClause)
}

func extractUpdate(stmt *Statement, node *pg_query.UpdateStmt) {
	stmt.Type = StmtTypeUpdate
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_update_node")
		return
	}
	addRangeVar(stmt, node.Relation, RelationWrite)
	walkFromList(stmt, node.FromClause, RelationRead, nil)
	applyPredicate(stmt, node.WhereClause)
	walkWithClause(stmt, node.WithClause)
}

func extractSelect(stmt *Statement, node *pg_query.SelectStmt) {
	stmt.Type = StmtTypeSelect
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_select_node")
		return
	}
	extractSelectBody(stmt, node, nil)
}

func extractSelectBody(stmt *Statement, node *pg_query.SelectStmt, cteNames map[string]struct{}) {
	if node == nil {
		return
	}
	if node.WithClause != nil {
		if cteNames == nil {
			cteNames = map[string]struct{}{}
		}
		walkWithClauseInto(stmt, node.WithClause, cteNames)
	}
	if node.Op != pg_query.SetOperation_SET_OPERATION_UNDEFINED && node.Op != pg_query.SetOperation_SETOP_NONE {
		extractSelectBody(stmt, node.Larg, cteNames)
		extractSelectBody(stmt, node.Rarg, cteNames)
		return
	}
	stmt.SelectAll = selectListHasStar(node.TargetList)
	stmt.HasLimit = node.LimitCount != nil
	walkFromList(stmt, node.FromClause, RelationRead, cteNames)
	if node.WhereClause != nil {
		// SELECT where clauses are not used by trivial-write rules, but keep presence.
		if stmt.Predicate == PredicateAbsent {
			stmt.Predicate = PredicateNonTrivial
		}
		_ = classifyPredicate(node.WhereClause)
	}
	walkExpressionRelations(stmt, node.WhereClause, RelationRead, cteNames)
	for _, target := range node.TargetList {
		walkExpressionRelations(stmt, target, RelationRead, cteNames)
	}
}

func extractInsert(stmt *Statement, node *pg_query.InsertStmt) {
	stmt.Type = StmtTypeInsert
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_insert_node")
		return
	}
	addRangeVar(stmt, node.Relation, RelationWrite)
	if node.SelectStmt != nil {
		if sel := node.SelectStmt.GetSelectStmt(); sel != nil {
			tmp := Statement{Completeness: AnalysisComplete}
			extractSelectBody(&tmp, sel, nil)
			mergeSemantic(stmt, tmp)
		} else {
			walkNodeRelations(stmt, node.SelectStmt, RelationRead, nil)
		}
	}
	walkWithClause(stmt, node.WithClause)
}

func extractTruncate(stmt *Statement, node *pg_query.TruncateStmt) {
	stmt.Type = StmtTypeTruncate
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_truncate_node")
		return
	}
	for _, rel := range node.Relations {
		if rv := rel.GetRangeVar(); rv != nil {
			addRangeVar(stmt, rv, RelationWrite)
		} else {
			markIncomplete(stmt, AnalysisPartial, "truncate_unrecognized_relation_node")
		}
	}
}

func extractDrop(stmt *Statement, node *pg_query.DropStmt) {
	if node == nil {
		stmt.Type = StmtTypeOther
		markIncomplete(stmt, AnalysisPartial, "missing_drop_node")
		return
	}
	switch node.RemoveType {
	case pg_query.ObjectType_OBJECT_TABLE:
		stmt.Type = StmtTypeDropTable
		for _, obj := range node.Objects {
			name, schema, ok := objectNameFromNode(obj)
			if !ok {
				markIncomplete(stmt, AnalysisPartial, "drop_table_unrecognized_object")
				continue
			}
			addRelation(stmt, schema, name, RelationWrite)
		}
	case pg_query.ObjectType_OBJECT_SCHEMA:
		stmt.Type = StmtTypeDropSchema
		for _, obj := range node.Objects {
			name, _, ok := objectNameFromNode(obj)
			if !ok {
				markIncomplete(stmt, AnalysisPartial, "drop_schema_unrecognized_object")
				continue
			}
			stmt.Object = name
			stmt.Schema = name
			addRelation(stmt, "", name, RelationTarget)
		}
	default:
		stmt.Type = StmtTypeOther
		markIncomplete(stmt, AnalysisUnsupported, "unsupported_drop_object_type")
	}
}

func extractDropDatabase(stmt *Statement, node *pg_query.DropdbStmt) {
	stmt.Type = StmtTypeDropDatabase
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_dropdb_node")
		return
	}
	stmt.Object = node.Dbname
}

func extractAlterTable(stmt *Statement, node *pg_query.AlterTableStmt) {
	if node == nil {
		stmt.Type = StmtTypeAlterTable
		markIncomplete(stmt, AnalysisPartial, "missing_alter_table_node")
		return
	}
	stmt.Type = StmtTypeAlterTable
	addRangeVar(stmt, node.Relation, RelationWrite)
	for _, cmdNode := range node.Cmds {
		cmd := cmdNode.GetAlterTableCmd()
		if cmd == nil {
			markIncomplete(stmt, AnalysisPartial, "alter_table_unrecognized_cmd")
			continue
		}
		switch cmd.Subtype {
		case pg_query.AlterTableType_AT_DropColumn:
			stmt.DropColumn = true
			stmt.Type = StmtTypeAlterTableDropCol
		case pg_query.AlterTableType_AT_DropConstraint:
			stmt.DropConstraint = true
		case pg_query.AlterTableType_AT_DropNotNull:
			stmt.DropNotNull = true
		}
	}
}

func extractAlterDefaultPrivileges(stmt *Statement, node *pg_query.AlterDefaultPrivilegesStmt) {
	stmt.Type = StmtTypeAlterDefaultPrivileges
	if node == nil || node.Action == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_alter_default_privileges_action")
		return
	}
	applyGrantCommon(stmt, node.Action)
}

func extractGrant(stmt *Statement, node *pg_query.GrantStmt) {
	stmt.Type = StmtTypeGrant
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_grant_node")
		return
	}
	applyGrantCommon(stmt, node)
}

func extractGrantRole(stmt *Statement, node *pg_query.GrantRoleStmt) {
	stmt.Type = StmtTypeGrant
	stmt.IsRoleMembershipGrant = true
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_grant_role_node")
		return
	}
	for _, roleNode := range node.GrantedRoles {
		if priv := roleNode.GetAccessPriv(); priv != nil && priv.PrivName != "" {
			stmt.GrantedRoles = append(stmt.GrantedRoles, strings.ToLower(priv.PrivName))
		} else {
			markIncomplete(stmt, AnalysisPartial, "grant_role_unrecognized_role_node")
		}
	}
	for _, g := range node.GranteeRoles {
		name, isPublic, ok := roleSpecName(g.GetRoleSpec())
		if !ok {
			markIncomplete(stmt, AnalysisPartial, "grant_role_unrecognized_grantee")
			continue
		}
		if isPublic {
			stmt.IsGrantToPublic = true
			stmt.Grantees = append(stmt.Grantees, "public")
		} else {
			stmt.Grantees = append(stmt.Grantees, name)
		}
	}
}

func applyGrantCommon(stmt *Statement, node *pg_query.GrantStmt) {
	if node == nil {
		return
	}
	for _, g := range node.Grantees {
		name, isPublic, ok := roleSpecName(g.GetRoleSpec())
		if !ok {
			markIncomplete(stmt, AnalysisPartial, "grant_unrecognized_grantee")
			continue
		}
		if isPublic {
			stmt.IsGrantToPublic = true
			stmt.Grantees = append(stmt.Grantees, "public")
		} else if name != "" {
			stmt.Grantees = append(stmt.Grantees, name)
		}
	}
	switch node.Objtype {
	case pg_query.ObjectType_OBJECT_TABLE:
		for _, obj := range node.Objects {
			if rv := obj.GetRangeVar(); rv != nil {
				addRangeVar(stmt, rv, RelationTarget)
			} else {
				markIncomplete(stmt, AnalysisPartial, "grant_table_unrecognized_object")
			}
		}
	case pg_query.ObjectType_OBJECT_SCHEMA:
		for _, obj := range node.Objects {
			name, _, ok := objectNameFromNode(obj)
			if !ok {
				markIncomplete(stmt, AnalysisPartial, "grant_schema_unrecognized_object")
				continue
			}
			stmt.Schema = name
			stmt.Object = name
			addRelation(stmt, "", name, RelationTarget)
		}
	default:
		if node.Targtype == pg_query.GrantTargetType_ACL_TARGET_ALL_IN_SCHEMA {
			for _, obj := range node.Objects {
				name, _, ok := objectNameFromNode(obj)
				if ok {
					stmt.Schema = name
					stmt.Object = name
					addRelation(stmt, "", name, RelationTarget)
				}
			}
		} else if len(node.Objects) > 0 {
			markIncomplete(stmt, AnalysisPartial, "grant_partial_object_type")
		}
	}
}

func extractCopy(stmt *Statement, node *pg_query.CopyStmt) {
	stmt.Type = StmtTypeCopy
	if node == nil {
		markIncomplete(stmt, AnalysisPartial, "missing_copy_node")
		return
	}
	stmt.CopyToProgram = node.IsProgram
	stmt.CopyToStdout = !node.IsFrom && !node.IsProgram && node.Filename == ""
	if node.Relation != nil {
		addRangeVar(stmt, node.Relation, RelationRead)
	}
	if node.Query != nil {
		if sel := node.Query.GetSelectStmt(); sel != nil {
			tmp := Statement{Completeness: AnalysisComplete, Type: StmtTypeSelect}
			extractSelectBody(&tmp, sel, nil)
			mergeSemantic(stmt, tmp)
		} else {
			walkNodeRelations(stmt, node.Query, RelationRead, nil)
			markIncomplete(stmt, AnalysisPartial, "copy_query_unrecognized_shape")
		}
	}
}

func walkWithClause(stmt *Statement, with *pg_query.WithClause) {
	if with == nil {
		return
	}
	cteNames := map[string]struct{}{}
	walkWithClauseInto(stmt, with, cteNames)
}

func walkWithClauseInto(stmt *Statement, with *pg_query.WithClause, cteNames map[string]struct{}) {
	if with == nil {
		return
	}
	for _, cteNode := range with.Ctes {
		cte := cteNode.GetCommonTableExpr()
		if cte == nil {
			markIncomplete(stmt, AnalysisPartial, "cte_unrecognized_node")
			continue
		}
		if cte.Ctename != "" {
			cteNames[strings.ToLower(cte.Ctename)] = struct{}{}
		}
		if cte.Ctequery == nil {
			markIncomplete(stmt, AnalysisPartial, "cte_missing_query")
			continue
		}
		nested, err := extractStatementFromNode(cte.Ctequery, "", stmt.StartLine)
		if err != nil {
			markIncomplete(stmt, AnalysisPartial, "cte_extract_failed")
			continue
		}
		// Merge nested relations into parent and keep nested statement for rule recursion.
		mergeSemantic(stmt, nested)
		stmt.Nested = append(stmt.Nested, nested)
	}
}

func walkFromList(stmt *Statement, nodes []*pg_query.Node, role RelationRole, cteNames map[string]struct{}) {
	for _, n := range nodes {
		walkFromItem(stmt, n, role, cteNames)
	}
}

func walkFromItem(stmt *Statement, node *pg_query.Node, role RelationRole, cteNames map[string]struct{}) {
	if node == nil {
		return
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_RangeVar:
		if n.RangeVar != nil {
			if cteNames != nil {
				if _, ok := cteNames[strings.ToLower(n.RangeVar.Relname)]; ok && n.RangeVar.Schemaname == "" {
					return
				}
			}
			addRangeVar(stmt, n.RangeVar, role)
		}
	case *pg_query.Node_JoinExpr:
		if n.JoinExpr == nil {
			return
		}
		walkFromItem(stmt, n.JoinExpr.Larg, role, cteNames)
		walkFromItem(stmt, n.JoinExpr.Rarg, role, cteNames)
		walkExpressionRelations(stmt, n.JoinExpr.Quals, RelationRead, cteNames)
	case *pg_query.Node_RangeSubselect:
		if n.RangeSubselect == nil || n.RangeSubselect.Subquery == nil {
			markIncomplete(stmt, AnalysisPartial, "subselect_missing")
			return
		}
		if sel := n.RangeSubselect.Subquery.GetSelectStmt(); sel != nil {
			tmp := Statement{Completeness: AnalysisComplete, Type: StmtTypeSelect}
			extractSelectBody(&tmp, sel, cteNames)
			mergeSemantic(stmt, tmp)
			stmt.Nested = append(stmt.Nested, tmp)
		} else {
			walkNodeRelations(stmt, n.RangeSubselect.Subquery, role, cteNames)
			markIncomplete(stmt, AnalysisPartial, "subselect_unrecognized")
		}
	case *pg_query.Node_RangeFunction, *pg_query.Node_RangeTableFunc, *pg_query.Node_RangeTableSample:
		markIncomplete(stmt, AnalysisPartial, "unsupported_from_item")
	default:
		markIncomplete(stmt, AnalysisPartial, "unrecognized_from_item")
	}
}

func walkNodeRelations(stmt *Statement, node *pg_query.Node, role RelationRole, cteNames map[string]struct{}) {
	if node == nil {
		return
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_RangeVar:
		addRangeVar(stmt, n.RangeVar, role)
	case *pg_query.Node_SelectStmt:
		tmp := Statement{Completeness: AnalysisComplete, Type: StmtTypeSelect}
		extractSelectBody(&tmp, n.SelectStmt, cteNames)
		mergeSemantic(stmt, tmp)
	case *pg_query.Node_JoinExpr:
		walkFromItem(stmt, node, role, cteNames)
	default:
		walkExpressionRelations(stmt, node, role, cteNames)
	}
}

func walkExpressionRelations(stmt *Statement, node *pg_query.Node, role RelationRole, cteNames map[string]struct{}) {
	if node == nil {
		return
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_SubLink:
		if n.SubLink != nil && n.SubLink.Subselect != nil {
			if sel := n.SubLink.Subselect.GetSelectStmt(); sel != nil {
				tmp := Statement{Completeness: AnalysisComplete, Type: StmtTypeSelect}
				extractSelectBody(&tmp, sel, cteNames)
				mergeSemantic(stmt, tmp)
				stmt.Nested = append(stmt.Nested, tmp)
			} else {
				markIncomplete(stmt, AnalysisPartial, "sublink_unrecognized")
			}
		}
	case *pg_query.Node_AExpr:
		if n.AExpr != nil {
			walkExpressionRelations(stmt, n.AExpr.Lexpr, role, cteNames)
			walkExpressionRelations(stmt, n.AExpr.Rexpr, role, cteNames)
		}
	case *pg_query.Node_BoolExpr:
		if n.BoolExpr != nil {
			for _, arg := range n.BoolExpr.Args {
				walkExpressionRelations(stmt, arg, role, cteNames)
			}
		}
	case *pg_query.Node_NullTest:
		if n.NullTest != nil {
			walkExpressionRelations(stmt, n.NullTest.Arg, role, cteNames)
		}
	case *pg_query.Node_BooleanTest:
		if n.BooleanTest != nil {
			walkExpressionRelations(stmt, n.BooleanTest.Arg, role, cteNames)
		}
	case *pg_query.Node_TypeCast:
		if n.TypeCast != nil {
			walkExpressionRelations(stmt, n.TypeCast.Arg, role, cteNames)
		}
	case *pg_query.Node_FuncCall:
		if n.FuncCall != nil {
			for _, arg := range n.FuncCall.Args {
				walkExpressionRelations(stmt, arg, role, cteNames)
			}
		}
	case *pg_query.Node_ResTarget:
		if n.ResTarget != nil {
			walkExpressionRelations(stmt, n.ResTarget.Val, role, cteNames)
		}
	case *pg_query.Node_List:
		if n.List != nil {
			for _, item := range n.List.Items {
				walkExpressionRelations(stmt, item, role, cteNames)
			}
		}
	}
}

func applyPredicate(stmt *Statement, where *pg_query.Node) {
	if where == nil {
		stmt.Predicate = PredicateAbsent
		return
	}
	stmt.Predicate = classifyPredicate(where)
	if stmt.Predicate == PredicateUnknown {
		markIncomplete(stmt, AnalysisPartial, "predicate_classification_unknown")
	}
}

func addRangeVar(stmt *Statement, rv *pg_query.RangeVar, role RelationRole) {
	if rv == nil || rv.Relname == "" {
		return
	}
	addRelation(stmt, rv.Schemaname, rv.Relname, role)
}

func objectNameFromNode(node *pg_query.Node) (name, schema string, ok bool) {
	if node == nil {
		return "", "", false
	}
	if s := node.GetString_(); s != nil {
		return s.Sval, "", true
	}
	if list := node.GetList(); list != nil {
		var parts []string
		for _, item := range list.Items {
			if s := item.GetString_(); s != nil {
				parts = append(parts, s.Sval)
			}
		}
		if len(parts) == 0 {
			return "", "", false
		}
		if len(parts) == 1 {
			return parts[0], "", true
		}
		return parts[len(parts)-1], strings.Join(parts[:len(parts)-1], "."), true
	}
	if rv := node.GetRangeVar(); rv != nil {
		return rv.Relname, rv.Schemaname, rv.Relname != ""
	}
	return "", "", false
}

func roleSpecName(rs *pg_query.RoleSpec) (name string, isPublic bool, ok bool) {
	if rs == nil {
		return "", false, false
	}
	if rs.Roletype == pg_query.RoleSpecType_ROLESPEC_PUBLIC {
		return "public", true, true
	}
	if rs.Rolename != "" {
		return strings.ToLower(rs.Rolename), false, true
	}
	return "", false, false
}

func selectListHasStar(targets []*pg_query.Node) bool {
	for _, t := range targets {
		rt := t.GetResTarget()
		if rt == nil || rt.Val == nil {
			continue
		}
		if col := rt.Val.GetColumnRef(); col != nil {
			for _, field := range col.Fields {
				if field.GetAStar() != nil {
					return true
				}
			}
		}
	}
	return false
}

func mergeSemantic(dst *Statement, src Statement) {
	for _, rel := range src.Relations {
		addRelation(dst, rel.Schema, rel.Name, rel.Role)
	}
	if src.SelectAll {
		dst.SelectAll = true
	}
	if src.HasLimit {
		dst.HasLimit = true
	}
	if src.Completeness == AnalysisUnsupported {
		markIncomplete(dst, AnalysisUnsupported, firstReason(src))
	} else if src.Completeness == AnalysisPartial {
		for _, reason := range src.IncompleteReasons {
			markIncomplete(dst, AnalysisPartial, reason)
		}
	}
	// Nested mutations: if nested DELETE/UPDATE has absent/trivial predicate metadata,
	// surface via Nested for rule recursion (already appended by caller).
}

func firstReason(stmt Statement) string {
	if len(stmt.IncompleteReasons) == 0 {
		return "nested_incomplete"
	}
	return stmt.IncompleteReasons[0]
}
