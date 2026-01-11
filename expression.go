package main

import (
	// "fmt"
	"fmt"
	"strings"
)

type ExprType int

func getPadding(depth int) string {
	if depth == 0 {
		return ""
	}
	return strings.Repeat("│   ", depth-1) + "└── "
}

const (
	EXPR_NOT = iota
	EXPR_OR
	EXPR_AND
	EXPR_ORLIST
	EXPR_ANDLIST
	EXPR_PAREN
	EXPR_VAR
)

type Expression interface {
	temp_evaluate() bool
	print(depth int) string
	is_equal(rhs Expression) bool
	optimize_stage1() (Expression, bool)
	optimize_stage2() (Expression, bool)
	pre_compile(temp_counter int) ([]Operation, int)
}

// NOT EXPRESSION
type NotExpr struct {
	expr Expression
}

func (e *NotExpr) temp_evaluate() bool {
	return !e.expr.temp_evaluate()
}

func (e *NotExpr) print(depth int) string {
	return getPadding(depth) + "NOT\n" + e.expr.print(depth+1)
}

func (e *NotExpr) is_equal(rhs Expression) bool {
	rhs_not, ok := rhs.(*NotExpr)
	if !ok {
		return false
	}
	return e.expr.is_equal(rhs_not.expr)
}

func (e *NotExpr) optimize_stage1() (Expression, bool) {
	var has_change bool
	e.expr, has_change = e.expr.optimize_stage1()

	// NOT NOT X === X
	if inner_not, ok := e.expr.(*NotExpr); ok {
		return inner_not.expr, true
	}

	// NOT (X) === NOT X
	if inner_paren, ok := e.expr.(*ParenExpr); ok {
		return &NotExpr{
			expr: inner_paren.expr,
		}, true
	}

	return e, has_change
}

func (e *NotExpr) optimize_stage2() (Expression, bool) {
	var has_change bool
	e.expr, has_change = e.expr.optimize_stage2()

	// NOT (X OR Y) = (NOT X) AND (NOT Y)
	if inner_or, ok := e.expr.(*OrExpr); ok {
		return &AndExpr{
			lhs: &NotExpr{inner_or.lhs},
			rhs: &NotExpr{inner_or.rhs},
		}, true
	}

	// NOT (X AND Y) = (NOT X) OR (NOT Y)
	if inner_and, ok := e.expr.(*AndExpr); ok {
		return &OrExpr{
			lhs: &NotExpr{inner_and.lhs},
			rhs: &NotExpr{inner_and.rhs},
		}, true
	}

	// NOT ORLIST(X1, X2, ..., XN) = ANDLIST(NOT X1, NOT X2, ..., NOT XN)
	if inner_orlist, ok := e.expr.(*OrListExpr); ok {
		for i := range len(inner_orlist.exprs) {
			inner_orlist.exprs[i] = &NotExpr{
				expr: inner_orlist.exprs[i],
			}
		}
		return &AndListExpr{
			exprs: inner_orlist.exprs,
		}, true
	}

	// NOT ANDLIST(X1, X2, ..., XN) = ORLIST(NOT X1, NOT X2, ..., NOT XN)
	if inner_andlist, ok := e.expr.(*AndListExpr); ok {
		for i := range len(inner_andlist.exprs) {
			inner_andlist.exprs[i] = &NotExpr{
				expr: inner_andlist.exprs[i],
			}
		}
		return &OrListExpr{
			exprs: inner_andlist.exprs,
		}, true
	}

	return e, has_change
}

func (e *NotExpr) pre_compile(temp_counter int) ([]Operation, int) {	
	var ops []Operation
	ops, temp_counter = e.expr.pre_compile(temp_counter)
	ops = append(ops, Operation{
		result_var: "*" + fmt.Sprint(temp_counter),
		op_type: OP_NOT,
		operands: []string{ops[len(ops) - 1].result_var},
	})

	return ops, temp_counter + 1
}

// OR EXPRESSION
type OrExpr struct {
	lhs Expression
	rhs Expression
}

