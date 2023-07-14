// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package grammar

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAST_Dump(t *testing.T) {
	t.Parallel()
	type testCase struct {
		expr     Expression
		expected string
	}

	tests := map[string]testCase{
		"MatchEqual": {
			expr:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchEqual, Right: &MatchValue{Raw: "baz"}},
			expected: "Equal {\n   Selector: foo.bar\n   Value: \"baz\"\n}\n",
		},
		"MatchNotEqual": {
			expr:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchNotEqual, Right: &MatchValue{Raw: "baz"}},
			expected: "Not Equal {\n   Selector: foo.bar\n   Value: \"baz\"\n}\n",
		},
		"MatchIn": {
			expr:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIn, Right: &MatchValue{Raw: "baz"}},
			expected: "In {\n   Selector: foo.bar\n   Value: \"baz\"\n}\n",
		},
		"MatchNotIn": {
			expr:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchNotIn, Right: &MatchValue{Raw: "baz"}},
			expected: "Not In {\n   Selector: foo.bar\n   Value: \"baz\"\n}\n",
		},
		"MatchIsEmpty": {
			expr:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil},
			expected: "Is Empty {\n   Selector: foo.bar\n}\n",
		},
		"MatchIsNotEmpty": {
			expr:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsNotEmpty, Right: nil},
			expected: "Is Not Empty {\n   Selector: foo.bar\n}\n",
		},
		"MatchUnknown": {
			expr:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchOperator(42), Right: nil},
			expected: "UNKNOWN {\n   Selector: foo.bar\n}\n",
		},
		"UnaryOpNot": {
			expr:     &UnaryExpression{Operator: UnaryOpNot, Operand: &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil}},
			expected: "Not {\n   Is Empty {\n      Selector: foo.bar\n   }\n}\n",
		},
		"UnaryOpUnknown": {
			expr:     &UnaryExpression{Operator: UnaryOperator(42), Operand: &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil}},
			expected: "UNKNOWN {\n   Is Empty {\n      Selector: foo.bar\n   }\n}\n",
		},
		"BinaryOpAnd": {
			expr: &BinaryExpression{
				Operator: BinaryOpAnd,
				Left:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil},
				Right:    &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil},
			},
			expected: "And {\n   Is Empty {\n      Selector: foo.bar\n   }\n   Is Empty {\n      Selector: foo.bar\n   }\n}\n",
		},
		"BinaryOpOr": {
			expr: &BinaryExpression{
				Operator: BinaryOpOr,
				Left:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil},
				Right:    &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil},
			},
			expected: "Or {\n   Is Empty {\n      Selector: foo.bar\n   }\n   Is Empty {\n      Selector: foo.bar\n   }\n}\n",
		},
		"BinaryOpUnknown": {
			expr: &BinaryExpression{
				Operator: BinaryOperator(42),
				Left:     &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil},
				Right:    &MatchExpression{Left: &MatchValue{Selector: Selector{Type: SelectorTypeBexpr, Path: []string{"foo", "bar"}}}, Operator: MatchIsEmpty, Right: nil},
			},
			expected: "UNKNOWN {\n   Is Empty {\n      Selector: foo.bar\n   }\n   Is Empty {\n      Selector: foo.bar\n   }\n}\n",
		},
	}

	for name, tcase := range tests {
		tcase := tcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			buf := new(bytes.Buffer)
			tcase.expr.ExpressionDump(buf, "   ", 0)
			actual := buf.String()

			require.Equal(t, tcase.expected, actual)
		})
	}
}
