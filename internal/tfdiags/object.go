// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1
package tfdiags

import (
	"fmt"
	"slices"
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

	if !obj.Type().IsObjectType() {
		panic("not an object")
	}

	it := obj.ElementIterator()
	keys := make([]string, 0, obj.LengthInt())
	objMap := make(map[string]cty.Value)
	result := ""
	// store the keys for the object, and sort them
	// before appending to the result so that the final value is deterministic.
	for it.Next() {
		key, val := it.Element()
		keyStr := key.AsString()
		keys = append(keys, keyStr)
		objMap[keyStr] = val
	}

	slices.Sort(keys)
	for _, key := range keys {
		val := objMap[key]
		if result != "" {
			result += ","
		}

		if val.IsNull() {
			result += fmt.Sprintf("%s=<null>", key)
			continue
		}

		result += fmt.Sprintf("%s=%s", key, ValueToString(val))
	}

	return result
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
