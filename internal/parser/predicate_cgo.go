//go:build cgo

package parser

import (
	"strconv"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

type constKind int

const (
	constUnknown constKind = iota
	constNull
	constBool
	constNumber
	constString
)

type constValue struct {
	kind   constKind
	boolV  bool
	numV   string
	strV   string
	known  bool // false => SQL UNKNOWN / non-constant
}

func classifyPredicate(node *pg_query.Node) PredicateAssessment {
	v := evalExpr(node)
	if !v.known {
		if exprHasNonConstant(node) {
			return PredicateNonTrivial
		}
		return PredicateUnknown
	}
	if v.kind == constNull {
		return PredicateUnknown
	}
	if v.kind == constBool && v.boolV {
		return PredicateAlwaysTrue
	}
	return PredicateNonTrivial
}

func evalExpr(node *pg_query.Node) constValue {
	if node == nil {
		return unknownValue()
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_AConst:
		return evalConst(n.AConst)
	case *pg_query.Node_TypeCast:
		if n.TypeCast == nil {
			return unknownValue()
		}
		return evalExpr(n.TypeCast.Arg)
	case *pg_query.Node_BoolExpr:
		return evalBoolExpr(n.BoolExpr)
	case *pg_query.Node_AExpr:
		return evalAExpr(n.AExpr)
	case *pg_query.Node_BooleanTest:
		return evalBooleanTest(n.BooleanTest)
	case *pg_query.Node_NullTest:
		// IS NULL / IS NOT NULL over non-constants is non-constant; over NULL const is foldable.
		if n.NullTest == nil {
			return unknownValue()
		}
		inner := evalExpr(n.NullTest.Arg)
		if !inner.known {
			return unknownValue()
		}
		isNull := inner.kind == constNull
		switch n.NullTest.Nulltesttype {
		case pg_query.NullTestType_IS_NULL:
			return boolValue(isNull)
		case pg_query.NullTestType_IS_NOT_NULL:
			return boolValue(!isNull)
		default:
			return unknownValue()
		}
	case *pg_query.Node_ColumnRef, *pg_query.Node_ParamRef, *pg_query.Node_FuncCall, *pg_query.Node_SubLink:
		return unknownValue()
	default:
		return unknownValue()
	}
}

func evalConst(c *pg_query.A_Const) constValue {
	if c == nil {
		return unknownValue()
	}
	if c.Isnull {
		return constValue{kind: constNull, known: true}
	}
	switch v := c.Val.(type) {
	case *pg_query.A_Const_Boolval:
		if v.Boolval == nil {
			return unknownValue()
		}
		return boolValue(v.Boolval.Boolval)
	case *pg_query.A_Const_Ival:
		if v.Ival == nil {
			return unknownValue()
		}
		return constValue{kind: constNumber, numV: strconv.FormatInt(int64(v.Ival.Ival), 10), known: true}
	case *pg_query.A_Const_Fval:
		if v.Fval == nil {
			return unknownValue()
		}
		return constValue{kind: constNumber, numV: normalizeNumber(v.Fval.Fval), known: true}
	case *pg_query.A_Const_Sval:
		if v.Sval == nil {
			return unknownValue()
		}
		return constValue{kind: constString, strV: v.Sval.Sval, known: true}
	default:
		return unknownValue()
	}
}

func evalBoolExpr(b *pg_query.BoolExpr) constValue {
	if b == nil {
		return unknownValue()
	}
	switch b.Boolop {
	case pg_query.BoolExprType_AND_EXPR:
		sawUnknown := false
		sawNull := false
		for _, arg := range b.Args {
			v := evalExpr(arg)
			if !v.known {
				sawUnknown = true
				continue
			}
			if v.kind == constNull {
				sawNull = true
				continue
			}
			if v.kind != constBool {
				return unknownValue()
			}
			if !v.boolV {
				return boolValue(false)
			}
		}
		if sawUnknown {
			return unknownValue()
		}
		if sawNull {
			return constValue{kind: constNull, known: true}
		}
		return boolValue(true)
	case pg_query.BoolExprType_OR_EXPR:
		sawUnknown := false
		sawNull := false
		for _, arg := range b.Args {
			v := evalExpr(arg)
			if !v.known {
				sawUnknown = true
				continue
			}
			if v.kind == constNull {
				sawNull = true
				continue
			}
			if v.kind != constBool {
				return unknownValue()
			}
			if v.boolV {
				return boolValue(true)
			}
		}
		if sawUnknown {
			return unknownValue()
		}
		if sawNull {
			return constValue{kind: constNull, known: true}
		}
		return boolValue(false)
	case pg_query.BoolExprType_NOT_EXPR:
		if len(b.Args) != 1 {
			return unknownValue()
		}
		v := evalExpr(b.Args[0])
		if !v.known {
			return unknownValue()
		}
		if v.kind == constNull {
			return constValue{kind: constNull, known: true}
		}
		if v.kind != constBool {
			return unknownValue()
		}
		return boolValue(!v.boolV)
	default:
		return unknownValue()
	}
}

func evalAExpr(expr *pg_query.A_Expr) constValue {
	if expr == nil {
		return unknownValue()
	}
	switch expr.Kind {
	case pg_query.A_Expr_Kind_AEXPR_OP:
		return evalAExprOp(expr)
	case pg_query.A_Expr_Kind_AEXPR_NOT_DISTINCT:
		left := evalExpr(expr.Lexpr)
		right := evalExpr(expr.Rexpr)
		if !left.known || !right.known {
			return unknownValue()
		}
		if left.kind == constNull && right.kind == constNull {
			return boolValue(true)
		}
		if left.kind == constNull || right.kind == constNull {
			return boolValue(false)
		}
		eq, ok := constEqual(left, right)
		if !ok {
			return unknownValue()
		}
		return boolValue(eq)
	case pg_query.A_Expr_Kind_AEXPR_DISTINCT:
		left := evalExpr(expr.Lexpr)
		right := evalExpr(expr.Rexpr)
		if !left.known || !right.known {
			return unknownValue()
		}
		if left.kind == constNull && right.kind == constNull {
			return boolValue(false)
		}
		if left.kind == constNull || right.kind == constNull {
			return boolValue(true)
		}
		eq, ok := constEqual(left, right)
		if !ok {
			return unknownValue()
		}
		return boolValue(!eq)
	default:
		return unknownValue()
	}
}

func evalAExprOp(expr *pg_query.A_Expr) constValue {
	op := aExprOperator(expr)
	left := evalExpr(expr.Lexpr)
	right := evalExpr(expr.Rexpr)
	if !left.known || !right.known {
		return unknownValue()
	}
	if left.kind == constNull || right.kind == constNull {
		return constValue{kind: constNull, known: true}
	}
	switch op {
	case "=":
		eq, ok := constEqual(left, right)
		if !ok {
			return unknownValue()
		}
		return boolValue(eq)
	case "<>", "!=":
		eq, ok := constEqual(left, right)
		if !ok {
			return unknownValue()
		}
		return boolValue(!eq)
	case ">", ">=", "<", "<=":
		cmp, ok := constCompare(left, right)
		if !ok {
			return unknownValue()
		}
		switch op {
		case ">":
			return boolValue(cmp > 0)
		case ">=":
			return boolValue(cmp >= 0)
		case "<":
			return boolValue(cmp < 0)
		default:
			return boolValue(cmp <= 0)
		}
	default:
		return unknownValue()
	}
}

func evalBooleanTest(bt *pg_query.BooleanTest) constValue {
	if bt == nil {
		return unknownValue()
	}
	inner := evalExpr(bt.Arg)
	if !inner.known {
		return unknownValue()
	}
	switch bt.Booltesttype {
	case pg_query.BoolTestType_IS_TRUE:
		return boolValue(inner.kind == constBool && inner.boolV)
	case pg_query.BoolTestType_IS_NOT_TRUE:
		return boolValue(!(inner.kind == constBool && inner.boolV))
	case pg_query.BoolTestType_IS_FALSE:
		return boolValue(inner.kind == constBool && !inner.boolV)
	case pg_query.BoolTestType_IS_NOT_FALSE:
		return boolValue(!(inner.kind == constBool && !inner.boolV))
	case pg_query.BoolTestType_IS_UNKNOWN:
		return boolValue(inner.kind == constNull)
	case pg_query.BoolTestType_IS_NOT_UNKNOWN:
		return boolValue(inner.kind != constNull)
	default:
		return unknownValue()
	}
}

func aExprOperator(expr *pg_query.A_Expr) string {
	if expr == nil || len(expr.Name) == 0 {
		return ""
	}
	if s := expr.Name[0].GetString_(); s != nil {
		return s.Sval
	}
	return ""
}

func constEqual(a, b constValue) (bool, bool) {
	if a.kind != b.kind {
		if a.kind == constNumber && b.kind == constNumber {
			return a.numV == b.numV, true
		}
		return false, false
	}
	switch a.kind {
	case constBool:
		return a.boolV == b.boolV, true
	case constNumber:
		return a.numV == b.numV, true
	case constString:
		return a.strV == b.strV, true
	default:
		return false, false
	}
}

func constCompare(a, b constValue) (int, bool) {
	if a.kind != constNumber || b.kind != constNumber {
		return 0, false
	}
	af, aerr := strconv.ParseFloat(a.numV, 64)
	bf, berr := strconv.ParseFloat(b.numV, 64)
	if aerr != nil || berr != nil {
		return 0, false
	}
	switch {
	case af < bf:
		return -1, true
	case af > bf:
		return 1, true
	default:
		return 0, true
	}
}

func exprHasNonConstant(node *pg_query.Node) bool {
	if node == nil {
		return false
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_ColumnRef, *pg_query.Node_ParamRef, *pg_query.Node_FuncCall, *pg_query.Node_SubLink:
		return true
	case *pg_query.Node_BoolExpr:
		if n.BoolExpr == nil {
			return false
		}
		for _, arg := range n.BoolExpr.Args {
			if exprHasNonConstant(arg) {
				return true
			}
		}
	case *pg_query.Node_AExpr:
		if n.AExpr == nil {
			return false
		}
		return exprHasNonConstant(n.AExpr.Lexpr) || exprHasNonConstant(n.AExpr.Rexpr)
	case *pg_query.Node_TypeCast:
		if n.TypeCast == nil {
			return false
		}
		return exprHasNonConstant(n.TypeCast.Arg)
	case *pg_query.Node_NullTest:
		if n.NullTest == nil {
			return false
		}
		return exprHasNonConstant(n.NullTest.Arg)
	case *pg_query.Node_BooleanTest:
		if n.BooleanTest == nil {
			return false
		}
		return exprHasNonConstant(n.BooleanTest.Arg)
	}
	return false
}

func boolValue(v bool) constValue {
	return constValue{kind: constBool, boolV: v, known: true}
}

func unknownValue() constValue {
	return constValue{known: false}
}

func normalizeNumber(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return s
}
