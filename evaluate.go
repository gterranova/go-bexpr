// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package bexpr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/go-bexpr/grammar"
	"github.com/mitchellh/pointerstructure"
)

var byteSliceTyp reflect.Type = reflect.TypeOf([]byte{})

type UndefinedType struct{}

var undefined UndefinedType = UndefinedType{}

func isUndefined(f interface{}) bool {
	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Ptr {
		//return fmt.Errorf("not ptr; is %T", f)
		return false
	}
	v = v.Elem() // dereference the pointer
	if v.Kind() != reflect.Struct {
		return false
	}
	t := v.Type()
	return t == reflect.TypeOf(undefined)
}

func primitiveEqualityFn(value interface{}) func(first interface{}, second interface{}) bool {
	t := reflect.Indirect(reflect.ValueOf(value))
	switch t.Kind() {
	case reflect.Bool:
		return doEqualBool
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return doEqualInt64
	case reflect.Float32, reflect.Float64:
		return doEqualFloat64
	case reflect.String:
		return doEqualString
	default:
		return nil
	}
}

func doEqualBool(first interface{}, second interface{}) bool {
	b1, _ := CoerceBool(fmt.Sprintf("%v", first))
	b2, _ := CoerceBool(fmt.Sprintf("%v", second))
	return b1.(bool) == b2.(bool)
}

func doEqualInt64(first interface{}, second interface{}) bool {
	b1, _ := CoerceInt64(fmt.Sprintf("%v", first))
	b2, _ := CoerceInt64(fmt.Sprintf("%v", second))
	return b1.(int64) == b2.(int64)
}

func doEqualFloat64(first interface{}, second interface{}) bool {
	b1, _ := CoerceFloat64(fmt.Sprintf("%v", first))
	b2, _ := CoerceFloat64(fmt.Sprintf("%v", second))
	return b1.(float64) == b2.(float64)
}

func doEqualString(first interface{}, second interface{}) bool {
	b1 := fmt.Sprintf("%v", first)
	b2 := fmt.Sprintf("%v", second)
	return b1 == b2
}

func primitiveLowerFn(value interface{}) func(first interface{}, second interface{}) bool {
	/*
		switch value.(type) {
		case bool:
			return doLowerBool
		case uint, uint8, uint16, uint32, uint64, int, int8, int16, int32, int64:
			return doLowerInt64
		case float32, float64:
			return doLowerFloat64
		case string:
			return doLowerString
		default:
			return nil
		}*/
	t := reflect.Indirect(reflect.ValueOf(value))
	switch t.Kind() {
	case reflect.Bool:
		return doLowerBool
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return doLowerInt64
	case reflect.Float32, reflect.Float64:
		return doLowerFloat64
	case reflect.String:
		return doLowerString
	default:
		return nil
	}
}

func doLowerBool(first interface{}, second interface{}) bool {
	b1, _ := CoerceBool(fmt.Sprintf("%v", first))
	b2, _ := CoerceBool(fmt.Sprintf("%v", second))
	return b1.(bool) && !b2.(bool)
}

func doLowerInt64(first interface{}, second interface{}) bool {
	b1, _ := CoerceInt64(fmt.Sprintf("%v", first))
	b2, _ := CoerceInt64(fmt.Sprintf("%v", second))
	return b1.(int64) < b2.(int64)
}

func doLowerFloat64(first interface{}, second interface{}) bool {
	b1, _ := CoerceFloat64(fmt.Sprintf("%v", first))
	b2, _ := CoerceFloat64(fmt.Sprintf("%v", second))
	return b1.(float64) < b2.(float64)
}

func doLowerString(first interface{}, second interface{}) bool {
	b1 := fmt.Sprintf("%v", first)
	b2 := fmt.Sprintf("%v", second)
	return strings.HasPrefix(b2, b1)
}

// Get rid of 0 to many levels of pointers to get at the real type
func derefType(rtype reflect.Type) reflect.Type {
	for rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}
	return rtype
}

