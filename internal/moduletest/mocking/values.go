// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// PlanComputedValuesForResource accepts a target value, and populates its computed
// values with values from the provider 'with' argument, and if 'with' is not provided,
// it sets the computed values to cty.UnknownVal.
//
// The latter behaviour simulates the behaviour of a plan request in a real
// provider.
func PlanComputedValuesForResource(original cty.Value, with *MockedData, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	if with == nil {
		with = &MockedData{
			Value:             cty.NilVal,
			ComputedAsUnknown: true,
		}
	}
	return populateComputedValues(original, *with, schema, isNull)
}

// ApplyComputedValuesForResource accepts a target value, and populates it
// either with values from the provided with argument, or with generated values
// created semi-randomly. This will only target values that are computed and
// unknown.
//
// This method basically simulates the behaviour of an apply request in a real
// provider.
func ApplyComputedValuesForResource(original cty.Value, with *MockedData, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	if with == nil {
		with = &MockedData{
			Value: cty.NilVal,
		}
	}
	return populateComputedValues(original, *with, schema, isUnknown)
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
func ComputedValuesForDataSource(original cty.Value, with *MockedData, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	if with == nil {
		with = &MockedData{
			Value: cty.NilVal,
		}
	}
	return populateComputedValues(original, *with, schema, isNull)
}

type processValue func(value cty.Value) bool

type generateValue func(attribute *configschema.Attribute, with cty.Value, path cty.Path) (cty.Value, tfdiags.Diagnostics)

func populateComputedValues(target cty.Value, with MockedData, schema *configschema.Block, processValue processValue) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var generateValue generateValue
	// If the computed attributes should be ignored, then we will generate
	// unknown values for them, otherwise we will
	// generate their values based on the mocked data.
	if with.ComputedAsUnknown {
		generateValue = makeUnknown
	} else {
		generateValue = with.makeKnown
	}

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

		// We still need to produce valid data for this. So, let's pretend we
		// had no mocked data. We still return the error diagnostic so whatever
		// operation was happening will still fail, but we won't cause any
		// panics or anything.
		with = MockedData{
			Value: cty.NilVal,
		}
	}

	// We're going to search for any elements within the target value that meet
	// the joint criteria of being computed and whatever processValue is
	// checking.
	//
	// We'll then replace anything that meets the criteria with the output of
	// generateValue.
	//
	// This transform should be robust (in that it should never fail), the
	// inner call to generateValue should be robust as well so it should always
	// return a valid value for us to use even if the embedded diagnostics
	// return errors.
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
			data, dataDiags := with.getMockedDataForPath(path)
			diags = diags.Append(dataDiags)

			// Upstream code (in node_resource_abstract_instance.go) expects
			// us to return a valid object (even if we have errors). That means
			// no unknown values, no cty.NilVals, etc. So, we're going to go
			// ahead and call generateValue with whatever getMockedDataForPath
			// gave us. getMockedDataForPath is robust, so even in an error it
			// should have given us something we can use in generateValue.

			// Now get the replacement value. This function should be robust in
			// that it may return diagnostics explaining why it couldn't replace
			// the value, but it'll still return a value for us to use.
			value, valueDiags := generateValue(attribute, data, path)
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

// makeUnknown produces an unknown value for the provided attribute. This is
// basically the output of a plan() call for a computed attribute in a mocked
// resource.
func makeUnknown(target *configschema.Attribute, _ cty.Value, _ cty.Path) (cty.Value, tfdiags.Diagnostics) {
	return cty.UnknownVal(target.ImpliedType()), nil
}

// MockedData wraps the value and the source location of the value into a single
// struct for easy access.
type MockedData struct {
	Value             cty.Value
	Range             hcl.Range
	ComputedAsUnknown bool // If true, computed values are replaced with unknown, otherwise they are replaced with overridden or generated values.
}

// NewMockedData creates a new MockedData struct with the given value and range.
func NewMockedData(value cty.Value, computedAsUnknown bool, rng hcl.Range) MockedData {
	return MockedData{
		Value:             value,
		ComputedAsUnknown: computedAsUnknown,
		Range:             rng,
	}
}

// makeKnown produces a valid value for the given attribute. The input value
// can provide data for this attribute or child attributes if this attribute
// represents an object. The input value is expected to be a representation of
// the schema of this attribute rather than a direct value.
func (data MockedData) makeKnown(attribute *configschema.Attribute, with cty.Value, path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if with != cty.NilVal {
		// Then we have a pre-made value to use as the basis for our value. We
		// just need to make sure the value is of the right type.

		if value, err := FillAttribute(with, attribute); err != nil {
			var relPath cty.Path
			if err, ok := err.(cty.PathError); ok {
				relPath = err.Path
			}

			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Failed to compute attribute",
				fmt.Sprintf("Terraform could not compute a value for the target type %s with the mocked data defined at %s with the attribute %q: %s.", attribute.ImpliedType().FriendlyName(), data.Range, tfdiags.FormatCtyPath(append(path, relPath...)), err),
				path))

			// We still want to return a valid value here. If the conversion did
			// not work we carry on and just create a value instead. We've made
			// a note of the diagnostics tracking why it didn't work so the
			// overall operation will still fail, but we won't crash later on
			// because of an unknown value or something.

			// Fall through to the GenerateValueForAttribute call below.
		} else {
			// Successful conversion! We can just return the new value.
			return value, diags
		}
	}

	// Otherwise, we'll have to generate some values.
	return GenerateValueForAttribute(attribute), diags
}

// We can only do replacements if the replacement value is an object type.
func (data MockedData) validate() bool {
	return data.Value == cty.NilVal || data.Value.Type().IsObjectType()
}

// getMockedDataForPath walks the path to find any potential mock data for the
// given path. We have implemented custom logic for walking the path here.
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
func (data MockedData) getMockedDataForPath(path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if data.Value == cty.NilVal {
		return cty.NilVal, diags
	}

	// We want to provide a nice print out of the path in case of an error.
	// We'll format it as we go.
	var currentPath cty.Path

	// We are copying the implementation within AttributeByPath inside the
	// schema for this. We skip over GetIndexSteps as they'll be referring to
	// the intermediate nested blocks and attributes that we aren't capturing
	// within the user supplied mock values.
	current := data.Value
	for _, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:

			if !current.Type().IsObjectType() {
				// As we're still traversing the path, we expect things to be
				// objects at every level.
				diags = diags.Append(tfdiags.AttributeValue(
					tfdiags.Error,
					"Failed to compute attribute",
					fmt.Sprintf("Terraform expected an object type for attribute %q defined within the mocked data at %s, but found %s.", tfdiags.FormatCtyPath(currentPath), data.Range, current.Type().FriendlyName()),
					currentPath))

				return cty.NilVal, diags
			}

			if !current.Type().HasAttribute(step.Name) {
				// Then we have no mocked data for this attribute.
				return cty.NilVal, diags
			}

			current = current.GetAttr(step.Name)
		}

		currentPath = append(currentPath, step)
	}

	return current, diags
}
