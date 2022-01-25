package typeexpr

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// Type attempts to process the given expression as a type expression and, if
// successful, returns the resulting type. If unsuccessful, error diagnostics
// are returned.
func Type(expr hcl.Expression) (cty.Type, hcl.Diagnostics) {
	return getType(expr, false)
}

// TypeConstraint attempts to parse the given expression as a type constraint
// and, if successful, returns the resulting type. If unsuccessful, error
// diagnostics are returned.
//
// A type constraint has the same structure as a type, but it additionally
// allows the keyword "any" to represent cty.DynamicPseudoType, which is often
// used as a wildcard in type checking and type conversion operations.
func TypeConstraint(expr hcl.Expression) (cty.Type, hcl.Diagnostics) {
	return getType(expr, true)
}

// TypeString returns a string rendering of the given type as it would be
// expected to appear in the HCL native syntax.
//
// This is primarily intended for showing types to the user in an application
// that uses typexpr, where the user can be assumed to be familiar with the
// type expression syntax. In applications that do not use typeexpr these
// results may be confusing to the user and so type.FriendlyName may be
// preferable, even though it's less precise.
//
// TypeString produces reasonable results only for types like what would be
// produced by the Type and TypeConstraint functions. In particular, it cannot
// support capsule types.
func TypeString(ty cty.Type) string {
	// Easy cases first
	switch ty {
	case cty.String:
		return "string"
	case cty.Bool:
		return "bool"
	case cty.Number:
		return "number"
	case cty.DynamicPseudoType:
		return "any"
	}

	if ty.IsCapsuleType() {
		panic("TypeString does not support capsule types")
	}

	if ty.IsCollectionType() {
		ety := ty.ElementType()
		etyString := TypeString(ety)
		switch {
		case ty.IsListType():
			return fmt.Sprintf("list(%s)", etyString)
		case ty.IsSetType():
			return fmt.Sprintf("set(%s)", etyString)
		case ty.IsMapType():
			return fmt.Sprintf("map(%s)", etyString)
		default:
			// Should never happen because the above is exhaustive
			panic("unsupported collection type")
		}
	}

	if ty.IsObjectType() {
		var buf bytes.Buffer
		buf.WriteString("object({")
		atys := ty.AttributeTypes()
		names := make([]string, 0, len(atys))
		for name := range atys {
			names = append(names, name)
		}
		sort.Strings(names)
		first := true
		for _, name := range names {
			aty := atys[name]
			if !first {
				buf.WriteByte(',')
			}
			if !hclsyntax.ValidIdentifier(name) {
				// Should never happen for any type produced by this package,
				// but we'll do something reasonable here just so we don't
				// produce garbage if someone gives us a hand-assembled object
				// type that has weird attribute names.
				// Using Go-style quoting here isn't perfect, since it doesn't
				// exactly match HCL syntax, but it's fine for an edge-case.
				buf.WriteString(fmt.Sprintf("%q", name))
			} else {
				buf.WriteString(name)
			}
			buf.WriteByte('=')
			buf.WriteString(TypeString(aty))
			first = false
		}
		buf.WriteString("})")
		return buf.String()
	}

	if ty.IsTupleType() {
		var buf bytes.Buffer
		buf.WriteString("tuple([")
		etys := ty.TupleElementTypes()
		first := true
		for _, ety := range etys {
			if !first {
				buf.WriteByte(',')
			}
			buf.WriteString(TypeString(ety))
			first = false
		}
		buf.WriteString("])")
		return buf.String()
	}

	// Should never happen because we covered all cases above.
	panic(fmt.Errorf("unsupported type %#v", ty))
}