func doMatchMatches(leftValue interface{}, rightValue interface{}) (bool, error) {
	value := reflect.Indirect(reflect.ValueOf(leftValue))

	if !value.Type().ConvertibleTo(byteSliceTyp) {
		return false, fmt.Errorf("Value of type %s is not convertible to []byte", value.Type())
	}

	var re *regexp.Regexp
	//var ok bool
	//if expression.Right.Left.Converted != nil {
	//	re, ok = expression.Right.Left.Converted.(*regexp.Regexp)
	//}
	//if !ok || re == nil {
	var err error
	re, err = regexp.Compile(rightValue.(string))
	if err != nil {
		return false, fmt.Errorf("Failed to compile regular expression %q: %v", rightValue.(string), err)
	}
	//	expression.Right.Left.Converted = re
	//}

	return re.Match(value.Convert(byteSliceTyp).Interface().([]byte)), nil
}

func doMatchLower(leftValue interface{}, rightValue interface{}) (bool, error) {
	// NOTE: see preconditions in evaluategrammar.MatchExpressionRecurse
	eqFn := primitiveLowerFn(leftValue)
	if eqFn == nil {
		return false, fmt.Errorf("unable to find suitable primitive comparison function for matching %T and %T", leftValue, rightValue)
	}
	return eqFn(leftValue, rightValue), nil
}

func doMatchEqual(leftValue interface{}, rightValue interface{}) (bool, error) {
	eqFn := primitiveEqualityFn(leftValue)
	if eqFn == nil {
		return false, fmt.Errorf("unable to find suitable primitive comparison function for matching %T and %T", leftValue, rightValue)
	}
	return eqFn(leftValue, rightValue), nil
}

func doMatchIn(leftValue interface{}, rightValue interface{}) (bool, error) {
	value := reflect.ValueOf(leftValue)
	switch kind := value.Kind(); kind {
	case reflect.Map:
		found := value.MapIndex(reflect.ValueOf(rightValue))
		return found.IsValid(), nil

	case reflect.Slice, reflect.Array:
		itemType := derefType(value.Type().Elem())
		kind := itemType.Kind()
		switch kind {
		case reflect.Interface:
			// If it's an interface, that is, the type was []interface{}, we
			// have to treat each element individually, checking each element's
			// type/kind and rederiving the match value.
			for i := 0; i < value.Len(); i++ {
				item := value.Index(i).Elem()
				itemType := derefType(item.Type())
				kind := itemType.Kind()
				// We need to special case errors here. The reason is that in an
				// interface slice there can be a mix/match of types, but the
				// coerce functions expect a certain type. So the expression
				// passed in might be `"true" in "/my/slice"` but the value it's
				// checking against might be an integer, thus it will try to
				// coerce "true" to an integer and fail. However, all of the
				// functions use strconv which has a specific error type for
				// syntax errors, so as a special case in this situation, don't
				// error on a strconv.ErrSyntax, just continue on to the next
				// element.
				eqFn := primitiveEqualityFn(rightValue)
				if eqFn == nil {
					return false, fmt.Errorf(`unable to find suitable primitive comparison function for "in" comparison in interface slice: %s`, kind)
				}
				// the value will be the correct type as we verified the itemType
				if eqFn(item.Interface(), rightValue) {
					return true, nil
				}
			}
			return false, nil

		default:
			// Otherwise it's a concrete type and we can essentially cache the
			// answers. First we need to re-derive the match value for equality
			// assertion.
			eqFn := primitiveEqualityFn(rightValue)
			if eqFn == nil {
				return false, errors.New(`unable to find suitable primitive comparison function for "in" comparison`)
			}
			for i := 0; i < value.Len(); i++ {
				item := value.Index(i)
				// the value will be the correct type as we verified the itemType
				if eqFn(item.Interface(), rightValue) {
					return true, nil
				}
			}
			return false, nil
		}

	case reflect.String:
		return strings.Contains(value.String(), rightValue.(string)), nil

	default:
		return false, fmt.Errorf("Cannot perform in/contains operations on type %s", kind)
	}
}

func doMatchIsEmpty(leftValue interface{}) (bool, error) {
	value := reflect.Indirect(reflect.ValueOf(leftValue))

	return value.Len() == 0, nil
}