func (e *OrExpr) temp_evaluate() bool {
	return e.lhs.temp_evaluate() || e.rhs.temp_evaluate()
}

func (e *OrExpr) print(depth int) string {
	return getPadding(depth) + "OR\n" + e.lhs.print(depth+1) + "\n" + e.rhs.print(depth+1)
}

func (e *OrExpr) is_equal(rhs Expression) bool {
	rhs_or, ok := rhs.(*OrExpr)
	if !ok {
		return false
	}
	return e.lhs.is_equal(rhs_or.lhs) && e.rhs.is_equal(rhs_or.rhs)
}

func (e *OrExpr) optimize_stage1() (Expression, bool) {
	var has_change bool = false
	var c1 bool
	var c2 bool
	e.lhs, c1 = e.lhs.optimize_stage1()
	e.rhs, c2 = e.rhs.optimize_stage1()
	has_change = (c1 || c2)

	// X OR X === X
	if e.lhs.is_equal(e.rhs) {
		return e.lhs, true
	}

	// 0 OR X === X
	if lhs_var, ok := e.lhs.(*VarExpr); ok && lhs_var.name == "0" {
		return e.rhs, true
	}

	// X OR 0 === X
	if rhs_var, ok := e.rhs.(*VarExpr); ok && rhs_var.name == "0" {
		return e.lhs, true
	}

	// 1 OR X === 1
	if lhs_var, ok := e.lhs.(*VarExpr); ok && lhs_var.name == "1" {
		return e.lhs, true
	}

	// X OR 1 === 1
	if rhs_var, ok := e.rhs.(*VarExpr); ok && rhs_var.name == "1" {
		return e.rhs, true
	}

	// X OR NOT X === 1
	if rhs_not, ok := e.rhs.(*NotExpr); ok && e.lhs.is_equal(rhs_not.expr) {
		return &VarExpr{name: "1"}, true
	}

	// NOT X OR X === 1
	if lhs_not, ok := e.lhs.(*NotExpr); ok && e.rhs.is_equal(lhs_not.expr) {
		return &VarExpr{name: "1"}, true
	}

	return e, has_change
}

func (e *OrExpr) optimize_stage2() (Expression, bool) {
	var has_change bool
	var c1 bool
	var c2 bool
	e.lhs, c1 = e.lhs.optimize_stage2()
	e.rhs, c2 = e.rhs.optimize_stage2()
	has_change = (c1 || c2)

	lhs := e.lhs
	rhs := e.rhs

	// Ignore redundant parenthesis
	if lhs_paren, ok := lhs.(*ParenExpr); ok {
		lhs = lhs_paren.expr
	}
	if rhs_paren, ok := rhs.(*ParenExpr); ok {
		rhs = rhs_paren.expr
	}

	//      OR
	//     /  \
	//    OR   Z
	//   /  \
	//  X    Y
	// Is equivalent to ORLIST(X, Y, Z)
	if lhs_or, ok := lhs.(*OrExpr); ok {
		return &OrListExpr{
			exprs: []Expression{lhs_or.lhs, lhs_or.rhs, rhs},
		}, true
	}
	//      OR
	//     /  \
	//    X   OR
	//       /  \
	//      Y    Z
	// Is equivalent to ORLIST(X, Y, Z)
	if rhs_or, ok := rhs.(*OrExpr); ok {
		return &OrListExpr{
			exprs: []Expression{lhs, rhs_or.lhs, rhs_or.rhs},
		}, true
	}

	//      OR
	//     /  \
	// ORLIST  Z
	// Is equivalent to adding one more element to the orlist
	if lhs_orlist, ok := lhs.(*OrListExpr); ok {
		lhs_orlist.exprs = append(lhs_orlist.exprs, rhs)
		return lhs_orlist, true
	}
	//      OR
	//     /  \
	//    X ORLIST
	//       /  \
	//      Y    Z
	// Is equivalent to adding one more element to the orlist
	if rhs_orlist, ok := rhs.(*OrListExpr); ok {
		rhs_orlist.exprs = append(rhs_orlist.exprs, lhs)
		return rhs_orlist, true
	}

	return e, has_change
}

