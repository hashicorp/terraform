// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"
)

// ObjectToString is a helper function that converts a go-cty object to a string representation
func ObjectToString(obj cty.Value) string {
	if obj.IsNull() {
		return "<null>"
	}

	if !obj.IsWhollyKnown() {
		return "<unknown>"
	}

	if obj.Type().IsObjectType() && len(obj.Type().AttributeTypes()) == 0 {
		return "<empty>"
	}

	if obj.Type().IsObjectType() {
		result := ""
		it := obj.ElementIterator()
		for it.Next() {
			key, val := it.Element()
			keyStr := key.AsString()

			if result != "" {
				result += ","
			}

			if val.IsNull() {
				result += fmt.Sprintf("%s=<null>", keyStr)
				continue
			}

			result += fmt.Sprintf("%s=%s", keyStr, ValueToString(val))
		}

		return result
	}

	panic("not an object")
}

func ValueToString(val cty.Value) string {
	if val.IsNull() {
		return "<null>"
	}

	switch val.Type() {
	case cty.Bool:
		return fmt.Sprintf("%t", val.True())
	case cty.Number:
		return val.AsBigFloat().String()
	case cty.String:
		return val.AsString()
	case cty.List(cty.Bool):
		elements := val.AsValueSlice()
		parts := make([]string, len(elements))
		for i, element := range elements {
			parts[i] = fmt.Sprintf("%t", element.True())
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ","))
	case cty.List(cty.Number):
		elements := val.AsValueSlice()
		parts := make([]string, len(elements))
		for i, element := range elements {
			parts[i] = element.AsBigFloat().String()
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ","))
	case cty.List(cty.String):
		elements := val.AsValueSlice()
		parts := make([]string, len(elements))
		for i, element := range elements {
			parts[i] = element.AsString()
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ","))
	}

	return "<unknown type>"
}
