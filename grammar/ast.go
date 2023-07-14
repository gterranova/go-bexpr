// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package grammar

import (
	"fmt"
	"io"
	"strings"
)

// TODO - Probably should make most of what is in here un-exported

type Expression interface {
	ExpressionDump(w io.Writer, indent string, level int)
}

type UnaryOperator int

const (
	UnaryOpNot UnaryOperator = iota
)

func (op UnaryOperator) String() string {
	switch op {
	case UnaryOpNot:
		return "Not"
	default:
		return "UNKNOWN"
	}
}

type BinaryOperator int

const (
	BinaryOpAnd BinaryOperator = iota
	BinaryOpOr
)

func (op BinaryOperator) String() string {
	switch op {
	case BinaryOpAnd:
		return "And"
	case BinaryOpOr:
		return "Or"
	default:
		return "UNKNOWN"
	}
}

type MathOperator int

const (
	MathOpValue MathOperator = iota
	MathOpPlus
	MathOpMinus
	MathOpMul
	MathOpDiv
)

func (op MathOperator) String() string {
	switch op {
	case MathOpValue:
		return "="
	case MathOpPlus:
		return "+"
	case MathOpMinus:
		return "-"
	case MathOpMul:
		return "*"
	case MathOpDiv:
		return "/"
	default:
		return "UNKNOWN"
	}
}

type MatchOperator int

const (
	MatchEqual MatchOperator = iota
	MatchNotEqual
	MatchIn
	MatchNotIn
	MatchIsEmpty
	MatchIsNotEmpty
	MatchMatches
	MatchNotMatches
	MatchLower
	MatchLowerOrEqual
	MatchHigher
	MatchHigherOrEqual
)

func (op MatchOperator) String() string {
	switch op {
	case MatchEqual:
		return "Equal"
	case MatchNotEqual:
		return "Not Equal"
	case MatchIn:
		return "In"
	case MatchNotIn:
		return "Not In"
	case MatchIsEmpty:
		return "Is Empty"
	case MatchIsNotEmpty:
		return "Is Not Empty"
	case MatchMatches:
		return "Matches"
	case MatchNotMatches:
		return "Not Matches"
	case MatchLower:
		return "Lower"
	case MatchHigher:
		return "Higher"
	case MatchLowerOrEqual:
		return "Lower or Equal"
	case MatchHigherOrEqual:
		return "Higher or Equal"
	default:
		return "UNKNOWN"
	}
}

// NotPresentDisposition is called during evaluation when Selector fails to
// find a map key to determine the operator's behavior.
func (op MatchOperator) NotPresentDisposition() bool {
	// For a selector M["x"] against a map M that lacks an "x" key...
	switch op {
	case MatchEqual:
		// ...M["x"] == <anything> is false. Nothing is equal to a missing key
		return false
	case MatchNotEqual:
		// ...M["x"] != <anything> is true. Nothing is equal to a missing key
		return true
	case MatchIn:
		// "a" in M["x"] is false. Missing keys contain no values
		return false
	case MatchNotIn:
		// "a" not in M["x"] is true. Missing keys contain no values
		return true
	case MatchIsEmpty:
		// M["x"] is empty is true. Missing keys contain no values
		return true
	case MatchIsNotEmpty:
		// M["x"] is not empty is false. Missing keys contain no values
		return false
	case MatchMatches:
		// M["x"] matches <anything> is false. Nothing matches a missing key
		return false
	case MatchNotMatches:
		// M["x"] not matches <anything> is true. Nothing matches a missing key
		return true
	case MatchLower:
		// ...M["x"] < <anything> is false. Nothing is higher than a missing key
		return true
	case MatchHigher:
		// ...M["x"] > <anything> is false. Nothing is higher than a missing key
		return false
	case MatchLowerOrEqual:
		// ...M["x"] <= <anything> is false. Nothing is higher than a missing key
		return true
	case MatchHigherOrEqual:
		// ...M["x"] => <anything> is false. Nothing is higher than a missing key
		return false
	default:
		// Should never be reached as every operator should explicitly define its
		// behavior.
		return false
	}
}

type MatchValue struct {
	Selector  Selector
	Type      ValueType
	Raw       string
	Converted interface{}
}

type UnaryExpression struct {
	Operator UnaryOperator
	Operand  Expression
}

type BinaryExpression struct {
	Left     Expression
	Operator BinaryOperator
	Right    Expression
}

type ExpressionValue struct {
	Left     interface{} // *MatchValue or *EExpressionValue
	Operator MathOperator
	Right    interface{} // *MatchValue or *EExpressionValue
}

type SelectorType uint32

const (
	SelectorTypeUnknown = iota
	SelectorTypeBexpr
	SelectorTypeJsonPointer
)

type ValueType uint32

const (
	ValueTypeUndefined = iota
	ValueTypeBool
	ValueTypeInt
	ValueTypeUint
	ValueTypeFloat32
	ValueTypeFloat64
	ValueTypeString
	ValueTypeReflect
)

type Selector struct {
	Type SelectorType
	Path []string
}

func (sel Selector) String() string {
	if len(sel.Path) == 0 {
		return ""
	}
	switch sel.Type {
	case SelectorTypeBexpr:
		return strings.Join(sel.Path, ".")
	case SelectorTypeJsonPointer:
		return strings.Join(sel.Path, "/")
	default:
		return ""
	}
}

type MatchExpression struct {
	Operator MatchOperator
	Left     *ExpressionValue
	Right    *ExpressionValue
}

func (expr *UnaryExpression) ExpressionDump(w io.Writer, indent string, level int) {
	localIndent := strings.Repeat(indent, level)
	fmt.Fprintf(w, "%s%s {\n", localIndent, expr.Operator.String())
	expr.Operand.ExpressionDump(w, indent, level+1)
	fmt.Fprintf(w, "%s}\n", localIndent)
}

func (expr *BinaryExpression) ExpressionDump(w io.Writer, indent string, level int) {
	localIndent := strings.Repeat(indent, level)
	fmt.Fprintf(w, "%s%s {\n", localIndent, expr.Operator.String())
	expr.Left.ExpressionDump(w, indent, level+1)
	expr.Right.ExpressionDump(w, indent, level+1)
	fmt.Fprintf(w, "%s}\n", localIndent)
}

func (expr *ExpressionValue) ExpressionDump(w io.Writer, indent string, level int) {
	localIndent := strings.Repeat(indent, level)
	fmt.Fprintf(w, "%s%s %v %v\n", localIndent, expr.Left, expr.Operator.String(), expr.Right)
}

func (expr *MatchExpression) ExpressionDump(w io.Writer, indent string, level int) {
	switch expr.Operator {
	case MatchEqual, MatchNotEqual, MatchIn, MatchNotIn, MatchLower, MatchHigher, MatchLowerOrEqual, MatchHigherOrEqual:
		fmt.Fprintf(w, "%[1]s%[3]s {\n%[2]sSelector: %[4]v\n%[2]sValue: %[5]q\n%[1]s}\n", strings.Repeat(indent, level), strings.Repeat(indent, level+1), expr.Operator.String(), expr.Left.Left, expr.Right.Right)
	default:
		fmt.Fprintf(w, "%[1]s%[3]s {\n%[2]sSelector: %[4]v\n%[1]s}\n", strings.Repeat(indent, level), strings.Repeat(indent, level+1), expr.Operator.String(), expr.Left.Left)
	}
}
