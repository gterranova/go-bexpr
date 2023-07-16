// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package bexpr

import (
	"fmt"
	"strconv"
)

// CoerceInt64 conforms to the FieldValueCoercionFn signature
// and can be used to convert the raw string value of
// an expression into an `int64`
func CoerceInt64(value interface{}) (int64, error) {
	s := fmt.Sprintf("%v", value)
	i, err := strconv.ParseInt(s, 0, 64)
	return int64(i), err
}

// CoerceUint64 conforms to the FieldValueCoercionFn signature
// and can be used to convert the raw string value of
// an expression into an `int64`
func CoerceUint64(value string) (interface{}, error) {
	i, err := strconv.ParseUint(value, 0, 64)
	return uint64(i), err
}

// CoerceBool conforms to the FieldValueCoercionFn signature
// and can be used to convert the raw string value of
// an expression into a `bool`
func CoerceBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case int, int64:
		return v != 0, nil
	case float64:
		return v != 0, nil
	case string:
		b, err := strconv.ParseBool(fmt.Sprintf("%v", value))
		if err != nil {
			return len(v) > 0, nil
		}
		return b, nil
	}
	return strconv.ParseBool(fmt.Sprintf("%v", value))
}

// CoerceFloat32 conforms to the FieldValueCoercionFn signature
// and can be used to convert the raw string value of
// an expression into an `float32`
func CoerceFloat32(value string) (interface{}, error) {
	// ParseFloat always returns a float64 but ensures
	// it can be converted to a float32 without changing
	// its value
	f, err := strconv.ParseFloat(value, 32)
	return float32(f), err
}

// CoerceFloat64 conforms to the FieldValueCoercionFn signature
// and can be used to convert the raw string value of
// an expression into an `float64`
func CoerceFloat64(value interface{}) (float64, error) {
	s := fmt.Sprintf("%v", value)
	return strconv.ParseFloat(s, 64)
}
