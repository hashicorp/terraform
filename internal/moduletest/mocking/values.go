// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"fmt"
	"math/rand"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	// testRand and chars are used to generate random strings for the computed
	// values.
	//
	// If testRand is null, then the global random is used. This allows us to
	// seed tests for repeatable results.
	testRand *rand.Rand
	chars    = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
)

// PlanComputedValuesForResource accepts a target value, and populates it with
// cty.UnknownValues wherever a value should be computed during the apply stage.
//
// This method basically simulates the behaviour of a plan request in a real
// provider.
func PlanComputedValuesForResource(original cty.Value, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	return populateComputedValues(original, ReplacementValue{}, schema, isNull, makeUnknown)
}

// ApplyComputedValuesForResource accepts a target value, and populates it
// either with values from the provided with argument, or with generated values
// created semi-randomly. This will only target values that are computed and
// unknown.
//
// This method basically simulates the behaviour of an apply request in a real
// provider.
func ApplyComputedValuesForResource(original cty.Value, with ReplacementValue, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	return populateComputedValues(original, with, schema, isUnknown, with.makeKnown)
}

// ComputedValuesForDataSource accepts a target value, and populates it either
// with values from the provided with argument, or with generated values created
// semi-randomly. This will only target values that are computed and null.
//
// This function does what PlanComputedValuesForResource and
// ApplyComputedValuesForResource do but in a single step with no intermediary
// unknown stage.
//
// This method basically simulates the behaviour of a get data source request
// in a real provider.
func ComputedValuesForDataSource(original cty.Value, with ReplacementValue, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	return populateComputedValues(original, with, schema, isNull, with.makeKnown)
}

type processValue func(value cty.Value) bool

type populateValue func(value cty.Value, with cty.Value, path cty.Path) (cty.Value, tfdiags.Diagnostics)

func populateComputedValues(target cty.Value, with ReplacementValue, schema *configschema.Block, processValue processValue, populateValue populateValue) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if !with.validate() {
		// This is actually a user error, it means the user wrote something like
		// `values = "not an object"` when defining the replacement values for
		// this in the mock or test file. We should have caught this earlier in
		// the validation, but we want this function to be robust and not panic
		// so we'll check again just in case.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid replacement value",
			Detail:   fmt.Sprintf("The requested replacement value must be an object type, but was %s.", with.Value.Type().FriendlyName()),
			Subject:  with.Range.Ptr(),
		})
	}

	// We're going to search for any elements within the target value that meet
	// the joint criteria of being computed and whatever processValue is
	// checking.
	//
	// We'll then replace anything that meets the criteria with the output of
	// populateValue.
	//
	// This transform should be robust (in that it should never fail), it'll
	// populate the external diags variable with any values it should have
	// replaced but couldn't and just return the original value.
	value, err := cty.Transform(target, func(path cty.Path, target cty.Value) (cty.Value, error) {

		// Get the attribute for the current target.
		attribute := schema.AttributeByPath(path)

		if attribute == nil {
			// Then this is an intermediate path which does not represent an
			// attribute, and it cannot be computed.
			return target, nil
		}

		// Now, we check if we should be replacing this value with something.
		if attribute.Computed && processValue(target) {

			// Get the value we should be replacing target with.
			replacement, replacementDiags := with.getReplacementSafe(path)
			diags = diags.Append(replacementDiags)

			// Upstream code (in node_resource_abstract_instance.go) expects
			// us to return a valid object (even if we have errors). That means
			// no unknown values, no cty.NilVals, etc. So, we're going to go
			// ahead and call populateValue with whatever getReplacementSafe
			// gave us. getReplacementSafe is robust, so even in an error it
			// should have given us something we can use in populateValue.

			// Now get the replacement value. This function should be robust in
			// that it may return diagnostics explaining why it couldn't replace
			// the value, but it'll still return a value for us to use.
			value, valueDiags := populateValue(target, replacement, path)
			diags = diags.Append(valueDiags)

			// We always return a valid value, the diags are attached to the
			// global diags outside the nested function.
			return value, nil
		}

		// If we don't need to replace this value, then just return it
		// untouched.
		return target, nil
	})
	if err != nil {
		// This shouldn't actually happen - we never return an error from inside
		// the transform function. But, just in case:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Detail:   "Failed to generate values",
			Summary:  fmt.Sprintf("Terraform failed to generate computed values for a mocked resource, data source, or module: %s. This is a bug in Terraform - please report it.", err),
			Subject:  with.Range.Ptr(),
		})
	}

	return value, diags
}

func isNull(target cty.Value) bool {
	return target.IsNull()
}

func isUnknown(target cty.Value) bool {
	return !target.IsKnown()
}

func makeUnknown(target, _ cty.Value, _ cty.Path) (cty.Value, tfdiags.Diagnostics) {
	return cty.UnknownVal(target.Type()), nil
}

// ReplacementValue is just a helper struct that wraps the think we're
// interested in (the value) with some metadata that will make our diagnostics
// a bit more helpful.
type ReplacementValue struct {
	Value cty.Value
	Range hcl.Range
}

