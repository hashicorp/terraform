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

			switch val.Type() {
			case cty.Bool:
				result += fmt.Sprintf("%s=%t", keyStr, val.True())
			case cty.Number:
				result += fmt.Sprintf("%s=%s", keyStr, val.AsBigFloat().String())
			case cty.String:
				result += fmt.Sprintf("%s=%s", keyStr, val.AsString())
			case cty.List(cty.Bool):
				elements := val.AsValueSlice()
				parts := make([]string, len(elements))
				for i, element := range elements {
					parts[i] = fmt.Sprintf("%t", element.True())
				}
				result += fmt.Sprintf("%s=[%s]", keyStr, strings.Join(parts, ","))
			case cty.List(cty.Number):
				elements := val.AsValueSlice()
				parts := make([]string, len(elements))
				for i, element := range elements {
					parts[i] = element.AsBigFloat().String()
				}
				result += fmt.Sprintf("%s=[%s]", keyStr, strings.Join(parts, ","))
			case cty.List(cty.String):
				elements := val.AsValueSlice()
				parts := make([]string, len(elements))
				for i, element := range elements {
					parts[i] = element.AsString()
				}
				result += fmt.Sprintf("%s=[%s]", keyStr, strings.Join(parts, ","))
			}
		}

		return result
	}

	panic("not an object")
}