// evaluateNotPresent is called after a pointerstructure.ErrNotFound is
// encountered during evaluation.
//
// Returns true if the Selector Path's parent is a map as the missing key may
// be handled by the MatchOperator's NotPresentDisposition method.
//
// Returns false if the Selector Path has a length of 1, or if the parent of
// the Selector's Path is not a map, a pointerstructure.ErrrNotFound error is
// returned.
func evaluateNotPresent(ptr pointerstructure.Pointer, datum interface{}) bool {
	if len(ptr.Parts) < 2 {
		return false
	}

	// Pop the missing leaf part of the path
	ptr.Parts = ptr.Parts[0 : len(ptr.Parts)-1]

	val, _ := ptr.Get(datum)
	return reflect.ValueOf(val).Kind() == reflect.Map
}

func evaluateMatchExpression(expression *grammar.MatchExpression, datum interface{}, opt ...Option) (bool, error) {
	leftValue, err := getExprValue(expression.Left, datum, opt...)
	if err != nil {
		return false, err
	}

	if isUndefined(leftValue) {
		return expression.Operator.NotPresentDisposition(), nil
	}

	rightValue, err := getExprValue(expression.Right, datum, opt...)
	if err != nil {
		return false, err
	}

	//if isUndefined(rightValue) {
	//	return expression.Operator.NotPresentDisposition(), nil
	//}

	switch expression.Operator {
	case grammar.MatchLower:
		return doMatchLower(leftValue, rightValue)
	case grammar.MatchHigher:
		result, err := doMatchLower(leftValue, rightValue)
		if err == nil {
			if result {
				return !result, nil
			}
		}
		result, err = doMatchEqual(leftValue, rightValue)
		if err == nil {
			return !result, nil
		}
		return true, err
	case grammar.MatchLowerOrEqual:
		result, err := doMatchEqual(leftValue, rightValue)
		if err == nil {
			if result {
				return result, nil
			}
		}
		return doMatchLower(leftValue, rightValue)
	case grammar.MatchHigherOrEqual:
		result, err := doMatchLower(leftValue, rightValue)
		if err == nil {
			return !result, nil
		}
		return true, nil
	case grammar.MatchEqual:
		return doMatchEqual(leftValue, rightValue)
	case grammar.MatchNotEqual:
		result, err := doMatchEqual(leftValue, rightValue)
		if err == nil {
			return !result, nil
		}
		return false, err
	case grammar.MatchIn:
		return doMatchIn(leftValue, rightValue)
	case grammar.MatchNotIn:
		result, err := doMatchIn(leftValue, rightValue)
		if err == nil {
			return !result, nil
		}
		return false, err
	case grammar.MatchIsEmpty:
		return doMatchIsEmpty(leftValue)
	case grammar.MatchIsNotEmpty:
		result, err := doMatchIsEmpty(leftValue)
		if err == nil {
			return !result, nil
		}
		return false, err
	case grammar.MatchMatches:
		return doMatchMatches(leftValue, rightValue)
	case grammar.MatchNotMatches:
		result, err := doMatchMatches(leftValue, rightValue)
		if err == nil {
			return !result, nil
		}
		return false, err
	default:
		return false, fmt.Errorf("Invalid match operation: %d", expression.Operator)
	}
}

func evaluateExpressionValue(expression *grammar.ExpressionValue, datum interface{}, opt ...Option) (bool, error) {
	buf := new(bytes.Buffer)
	expression.ExpressionDump(buf, "    ", 0)
	fmt.Println(buf.String())
	return false, fmt.Errorf("Invalid match operation: %d", expression.Operator)
}