func (e *OrExpr) pre_compile(temp_counter int) ([]Operation, int) {	
	var ops_left []Operation
	var ops_right []Operation
	ops_left, temp_counter = e.lhs.pre_compile(temp_counter)
	ops_right, temp_counter = e.rhs.pre_compile(temp_counter)
	left_operand := ops_left[len(ops_left) - 1].result_var
	right_operand := ops_right[len(ops_right) - 1].result_var

	ops_left = append(ops_left, ops_right...)
	ops_left = append(ops_left, Operation{
		result_var: "*"+fmt.Sprint(temp_counter),
		op_type: OP_OR,
		operands: []string{left_operand, right_operand},
	})

	return ops_left, temp_counter + 1
}

// AND EXPRESSION
type AndExpr struct {
	lhs Expression
	rhs Expression
}

func (e *AndExpr) temp_evaluate() bool {
	return e.lhs.temp_evaluate() && e.rhs.temp_evaluate()
}

func (e *AndExpr) print(depth int) string {
	return getPadding(depth) + "AND\n" + e.lhs.print(depth+1) + "\n" + e.rhs.print(depth+1)
}

func (e *AndExpr) is_equal(rhs Expression) bool {
	rhs_and, ok := rhs.(*AndExpr)
	if !ok {
		return false
	}
	return e.lhs.is_equal(rhs_and.lhs) && e.rhs.is_equal(rhs_and.rhs)
}

func (e *AndExpr) optimize_stage1() (Expression, bool) {
	var has_change bool = false
	var c1 bool
	var c2 bool
	e.lhs, c1 = e.lhs.optimize_stage1()
	e.rhs, c2 = e.rhs.optimize_stage1()
	has_change = (c1 || c2)

	// X AND X === X
	if e.lhs.is_equal(e.rhs) {
		return e.lhs, true
	}

	// 0 AND X === 0
	if lhs_var, ok := e.lhs.(*VarExpr); ok && lhs_var.name == "0" {
		return e.lhs, true
	}

	// X AND 0 === 0
	if rhs_var, ok := e.rhs.(*VarExpr); ok && rhs_var.name == "0" {
		return e.rhs, true
	}

	// 1 AND X === X
	if lhs_var, ok := e.lhs.(*VarExpr); ok && lhs_var.name == "1" {
		return e.rhs, true
	}

	// X AND 1 === X
	if rhs_var, ok := e.rhs.(*VarExpr); ok && rhs_var.name == "1" {
		return e.lhs, true
	}

	// X AND NOT X === 0
	if rhs_not, ok := e.rhs.(*NotExpr); ok && e.lhs.is_equal(rhs_not.expr) {
		return &VarExpr{name: "0"}, true
	}

	// NOT X AND X === 0
	if lhs_not, ok := e.lhs.(*NotExpr); ok && e.rhs.is_equal(lhs_not.expr) {
		return &VarExpr{name: "0"}, true
	}
	return e, has_change
}

