package convert

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
)

// MismatchMessage is a helper to return an English-language description of
// the differences between got and want, phrased as a reason why got does
// not conform to want.
//
// This function does not itself attempt conversion, and so it should generally
// be used only after a conversion has failed, to report the conversion failure
// to an English-speaking user. The result will be confusing got is actually
// conforming to or convertable to want.
//
// The shorthand helper function Convert uses this function internally to
// produce its error messages, so callers of that function do not need to
// also use MismatchMessage.
//
// This function is similar to Type.TestConformance, but it is tailored to
// describing conversion failures and so the messages it generates relate
// specifically to the conversion rules implemented in this package.
func MismatchMessage(got, want cty.Type) string {
	switch {

	case got.IsObjectType() && want.IsObjectType():
		// If both types are object types then we may be able to say something
		// about their respective attributes.
		return mismatchMessageObjects(got, want)

	default:
		// If we have nothing better to say, we'll just state what was required.
		return want.FriendlyName() + " required"
	}
}

func mismatchMessageObjects(got, want cty.Type) string {
	// Per our conversion rules, "got" is allowed to be a superset of "want",
	// and so we'll produce error messages here under that assumption.
	gotAtys := got.AttributeTypes()
	wantAtys := want.AttributeTypes()

	// If we find missing attributes then we'll report those in preference,
	// but if not then we will report a maximum of one non-conforming
	// attribute, just to keep our messages relatively terse.
	// We'll also prefer to report a recursive type error from an _unsafe_
	// conversion over a safe one, because these are subjectively more
	// "serious".
	var missingAttrs []string
	var unsafeMismatchAttr string
	var safeMismatchAttr string

	for name, wantAty := range wantAtys {
		gotAty, exists := gotAtys[name]
		if !exists {
			missingAttrs = append(missingAttrs, name)
			continue
		}

		// We'll now try to convert these attributes in isolation and
		// see if we have a nested conversion error to report.
		// We'll try an unsafe conversion first, and then fall back on
		// safe if unsafe is possible.

		// If we already have an unsafe mismatch attr error then we won't bother
		// hunting for another one.
		if unsafeMismatchAttr != "" {
			continue
		}
		if conv := GetConversionUnsafe(gotAty, wantAty); conv == nil {
			unsafeMismatchAttr = fmt.Sprintf("attribute %q: %s", name, MismatchMessage(gotAty, wantAty))
		}

		// If we already have a safe mismatch attr error then we won't bother
		// hunting for another one.
		if safeMismatchAttr != "" {
			continue
		}
		if conv := GetConversion(gotAty, wantAty); conv == nil {
			safeMismatchAttr = fmt.Sprintf("attribute %q: %s", name, MismatchMessage(gotAty, wantAty))
		}
	}

	// We should now have collected at least one problem. If we have more than
	// one then we'll use our preference order to decide what is most important
	// to report.
	switch {

	case len(missingAttrs) != 0:
		switch len(missingAttrs) {
		case 1:
			return fmt.Sprintf("attribute %q is required", missingAttrs[0])
		case 2:
			return fmt.Sprintf("attributes %q and %q are required", missingAttrs[0], missingAttrs[1])
		default:
			sort.Strings(missingAttrs)
			var buf bytes.Buffer
			for _, name := range missingAttrs[:len(missingAttrs)-1] {
				fmt.Fprintf(&buf, "%q, ", name)
			}
			fmt.Fprintf(&buf, "and %q", missingAttrs[len(missingAttrs)-1])
			return fmt.Sprintf("attributes %s are required", buf.Bytes())
		}

	case unsafeMismatchAttr != "":
		return unsafeMismatchAttr

	case safeMismatchAttr != "":
		return safeMismatchAttr

	default:
		// We should never get here, but if we do then we'll return
		// just a generic message.
		return "incorrect object attributes"
	}
}
