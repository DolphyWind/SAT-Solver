package main

import (
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