func (e *AndExpr) optimize_stage2() (Expression, bool) {
	var has_change bool
	var c1 bool
	var c2 bool
	e.lhs, c1 = e.lhs.optimize_stage2()
	e.rhs, c2 = e.rhs.optimize_stage2()
	has_change = (c1 || c2)

	lhs := e.lhs
	rhs := e.rhs

	// Ignore redundant parenthesis
	if lhs_paren, ok := lhs.(*ParenExpr); ok {
		lhs = lhs_paren.expr
	}
	if rhs_paren, ok := rhs.(*ParenExpr); ok {
		rhs = rhs_paren.expr
	}

	//      AND
	//     /  \
	//   AND   Z
	//   /  \
	//  X    Y
	// Is equivalent to ANDLIST(X, Y, Z)
	if lhs_and, ok := lhs.(*AndExpr); ok {
		return &AndListExpr{
			exprs: []Expression{lhs_and.lhs, lhs_and.rhs, rhs},
		}, true
	}
	//     AND
	//     /  \
	//    X  AND
	//       /  \
	//      Y    Z
	// Is equivalent to ORLIST(X, Y, Z)
	if rhs_and, ok := rhs.(*AndExpr); ok {
		return &AndListExpr{
			exprs: []Expression{lhs, rhs_and.lhs, rhs_and.rhs},
		}, true
	}

	//     AND
	//     /  \
	// ANDLIST Z
	// Is equivalent to adding one more element to the orlist
	if lhs_andlist, ok := lhs.(*AndListExpr); ok {
		lhs_andlist.exprs = append(lhs_andlist.exprs, rhs)
		return lhs_andlist, true
	}
	//     AND
	//     /  \
	//    X ANDLIST
	//       /  \
	//      Y    Z
	// Is equivalent to adding one more element to the orlist
	if rhs_andlist, ok := rhs.(*AndListExpr); ok {
		rhs_andlist.exprs = append(rhs_andlist.exprs, lhs)
		return rhs_andlist, true
	}

	return e, has_change
}

func (e *AndExpr) pre_compile(temp_counter int) ([]Operation, int) {	
	var ops_left []Operation
	var ops_right []Operation
	ops_left, temp_counter = e.lhs.pre_compile(temp_counter)
	ops_right, temp_counter = e.rhs.pre_compile(temp_counter)
	left_operand := ops_left[len(ops_left) - 1].result_var
	right_operand := ops_right[len(ops_right) - 1].result_var

	ops_left = append(ops_left, ops_right...)
	ops_left = append(ops_left, Operation{
		result_var: "*" + fmt.Sprint(temp_counter),
		op_type: OP_AND,
		operands: []string{left_operand, right_operand},
	})

	return ops_left, temp_counter + 1
}

// OR LIST EXPRESSION
type OrListExpr struct {
	exprs []Expression
}

func (e *OrListExpr) temp_evaluate() bool {
	for _, ee := range e.exprs {
		if ee.temp_evaluate() {
			return true
		}
	}
	return false
}