func (replacement ReplacementValue) makeKnown(target, with cty.Value, path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if with != cty.NilVal {
		// Then we have a pre-made value to replace it with. We'll make sure it
		// is compatible with a conversion, and then just return it in place.

		if value, err := convert.Convert(with, target.Type()); err != nil {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Failed to replace target attribute",
				fmt.Sprintf("Terraform could not replace the target type %s with the replacement value defined at %s within %s: %s.", target.Type().FriendlyName(), fmtPath(path), replacement.Range, err),
				path))

			// We still want to return a valid value here. If the conversion did
			// not work we carry on and just create a value instead. We've made
			// a note of the diagnostics tracking why it didn't work so the
			// overall operation will still fail, but we won't crash later on
			// because of an unknown value or something.

		} else {
			// Successful conversion! We can just return the new value.
			return value, diags
		}
	}

	// Otherwise, we'll have to generate some values.
	// We just return zero values for most of the types. The only exceptions are
	// objects and strings. For strings, we generate 8 random alphanumeric
	// characters. Objects need to be valid types, so we recurse through the
	// attributes and recursively call this function to generate values for
	// each attribute.

	switch {
	case target.Type().IsPrimitiveType():
		switch target.Type() {
		case cty.String:
			return cty.StringVal(str(8)), diags
		case cty.Number:
			return cty.Zero, diags
		case cty.Bool:
			return cty.False, diags
		default:
			panic(fmt.Errorf("unknown primitive type: %s", target.Type().FriendlyName()))
		}
	case target.Type().IsListType():
		return cty.ListValEmpty(target.Type().ElementType()), diags
	case target.Type().IsSetType():
		return cty.SetValEmpty(target.Type().ElementType()), diags
	case target.Type().IsMapType():
		return cty.MapValEmpty(target.Type().ElementType()), diags
	case target.Type().IsObjectType():
		children := make(map[string]cty.Value)
		for name, attribute := range target.Type().AttributeTypes() {
			child, childDiags := replacement.makeKnown(cty.UnknownVal(attribute), cty.NilVal, path.GetAttr(name))
			diags = diags.Append(childDiags)
			children[name] = child
		}
		return cty.ObjectVal(children), diags
	default:
		panic(fmt.Errorf("unknown complex type: %s", target.Type().FriendlyName()))
	}
}

// We can only do replacements if the replacement value is an object type.
func (replacement ReplacementValue) validate() bool {
	return replacement.Value == cty.NilVal || replacement.Value.Type().IsObjectType()
}

// getReplacementSafe walks the path to find any potential replacement value for
// a given path. We have implemented custom logic for walking the path here.
//
// This is to support nested block types. It's complicated to work out how to
// replace computed values within nested types. For example, how would a user
// say they just want to replace values at index 3? Or how would users indicate
// they want to replace anything at all within nested sets. The indices for sets
// will never be the same because the user supplied values will, by design, have
// values for the computed attributes which will be null or unknown within the
// values from Terraform so the paths will never match.
//
// What the above paragraph means is that for nested blocks and attributes,
// users can only specify a single replacement value that will apply to all
// the values within the nested collection.
//
// TODO(liamcervante): Revisit this function, is it possible and/or easy for us
// to support specific targeting of elements in collections?
func (replacement ReplacementValue) getReplacementSafe(path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if replacement.Value == cty.NilVal {
		return cty.NilVal, diags
	}

	// We want to provide a nice print out of the path in case of an error.
	// We'll format it as we go.
	var currentPath cty.Path

	// We are copying the implementation within AttributeByPath inside the
	// schema for this. We skip over GetIndexSteps as they'll be referring to
	// the intermediate nested blocks and attributes that we aren't capturing
	// within the user supplied mock values.
	current := replacement.Value
	for _, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:

			if !current.Type().IsObjectType() {
				// As we're still traversing the path, we expect things to be
				// objects at every level.
				diags = diags.Append(tfdiags.AttributeValue(
					tfdiags.Error,
					"Failed to replace target attribute",
					fmt.Sprintf("Terraform expected an object type at %s within the replacement value defined at %s, but found %s.", fmtPath(currentPath), replacement.Range, current.Type().FriendlyName()),
					currentPath))

				return cty.NilVal, diags
			}

			if !current.Type().HasAttribute(step.Name) {
				// Then we're not providing a replacement value for this path.
				return cty.NilVal, diags
			}

			current = current.GetAttr(step.Name)
		}

		currentPath = append(currentPath, step)
	}

	return current, diags
}

func fmtPath(path cty.Path) string {
	var current string

	first := true
	for _, step := range path {
		// Since we only ever parse the attribute steps when finding replacement
		// values, we can do the same again here.
		switch step := step.(type) {
		case cty.GetAttrStep:
			if first {
				first = false
				current = step.Name
				continue
			}
			current = fmt.Sprintf("%s.%s", current, step.Name)
		}
	}
	return current
}

func str(n int) string {
	b := make([]rune, n)
	for i := range b {
		if testRand != nil {
			b[i] = chars[testRand.Intn(len(chars))]
		} else {
			b[i] = chars[rand.Intn(len(chars))]
		}
	}
	return string(b)
}
