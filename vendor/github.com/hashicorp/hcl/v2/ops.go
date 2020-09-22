package hcl

import (
	"fmt"
	"math/big"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// Index is a helper function that performs the same operation as the index
// operator in the HCL expression language. That is, the result is the
// same as it would be for collection[key] in a configuration expression.
//
// This is exported so that applications can perform indexing in a manner
// consistent with how the language does it, including handling of null and
// unknown values, etc.
//
// Diagnostics are produced if the given combination of values is not valid.
// Therefore a pointer to a source range must be provided to use in diagnostics,
// though nil can be provided if the calling application is going to
// ignore the subject of the returned diagnostics anyway.
func Index(collection, key cty.Value, srcRange *Range) (cty.Value, Diagnostics) {
	if collection.IsNull() {
		return cty.DynamicVal, Diagnostics{
			{
				Severity: DiagError,
				Summary:  "Attempt to index null value",
				Detail:   "This value is null, so it does not have any indices.",
				Subject:  srcRange,
			},
		}
	}
	if key.IsNull() {
		return cty.DynamicVal, Diagnostics{
			{
				Severity: DiagError,
				Summary:  "Invalid index",
				Detail:   "Can't use a null value as an indexing key.",
				Subject:  srcRange,
			},
		}
	}
	ty := collection.Type()
	kty := key.Type()
	if kty == cty.DynamicPseudoType || ty == cty.DynamicPseudoType {
		return cty.DynamicVal, nil
	}

	switch {

	case ty.IsListType() || ty.IsTupleType() || ty.IsMapType():
		var wantType cty.Type
		switch {
		case ty.IsListType() || ty.IsTupleType():
			wantType = cty.Number
		case ty.IsMapType():
			wantType = cty.String
		default:
			// should never happen
			panic("don't know what key type we want")
		}

		key, keyErr := convert.Convert(key, wantType)
		if keyErr != nil {
			return cty.DynamicVal, Diagnostics{
				{
					Severity: DiagError,
					Summary:  "Invalid index",
					Detail: fmt.Sprintf(
						"The given key does not identify an element in this collection value: %s.",
						keyErr.Error(),
					),
					Subject: srcRange,
				},
			}
		}

		has := collection.HasIndex(key)
		if !has.IsKnown() {
			if ty.IsTupleType() {
				return cty.DynamicVal, nil
			} else {
				return cty.UnknownVal(ty.ElementType()), nil
			}
		}
		if has.False() {
			// We have a more specialized error message for the situation of
			// using a fractional number to index into a sequence, because
			// that will tend to happen if the user is trying to use division
			// to calculate an index and not realizing that HCL does float
			// division rather than integer division.
			if (ty.IsListType() || ty.IsTupleType()) && key.Type().Equals(cty.Number) {
				if key.IsKnown() && !key.IsNull() {
					bf := key.AsBigFloat()
					if _, acc := bf.Int(nil); acc != big.Exact {
						return cty.DynamicVal, Diagnostics{
							{
								Severity: DiagError,
								Summary:  "Invalid index",
								Detail:   fmt.Sprintf("The given key does not identify an element in this collection value: indexing a sequence requires a whole number, but the given index (%g) has a fractional part.", bf),
								Subject:  srcRange,
							},
						}
					}
				}
			}

			return cty.DynamicVal, Diagnostics{
				{
					Severity: DiagError,
					Summary:  "Invalid index",
					Detail:   "The given key does not identify an element in this collection value.",
					Subject:  srcRange,
				},
			}
		}

		return collection.Index(key), nil

	case ty.IsObjectType():
		key, keyErr := convert.Convert(key, cty.String)
		if keyErr != nil {
			return cty.DynamicVal, Diagnostics{
				{
					Severity: DiagError,
					Summary:  "Invalid index",
					Detail: fmt.Sprintf(
						"The given key does not identify an element in this collection value: %s.",
						keyErr.Error(),
					),
					Subject: srcRange,
				},
			}
		}
		if !collection.IsKnown() {
			return cty.DynamicVal, nil
		}
		if !key.IsKnown() {
			return cty.DynamicVal, nil
		}

		attrName := key.AsString()

		if !ty.HasAttribute(attrName) {
			return cty.DynamicVal, Diagnostics{
				{
					Severity: DiagError,
					Summary:  "Invalid index",
					Detail:   "The given key does not identify an element in this collection value.",
					Subject:  srcRange,
				},
			}
		}

		return collection.GetAttr(attrName), nil

	default:
		return cty.DynamicVal, Diagnostics{
			{
				Severity: DiagError,
				Summary:  "Invalid index",
				Detail:   "This value does not have any indices.",
				Subject:  srcRange,
			},
		}
	}

}

// GetAttr is a helper function that performs the same operation as the
// attribute access in the HCL expression language. That is, the result is the
// same as it would be for obj.attr in a configuration expression.
//
// This is exported so that applications can access attributes in a manner
// consistent with how the language does it, including handling of null and
// unknown values, etc.
//
// Diagnostics are produced if the given combination of values is not valid.
// Therefore a pointer to a source range must be provided to use in diagnostics,
// though nil can be provided if the calling application is going to
// ignore the subject of the returned diagnostics anyway.
func GetAttr(obj cty.Value, attrName string, srcRange *Range) (cty.Value, Diagnostics) {
	if obj.IsNull() {
		return cty.DynamicVal, Diagnostics{
			{
				Severity: DiagError,
				Summary:  "Attempt to get attribute from null value",
				Detail:   "This value is null, so it does not have any attributes.",
				Subject:  srcRange,
			},
		}
	}

	ty := obj.Type()
	switch {
	case ty.IsObjectType():
		if !ty.HasAttribute(attrName) {
			return cty.DynamicVal, Diagnostics{
				{
					Severity: DiagError,
					Summary:  "Unsupported attribute",
					Detail:   fmt.Sprintf("This object does not have an attribute named %q.", attrName),
					Subject:  srcRange,
				},
			}
		}

		if !obj.IsKnown() {
			return cty.UnknownVal(ty.AttributeType(attrName)), nil
		}

		return obj.GetAttr(attrName), nil
	case ty.IsMapType():
		if !obj.IsKnown() {
			return cty.UnknownVal(ty.ElementType()), nil
		}

		idx := cty.StringVal(attrName)
		if obj.HasIndex(idx).False() {
			return cty.DynamicVal, Diagnostics{
				{
					Severity: DiagError,
					Summary:  "Missing map element",
					Detail:   fmt.Sprintf("This map does not have an element with the key %q.", attrName),
					Subject:  srcRange,
				},
			}
		}

		return obj.Index(idx), nil
	case ty == cty.DynamicPseudoType:
		return cty.DynamicVal, nil
	default:
		return cty.DynamicVal, Diagnostics{
			{
				Severity: DiagError,
				Summary:  "Unsupported attribute",
				Detail:   "This value does not have any attributes.",
				Subject:  srcRange,
			},
		}
	}

}

// ApplyPath is a helper function that applies a cty.Path to a value using the
// indexing and attribute access operations from HCL.
//
// This is similar to calling the path's own Apply method, but ApplyPath uses
// the more relaxed typing rules that apply to these operations in HCL, rather
// than cty's relatively-strict rules. ApplyPath is implemented in terms of
// Index and GetAttr, and so it has the same behavior for individual steps
// but will stop and return any errors returned by intermediate steps.
//
// Diagnostics are produced if the given path cannot be applied to the given
// value. Therefore a pointer to a source range must be provided to use in
// diagnostics, though nil can be provided if the calling application is going
// to ignore the subject of the returned diagnostics anyway.
func ApplyPath(val cty.Value, path cty.Path, srcRange *Range) (cty.Value, Diagnostics) {
	var diags Diagnostics

	for _, step := range path {
		var stepDiags Diagnostics
		switch ts := step.(type) {
		case cty.IndexStep:
			val, stepDiags = Index(val, ts.Key, srcRange)
		case cty.GetAttrStep:
			val, stepDiags = GetAttr(val, ts.Name, srcRange)
		default:
			// Should never happen because the above are all of the step types.
			diags = diags.Append(&Diagnostic{
				Severity: DiagError,
				Summary:  "Invalid path step",
				Detail:   fmt.Sprintf("Go type %T is not a valid path step. This is a bug in this program.", step),
				Subject:  srcRange,
			})
			return cty.DynamicVal, diags
		}

		diags = append(diags, stepDiags...)
		if stepDiags.HasErrors() {
			return cty.DynamicVal, diags
		}
	}

	return val, diags
}