func (e *OrListExpr) print(depth int) string {
	var out strings.Builder
	out.WriteString(getPadding(depth) + "ORLIST\n")
	for idx, item := range e.exprs {
		out.WriteString(item.print(depth + 1))
		if idx != len(e.exprs)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

func (e *OrListExpr) is_equal(rhs Expression) bool {
	rhs_orlist, ok := rhs.(*OrListExpr)
	if !ok {
		return false
	}
	if len(rhs_orlist.exprs) != len(e.exprs) {
		return false
	}

	for i := range len(e.exprs) {
		if !e.exprs[i].is_equal(rhs_orlist.exprs[i]) {
			return false
		}
	}
	return true
}

func (e *OrListExpr) optimize_stage1() (Expression, bool) {
	has_change := false
	if len(e.exprs) == 1 {
		return e.exprs[0].optimize_stage1()
	}

	for i := range len(e.exprs) {
		c := false
		e.exprs[i], c = e.exprs[i].optimize_stage1()
		has_change = has_change || c
	}

	for i := range len(e.exprs) {
		current := e.exprs[i]

		// If an ORLIST contains 1 it should evaluate to true
		if current_var, ok := current.(*VarExpr); ok && current_var.name == "1" {
			return current_var, true
		}

		// If an ORLIST contains 0 it should be removed
		if current_var, ok := current.(*VarExpr); ok && current_var.name == "0" {
			e.exprs = remove_at_index(e.exprs, i)
			return e, true
		}

		for j := i + 1; j < len(e.exprs); j++ {
			other := e.exprs[j]
			// X OR X === X
			if current.is_equal(other) {
				e.exprs = remove_at_index(e.exprs, j)
				return e, true
			}

			// X OR NOT X === 1
			if other_not, ok := other.(*NotExpr); ok && other_not.expr.is_equal(current) {
				return &VarExpr{name: "1"}, true
			}

			// NOT X OR X === 1
			if current_not, ok := current.(*NotExpr); ok && current_not.expr.is_equal(other) {
				return &VarExpr{name: "1"}, true
			}
		}
	}

	return e, has_change
}

func (e *OrListExpr) optimize_stage2() (Expression, bool) {
	has_change := false

	for i := range len(e.exprs) {
		c := false
		e.exprs[i], c = e.exprs[i].optimize_stage2()
		has_change = has_change || c
	}

	for i := range len(e.exprs) {
		current := e.exprs[i]
		if current_paren, ok := current.(*ParenExpr); ok {
			current = current_paren.expr
		}

		// Nested OR in ORLIST
		if current_or, ok := current.(*OrExpr); ok {
			e.exprs = remove_at_index(e.exprs, i)
			e.exprs = append(e.exprs, current_or.lhs)
			e.exprs = append(e.exprs, current_or.rhs)
			return e, true
		}
		// Nested ORLIST in ORLIST
		if current_orlist, ok := current.(*OrListExpr); ok {
			e.exprs = remove_at_index(e.exprs, i)
			e.exprs = append(e.exprs, current_orlist.exprs...)
			return e, true
		}
	}

	return e, has_change
}

func (e *OrListExpr) pre_compile(temp_counter int) ([]Operation, int) {	
	var ops []Operation
	var operands []string

	for _, e := range e.exprs {
		var expr_ops []Operation
		expr_ops, temp_counter = e.pre_compile(temp_counter)
		ops = append(ops, expr_ops...)
		operands = append(operands, expr_ops[len(expr_ops) - 1].result_var)
	}

	ops = append(ops, Operation{
		result_var: "*" + fmt.Sprint(temp_counter),
		op_type: OP_OR,
		operands: operands,
	})

	return ops, temp_counter + 1
}

// AND LIST EXPORESSION
type AndListExpr struct {
	exprs []Expression
}

func (e *AndListExpr) temp_evaluate() bool {
	for _, ee := range e.exprs {
		if !ee.temp_evaluate() {
			return false
		}
	}
	return true
}

func (e *AndListExpr) print(depth int) string {
	var out strings.Builder
	out.WriteString(getPadding(depth) + "ANDLIST\n")
	for idx, item := range e.exprs {
		out.WriteString(item.print(depth + 1))
		if idx != len(e.exprs)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

func (e *AndListExpr) is_equal(rhs Expression) bool {
	rhs_andlist, ok := rhs.(*AndListExpr)
	if !ok {
		return false
	}
	if len(rhs_andlist.exprs) != len(e.exprs) {
		return false
	}

	for i := range len(e.exprs) {
		if !e.exprs[i].is_equal(rhs_andlist.exprs[i]) {
			return false
		}
	}
	return true
}

func (e *AndListExpr) optimize_stage1() (Expression, bool) {
	if len(e.exprs) == 1 {
		return e.exprs[0].optimize_stage1()
	}
	has_change := false
	for i := range len(e.exprs) {
		c := false
		e.exprs[i], c = e.exprs[i].optimize_stage1()
		has_change = has_change || c
	}

	for i := range len(e.exprs) {
		current := e.exprs[i]

		// If an ANDLIST contains 0 it should evaluate to false
		if current_var, ok := current.(*VarExpr); ok && current_var.name == "0" {
			return current_var, true
		}

		// If an ANDLIST contains 1 it should be removed
		if current_var, ok := current.(*VarExpr); ok && current_var.name == "1" {
			e.exprs = remove_at_index(e.exprs, i)
			return e, true
		}

		for j := i + 1; j < len(e.exprs); j++ {
			other := e.exprs[j]
			// X AND X === X
			if current.is_equal(e.exprs[j]) {
				e.exprs = remove_at_index(e.exprs, j)
				return e, true
			}

			// X AND NOT X === 0
			if other_not, ok := other.(*NotExpr); ok && other_not.expr.is_equal(current) {
				return &VarExpr{name: "0"}, true
			}

			// NOT X AND X === 0
			if current_not, ok := current.(*NotExpr); ok && current_not.expr.is_equal(other) {
				return &VarExpr{name: "0"}, true
			}
		}
	}

	return e, has_change
}

func (e *AndListExpr) optimize_stage2() (Expression, bool) {
	has_change := false
	for i := range len(e.exprs) {
		c := false
		e.exprs[i], c = e.exprs[i].optimize_stage2()
		has_change = has_change || c
	}

	for i := range len(e.exprs) {
		current := e.exprs[i]
		if current_paren, ok := current.(*ParenExpr); ok {
			current = current_paren.expr
		}

		// Nested AND in ANDLIST
		if current_and, ok := current.(*AndExpr); ok {
			e.exprs = remove_at_index(e.exprs, i)
			e.exprs = append(e.exprs, current_and.lhs)
			e.exprs = append(e.exprs, current_and.rhs)
			return e, true
		}
		// Nested ANDLIST in ANDLIST
		if current_andlist, ok := current.(*AndListExpr); ok {
			e.exprs = remove_at_index(e.exprs, i)
			e.exprs = append(e.exprs, current_andlist.exprs...)
			return e, true
		}
	}

	return e, has_change
}

func (e *AndListExpr) pre_compile(temp_counter int) ([]Operation, int) {	
	var ops []Operation
	var operands []string

	for _, e := range e.exprs {
		var expr_ops []Operation
		expr_ops, temp_counter = e.pre_compile(temp_counter)
		ops = append(ops, expr_ops...)
		operands = append(operands, expr_ops[len(expr_ops) - 1].result_var)
	}

	ops = append(ops, Operation{
		result_var: "*" + fmt.Sprint(temp_counter),
		op_type: OP_AND,
		operands: operands,
	})

	return ops, temp_counter + 1
}

// PAREN EXPRESSION
type ParenExpr struct {
	expr Expression
}

func (e *ParenExpr) temp_evaluate() bool {
	return e.expr.temp_evaluate()
}

func (e *ParenExpr) print(depth int) string {
	return getPadding(depth) + "PAREN\n" + e.expr.print(depth+1)
}

func (e *ParenExpr) is_equal(rhs Expression) bool {
	rhs_paren, ok := rhs.(*ParenExpr)
	if !ok {
		return false
	}
	return e.expr.is_equal(rhs_paren.expr)
}

func (e *ParenExpr) optimize_stage1() (Expression, bool) {
	var has_change bool
	e.expr, has_change = e.expr.optimize_stage1()

	// (VAR) === VAR
	if inner_var, ok := e.expr.(*VarExpr); ok {
		return inner_var, true
	}

	// (NOT) === NOT
	if inner_not, ok := e.expr.(*NotExpr); ok {
		return inner_not, true
	}

	// ((X)) === (X)
	if inner_paren, ok := e.expr.(*ParenExpr); ok {
		return inner_paren, true
	}

	return e, has_change
}

func (e *ParenExpr) optimize_stage2() (Expression, bool) {
	var has_change bool
	e.expr, has_change = e.expr.optimize_stage2()

	return e, has_change
}

func (e *ParenExpr) pre_compile(temp_counter int) ([]Operation, int) {
	return e.expr.pre_compile(temp_counter)
}

// VAR EXPRESSION
type VarExpr struct {
	name string
}

func (e *VarExpr) temp_evaluate() bool {
	return temp_env[e.name]
}

func (e *VarExpr) print(depth int) string {
	return getPadding(depth) + "VAR(" + e.name + ")"
}

func (e *VarExpr) is_equal(rhs Expression) bool {
	rhs_var, ok := rhs.(*VarExpr)
	if !ok {
		return false
	}

	return e.name == rhs_var.name
}

func (e *VarExpr) optimize_stage1() (Expression, bool) {
	return e, false
}

func (e *VarExpr) optimize_stage2() (Expression, bool) {
	return e, false
}

func (e *VarExpr) pre_compile(temp_counter int) ([]Operation, int) {
	return []Operation{
		Operation{
			result_var: e.name,
			op_type: OP_NOP,
			operands: []string{},
		},
	}, temp_counter
}
