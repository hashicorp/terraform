package change

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/plans"
)

type ValidateChangeFunc func(t *testing.T, change Change)

func validateChange(t *testing.T, change Change, expectedAction plans.Action, expectedReplace bool) {
	if change.replace != expectedReplace || change.action != expectedAction {
		t.Errorf("\nreplace:\n\texpected:%t\n\tactual:%t\naction:\n\texpected:%s\n\tactual:%s", expectedReplace, change.replace, expectedAction, change.action)
	}
}

func ValidatePrimitive(before, after interface{}, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		primitive, ok := change.renderer.(*primitiveRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		beforeDiff := cmp.Diff(primitive.before, before)
		afterDiff := cmp.Diff(primitive.after, after)

		if len(beforeDiff) > 0 || len(afterDiff) > 0 {
			t.Errorf("before diff: (%s), after diff: (%s)", beforeDiff, afterDiff)
		}
	}
}

func ValidateObject(attributes map[string]ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		object, ok := change.renderer.(*objectRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		if !object.overrideNullSuffix {
			t.Errorf("created the wrong type of object renderer")
		}

		validateMapType(t, object.attributes, attributes)
	}
}

func ValidateNestedObject(attributes map[string]ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		object, ok := change.renderer.(*objectRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		if object.overrideNullSuffix {
			t.Errorf("created the wrong type of object renderer")
		}

		validateMapType(t, object.attributes, attributes)
	}
}

func ValidateMap(elements map[string]ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		m, ok := change.renderer.(*mapRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		validateMapType(t, m.elements, elements)
	}
}

func validateMapType(t *testing.T, actual map[string]Change, expected map[string]ValidateChangeFunc) {
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

func ValidateList(elements []ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		list, ok := change.renderer.(*listRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		if !list.displayContext {
			t.Errorf("created the wrong type of list renderer")
		}

		validateSliceType(t, list.elements, elements)
	}
}

func ValidateNestedList(elements []ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		list, ok := change.renderer.(*listRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		if list.displayContext {
			t.Errorf("created the wrong type of list renderer")
		}

		validateSliceType(t, list.elements, elements)
	}
}

func ValidateSet(elements []ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		set, ok := change.renderer.(*setRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		validateSliceType(t, set.elements, elements)
	}
}

func validateSliceType(t *testing.T, actual []Change, expected []ValidateChangeFunc) {
	if len(actual) != len(expected) {
		t.Errorf("expected %d elements but found %d elements", len(expected), len(actual))
		return
	}

	for ix := 0; ix < len(expected); ix++ {
		expected[ix](t, actual[ix])
	}
}

func ValidateBlock(attributes map[string]ValidateChangeFunc, blocks map[string][]ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		block, ok := change.renderer.(*blockRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		validateKeys(t, block.attributes, attributes)
		validateKeys(t, block.blocks, blocks)

		for key, expected := range attributes {
			if actual, ok := block.attributes[key]; ok {
				expected(t, actual)
			}
		}

		for key, expected := range blocks {
			if actual, ok := block.blocks[key]; ok {
				if len(actual) != len(expected) {
					t.Errorf("expected %d blocks within %s but found %d elements", len(expected), key, len(actual))

					for ix := range expected {
						expected[ix](t, actual[ix])
					}
				}
			}
		}
	}
}

func ValidateTypeChange(before, after ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		typeChange, ok := change.renderer.(*typeChangeRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		before(t, typeChange.before)
		after(t, typeChange.after)
	}
}

func ValidateSensitive(inner ValidateChangeFunc, beforeSensitive, afterSensitive bool, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		sensitive, ok := change.renderer.(*sensitiveRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		if beforeSensitive != sensitive.beforeSensitive || afterSensitive != sensitive.afterSensitive {
			t.Errorf("before or after sensitive values don't match:\n\texpected; before: %t after: %t\n\tactual; before: %t, after: %t", beforeSensitive, afterSensitive, sensitive.beforeSensitive, sensitive.afterSensitive)
		}

		inner(t, sensitive.change)
	}
}

func ValidateComputed(before ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		computed, ok := change.renderer.(*computedRenderer)
		if !ok {
			t.Errorf("invalid renderer type: %T", change.renderer)
			return
		}

		if before == nil {
			if computed.before.renderer != nil {
				t.Errorf("did not expect a before renderer, but found one")
			}
			return
		}

		if computed.before.renderer == nil {
			t.Errorf("expected a before renderer, but found none")
		}

		before(t, computed.before)
	}
}
