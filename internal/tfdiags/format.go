// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"bytes"
	"encoding/json"
	"fmt"

	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/lang/marks"
)

// CompactValueStr produces a compact, single-line summary of a given value
// that is suitable for display in the UI.
//
// For primitives it returns a full representation, while for more complex
// types it instead summarizes the type, size, etc to produce something
// that is hopefully still somewhat useful but not as verbose as a rendering
// of the entire data structure.
func CompactValueStr(val cty.Value) string {
	// This is a specialized subset of value rendering tailored to producing
	// helpful but concise messages in diagnostics. It is not comprehensive
	// nor intended to be used for other purposes.

	val, valMarks := val.Unmark()
	for mark := range valMarks {
		switch mark {
		case marks.Sensitive:
			// We check this in here just to make sure, but note that the caller
			// of compactValueStr ought to have already checked this and skipped
			// calling into compactValueStr anyway, so this shouldn't actually
			// be reachable.
			return "(sensitive value)"
		case marks.Ephemeral:
			// A non-sensitive ephemeral value is fine to show in the UI. Values
			// that are both ephemeral and sensitive should have both markings
			// and should therefore get caught by the marks.Sensitive case
			// above.
			return "(ephemeral value)"
		default:
			// We don't know about any other marks, so we'll be conservative.
			// This shouldn't actually reachable since the caller should've
			// checked this and skipped calling compactValueStr anyway.
			return "value with unrecognized marks (this is a bug in Terraform)"
		}
	}

	// WARNING: We've only checked that the value isn't sensitive _shallowly_
	// here, and so we must never show any element values from complex types
	// in here. However, it's fine to show map keys and attribute names because
	// those are never sensitive in isolation: the entire value would be
	// sensitive in that case.

	ty := val.Type()
	switch {
	case val.IsNull():
		return "null"
	case !val.IsKnown():
		// Should never happen here because we should filter before we get
		// in here, but we'll do something reasonable rather than panic.
		return "(not yet known)"
	case ty == cty.Bool:
		if val.True() {
			return "true"
		}
		return "false"
	case ty == cty.Number:
		bf := val.AsBigFloat()
		return bf.Text('g', 10)
	case ty == cty.String:
		// Go string syntax is not exactly the same as HCL native string syntax,
		// but we'll accept the minor edge-cases where this is different here
		// for now, just to get something reasonable here.
		return fmt.Sprintf("%q", val.AsString())
	case ty.IsCollectionType() || ty.IsTupleType():
		l := val.LengthInt()
		switch l {
		case 0:
			return "empty " + ty.FriendlyName()
		case 1:
			return ty.FriendlyName() + " with 1 element"
		default:
			return fmt.Sprintf("%s with %d elements", ty.FriendlyName(), l)
		}
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		l := len(atys)
		switch l {
		case 0:
			return "object with no attributes"
		case 1:
			var name string
			for k := range atys {
				name = k
			}
			return fmt.Sprintf("object with 1 attribute %q", name)
		default:
			return fmt.Sprintf("object with %d attributes", l)
		}
	default:
		return ty.FriendlyName()
	}
}

// TraversalStr produces a representation of an HCL traversal that is compact,
// resembles HCL native syntax, and is suitable for display in the UI.
func TraversalStr(traversal hcl.Traversal) string {
	// This is a specialized subset of traversal rendering tailored to
	// producing helpful contextual messages in diagnostics. It is not
	// comprehensive nor intended to be used for other purposes.

	var buf bytes.Buffer
	for _, step := range traversal {
		switch tStep := step.(type) {
		case hcl.TraverseRoot:
			buf.WriteString(tStep.Name)
		case hcl.TraverseAttr:
			buf.WriteByte('.')
			buf.WriteString(tStep.Name)
		case hcl.TraverseIndex:
			buf.WriteByte('[')
			if keyTy := tStep.Key.Type(); keyTy.IsPrimitiveType() {
				buf.WriteString(CompactValueStr(tStep.Key))
			} else {
				// We'll just use a placeholder for more complex values,
				// since otherwise our result could grow ridiculously long.
				buf.WriteString("...")
			}
			buf.WriteByte(']')
		}
	}
	return buf.String()
}

// FormatValueStr produces a JSON-compatible, human-readable representation of a
// cty.Value that is suitable for display in the UI.
//
// The full representation of the value is produced, but with some redaction to
// nodes within the value sensitive and ephemeral marks.
// e.g {"a": "10", "b": "password"} => {"a": "10", "b": "(sensitive value)"}
func FormatValueStr(val cty.Value) (string, error) {
	var buf bytes.Buffer

	val, err := cty.Transform(val, func(path cty.Path, val cty.Value) (cty.Value, error) {
		// If a value is sensitive or ephemeral or unknown, we redact it, otherwise
		// we return the value as is.
		if val.HasMark(marks.Sensitive) || val.HasMark(marks.Ephemeral) || !val.IsKnown() {
			return cty.StringVal(CompactValueStr(val)), nil
		}
		return val, nil
	})
	if err != nil {
		return "", fmt.Errorf("unexpected error transforming value: %s", err)
	}

	jsonVal, err := ctyjson.Marshal(val, val.Type())
	if err != nil {
		return "", fmt.Errorf("unexpected error marshalling value: %s", err)
	}

	// indent the JSON output for better readability
	if err := json.Indent(&buf, jsonVal, "", "  "); err != nil {
		return "", fmt.Errorf("unexpected error formatting JSON: %s", err)
	}

	return buf.String(), nil
}
