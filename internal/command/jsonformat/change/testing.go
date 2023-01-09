package change

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/plans"
)

type ValidateChangeFunc func(t *testing.T, change Change)

func validateChange(t *testing.T, change Change, expectedAction plans.Action, expectedReplace bool) {
	if change.replace != expectedReplace || change.action != expectedAction {
		t.Fatalf("\nreplace:\n\texpected:%t\n\tactual:%t\naction:\n\texpected:%s\n\tactual:%s", expectedReplace, change.replace, expectedAction, change.action)
	}
}

func ValidatePrimitive(before, after *string, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		primitive, ok := change.renderer.(*primitiveRenderer)
		if !ok {
			t.Fatalf("invalid renderer type: %T", change.renderer)
		}

		beforeDiff := cmp.Diff(primitive.before, before)
		afterDiff := cmp.Diff(primitive.after, after)

		if len(beforeDiff) > 0 || len(afterDiff) > 0 {
			t.Fatalf("before diff: (%s), after diff: (%s)", beforeDiff, afterDiff)
		}
	}
}

func ValidateObject(attributes map[string]ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		object, ok := change.renderer.(*objectRenderer)
		if !ok {
			t.Fatalf("invalid renderer type: %T", change.renderer)
		}

		if !object.overrideNullSuffix {
			t.Fatalf("created the wrong type of object renderer")
		}

		validateObject(t, object, attributes)
	}
}

func ValidateNestedObject(attributes map[string]ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		object, ok := change.renderer.(*objectRenderer)
		if !ok {
			t.Fatalf("invalid renderer type: %T", change.renderer)
		}

		if object.overrideNullSuffix {
			t.Fatalf("created the wrong type of object renderer")
		}

		validateObject(t, object, attributes)
	}
}

func validateObject(t *testing.T, object *objectRenderer, attributes map[string]ValidateChangeFunc) {
	if len(object.attributes) != len(attributes) {
		t.Fatalf("expected %d attributes but found %d attributes", len(attributes), len(object.attributes))
	}

	var missing []string
	for key, expected := range attributes {
		actual, ok := object.attributes[key]
		if !ok {
			missing = append(missing, key)
		}

		if len(missing) > 0 {
			continue
		}

		expected(t, actual)
	}

	if len(missing) > 0 {
		t.Fatalf("missing the following attributes: %s", strings.Join(missing, ", "))
	}
}

func ValidateMap(elements map[string]ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		m, ok := change.renderer.(*mapRenderer)
		if !ok {
			t.Fatalf("invalid renderer type: %T", change.renderer)
		}

		if len(m.elements) != len(elements) {
			t.Fatalf("expected %d elements but found %d elements", len(elements), len(m.elements))
		}

		var missing []string
		for key, expected := range elements {
			actual, ok := m.elements[key]
			if !ok {
				missing = append(missing, key)
			}

			if len(missing) > 0 {
				continue
			}

			expected(t, actual)
		}

		if len(missing) > 0 {
			t.Fatalf("missing the following elements: %s", strings.Join(missing, ", "))
		}
	}
}

func ValidateSensitive(before, after interface{}, beforeSensitive, afterSensitive bool, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		sensitive, ok := change.renderer.(*sensitiveRenderer)
		if !ok {
			t.Fatalf("invalid renderer type: %T", change.renderer)
		}

		if beforeSensitive != sensitive.beforeSensitive || afterSensitive != sensitive.afterSensitive {
			t.Fatalf("before or after sensitive values don't match:\n\texpected; before: %t after: %t\n\tactual; before: %t, after: %t", beforeSensitive, afterSensitive, sensitive.beforeSensitive, sensitive.afterSensitive)
		}

		beforeDiff := cmp.Diff(sensitive.before, before)
		afterDiff := cmp.Diff(sensitive.after, after)

		if len(beforeDiff) > 0 || len(afterDiff) > 0 {
			t.Fatalf("before diff: (%s), after diff: (%s)", beforeDiff, afterDiff)
		}
	}
}

func ValidateComputed(before ValidateChangeFunc, action plans.Action, replace bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
		validateChange(t, change, action, replace)

		computed, ok := change.renderer.(*computedRenderer)
		if !ok {
			t.Fatalf("invalid renderer type: %T", change.renderer)
		}

		if before == nil {
			if computed.before.renderer != nil {
				t.Fatalf("did not expect a before renderer, but found one")
			}
			return
		}

		if computed.before.renderer == nil {
			t.Fatalf("expected a before renderer, but found none")
		}

		before(t, computed.before)
	}
}