func getValue(expressionValue *grammar.MatchValue, datum interface{}, opt ...Option) (val interface{}, err error) {
	switch expressionValue.Type {
	case grammar.ValueTypeUndefined:
		val, err = &undefined, nil
	case grammar.ValueTypeBool:
		val, err = CoerceBool(expressionValue.Raw)

	case grammar.ValueTypeInt:
		val, err = CoerceInt64(expressionValue.Raw)

	case grammar.ValueTypeFloat64:
		val, err = CoerceFloat64(expressionValue.Raw)

	case grammar.ValueTypeReflect:
		opts := getOpts(opt...)
		ptr := pointerstructure.Pointer{
			Parts: expressionValue.Selector.Path,
			Config: pointerstructure.Config{
				TagName:                 opts.withTagName,
				ValueTransformationHook: opts.withHookFn,
			},
		}
		val, err = ptr.Get(datum)
		if err != nil {
			if errors.Is(err, pointerstructure.ErrNotFound) {
				// Prefer the withUnknown option if set, otherwise defer to NotPresent
				// disposition
				switch {
				case opts.withUnknown != nil:
					err = nil
					val = *opts.withUnknown
				case evaluateNotPresent(ptr, datum):
					return &undefined, nil
				}
			}

			if err != nil {
				return &undefined, fmt.Errorf("error finding value in datum: %w", err)
				//return false, fmt.Errorf("error finding value in datum: %w", err)
			}
		}

		if jn, ok := val.(json.Number); ok {
			if jni, err := jn.Int64(); err == nil {
				val = jni
			} else if jnf, err := jn.Float64(); err == nil {
				val = jnf
			} else {
				return nil, fmt.Errorf("unable to convert json number %s to int or float", jn)
			}
		}
	default:
		val, err = expressionValue.Raw, nil
	}
	return
}

func getExprValue(expression *grammar.ExpressionValue, datum interface{}, opt ...Option) (val interface{}, err error) {
	var lvalue, rvalue, opvalue interface{}

	if expression == nil {
		return nil, nil
	}

	lvalue, err = evaluate(expression.Left, datum, opt...)
	if err != nil {
		return lvalue, err
	}
	if expression.Right != nil {
		rvalue, err = evaluate(expression.Right, datum, opt...)
		if err != nil {
			return rvalue, err
		}
	}

	switch expression.Operator {
	case grammar.MathOpValue:
		return lvalue, err
	case grammar.MathOpPlus:
		switch rvalue.(type) {
		case bool:
			opvalue = lvalue.(bool) && rvalue.(bool)
		case int, int64:
			opvalue = lvalue.(int64) + rvalue.(int64)
		case float64:
			opvalue = lvalue.(float64) + rvalue.(float64)
		case string:
			opvalue = lvalue.(string) + rvalue.(string)
		default:
			return nil, fmt.Errorf("unknown types %T for math op", rvalue)
		}
	case grammar.MathOpMinus:
		switch rvalue.(type) {
		case int, int64:
			opvalue = lvalue.(int64) - rvalue.(int64)
		case float64:
			opvalue = lvalue.(float64) - rvalue.(float64)
		default:
			return nil, fmt.Errorf("unknown types %T for math op", rvalue)
		}
	case grammar.MathOpMul:
		switch rvalue.(type) {
		case int, int64:
			opvalue = lvalue.(int64) * rvalue.(int64)
		case float64:
			opvalue = lvalue.(float64) * rvalue.(float64)
		default:
			return nil, fmt.Errorf("unknown types %T for math op", rvalue)
		}
	case grammar.MathOpDiv:
		switch rvalue.(type) {
		case int, int64:
			opvalue = lvalue.(int64) / rvalue.(int64)
		case float64:
			opvalue = lvalue.(float64) / rvalue.(float64)
		default:
			return nil, fmt.Errorf("unknown types %T for math op", rvalue)
		}
	}
	return opvalue, nil
}

func evaluate(ast interface{}, datum interface{}, opt ...Option) (interface{}, error) {
	switch node := ast.(type) {
	case *grammar.UnaryExpression:
		switch node.Operator {
		case grammar.UnaryOpNot:
			result, err := evaluate(node.Operand, datum, opt...)
			return !result.(bool), err
		}
	case *grammar.BinaryExpression:
		switch node.Operator {
		case grammar.BinaryOpAnd:
			result, err := evaluate(node.Left, datum, opt...)
			if err != nil || !result.(bool) {
				return result, err
			}

			return evaluate(node.Right, datum, opt...)

		case grammar.BinaryOpOr:
			result, err := evaluate(node.Left, datum, opt...)
			if err != nil || result.(bool) {
				return result, err
			}

			return evaluate(node.Right, datum, opt...)
		}
	case *grammar.MatchExpression:
		return evaluateMatchExpression(node, datum, opt...)
	case *grammar.ExpressionValue:
		return getExprValue(node, datum, opt...)
	case *grammar.MatchValue:
		return getValue(node, datum, opt...)

	}
	return false, fmt.Errorf("Invalid AST node")
}
