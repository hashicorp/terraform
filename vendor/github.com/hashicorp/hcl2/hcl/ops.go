package hcl

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// Index is a helper function that performs the same operation as the index
// operator in the zcl expression language. That is, the result is the
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
