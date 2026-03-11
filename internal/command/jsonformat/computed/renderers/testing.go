// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/plans"
)

type ValidateDiffFunction func(t *testing.T, diff computed.Diff)

func validateDiff(t *testing.T, diff computed.Diff, expectedAction plans.Action, expectedReplace bool) {
	if diff.Replace != expectedReplace || diff.Action != expectedAction {
		t.Errorf("\nreplace:\n\texpected:%t\n\tactual:%t\naction:\n\texpected:%s\n\tactual:%s", expectedReplace, diff.Replace, expectedAction, diff.Action)
	}
}

func ValidatePrimitive(before, after interface{}, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		primitive, ok := diff.Renderer.(*primitiveRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		beforeDiff := cmp.Diff(primitive.before, before)
		afterDiff := cmp.Diff(primitive.after, after)

		if len(beforeDiff) > 0 || len(afterDiff) > 0 {
			t.Errorf("before diff: (%s), after diff: (%s)", beforeDiff, afterDiff)
		}
	}
}

func ValidateObject(attributes map[string]ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		object, ok := diff.Renderer.(*objectRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		if !object.overrideNullSuffix {
			t.Errorf("created the wrong type of object renderer")
		}

		validateMapType(t, object.attributes, attributes)
	}
}

func ValidateNestedObject(attributes map[string]ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		object, ok := diff.Renderer.(*objectRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		if object.overrideNullSuffix {
			t.Errorf("created the wrong type of object renderer")
		}

		validateMapType(t, object.attributes, attributes)
	}
}

func ValidateMap(elements map[string]ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		m, ok := diff.Renderer.(*mapRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		validateMapType(t, m.elements, elements)
	}
}

func validateMapType(t *testing.T, actual map[string]computed.Diff, expected map[string]ValidateDiffFunction) {
	validateKeys(t, actual, expected)

	for key, expected := range expected {
		if actual, ok := actual[key]; ok {
			expected(t, actual)
		}
	}
}

func validateKeys[C, V any](t *testing.T, actual map[string]C, expected map[string]V) {
	if len(actual) != len(expected) {

		var actualAttributes []string
		var expectedAttributes []string

		for key := range actual {
			actualAttributes = append(actualAttributes, key)
		}
		for key := range expected {
			expectedAttributes = append(expectedAttributes, key)
		}

		sort.Strings(actualAttributes)
		sort.Strings(expectedAttributes)

		if diff := cmp.Diff(actualAttributes, expectedAttributes); len(diff) > 0 {
			t.Errorf("actual and expected attributes did not match: %s", diff)
		}
	}
}

func ValidateList(elements []ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		list, ok := diff.Renderer.(*listRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		if !list.displayContext {
			t.Errorf("created the wrong type of list renderer")
		}

		validateSliceType(t, list.elements, elements)
	}
}

func ValidateNestedList(elements []ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		list, ok := diff.Renderer.(*listRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		if list.displayContext {
			t.Errorf("created the wrong type of list renderer")
		}

		validateSliceType(t, list.elements, elements)
	}
}

func ValidateSet(elements []ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		set, ok := diff.Renderer.(*setRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		validateSliceType(t, set.elements, elements)
	}
}

func validateSliceType(t *testing.T, actual []computed.Diff, expected []ValidateDiffFunction) {
	if len(actual) != len(expected) {
		t.Errorf("expected %d elements but found %d elements", len(expected), len(actual))
		return
	}

	for ix := 0; ix < len(expected); ix++ {
		expected[ix](t, actual[ix])
	}
}

func ValidateBlock(
	attributes map[string]ValidateDiffFunction,
	singleBlocks map[string]ValidateDiffFunction,
	listBlocks map[string][]ValidateDiffFunction,
	mapBlocks map[string]map[string]ValidateDiffFunction,
	setBlocks map[string][]ValidateDiffFunction,
	action plans.Action,
	replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		block, ok := diff.Renderer.(*blockRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		validateKeys(t, block.attributes, attributes)
		validateKeys(t, block.blocks.SingleBlocks, singleBlocks)
		validateKeys(t, block.blocks.ListBlocks, listBlocks)
		validateKeys(t, block.blocks.MapBlocks, mapBlocks)
		validateKeys(t, block.blocks.SetBlocks, setBlocks)

		for key, expected := range attributes {
			if actual, ok := block.attributes[key]; ok {
				expected(t, actual)
			}
		}

		for key, expected := range singleBlocks {
			expected(t, block.blocks.SingleBlocks[key])
		}

		for key, expected := range listBlocks {
			if actual, ok := block.blocks.ListBlocks[key]; ok {
				if len(actual) != len(expected) {
					t.Errorf("expected %d blocks within %s but found %d elements", len(expected), key, len(actual))
				}
				for ix := range expected {
					expected[ix](t, actual[ix])
				}
			}
		}

		for key, expected := range setBlocks {
			if actual, ok := block.blocks.SetBlocks[key]; ok {
				if len(actual) != len(expected) {
					t.Errorf("expected %d blocks within %s but found %d elements", len(expected), key, len(actual))
				}
				for ix := range expected {
					expected[ix](t, actual[ix])
				}
			}
		}

		for key, expected := range setBlocks {
			if actual, ok := block.blocks.SetBlocks[key]; ok {
				if len(actual) != len(expected) {
					t.Errorf("expected %d blocks within %s but found %d elements", len(expected), key, len(actual))
				}
				for ix := range expected {
					expected[ix](t, actual[ix])
				}
			}
		}

		for key, expected := range mapBlocks {
			if actual, ok := block.blocks.MapBlocks[key]; ok {
				if len(actual) != len(expected) {
					t.Errorf("expected %d blocks within %s but found %d elements", len(expected), key, len(actual))
				}
				for dKey := range expected {
					expected[dKey](t, actual[dKey])
				}
			}
		}
	}
}

func ValidateTypeChange(before, after ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		typeChange, ok := diff.Renderer.(*typeChangeRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		before(t, typeChange.before)
		after(t, typeChange.after)
	}
}

func ValidateSensitive(inner ValidateDiffFunction, beforeSensitive, afterSensitive bool, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		sensitive, ok := diff.Renderer.(*sensitiveRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		if beforeSensitive != sensitive.beforeSensitive || afterSensitive != sensitive.afterSensitive {
			t.Errorf("before or after sensitive values don't match:\n\texpected; before: %t after: %t\n\tactual; before: %t, after: %t", beforeSensitive, afterSensitive, sensitive.beforeSensitive, sensitive.afterSensitive)
		}

		inner(t, sensitive.inner)
	}
}

func ValidateUnknown(before ValidateDiffFunction, action plans.Action, replace bool) ValidateDiffFunction {
	return func(t *testing.T, diff computed.Diff) {
		validateDiff(t, diff, action, replace)

		unknown, ok := diff.Renderer.(*unknownRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", diff.Renderer)
			return
		}

		if before == nil {
			if unknown.before.Renderer != nil {
				t.Errorf("did not expect a before renderer, but found one")
			}
			return
		}

		if unknown.before.Renderer == nil {
			t.Errorf("expected a before renderer, but found none")
		}

		before(t, unknown.before)
	}
}
