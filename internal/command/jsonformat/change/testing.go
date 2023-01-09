package change

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/plans"
)

type ValidateChangeFunc func(t *testing.T, change Change)

func ValidateChange(t *testing.T, f ValidateChangeFunc, change Change, expectedAction plans.Action, expectedReplace bool) {
	if change.replace != expectedReplace || change.action != expectedAction {
		t.Fatalf("\nreplace:\n\texpected:%t\n\tactual:%t\naction:\n\texpected:%s\n\tactual:%s", expectedReplace, change.replace, expectedAction, change.action)
	}

	f(t, change)
}

func ValidatePrimitive(before, after *string) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
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

func ValidateSensitive(before, after interface{}, beforeSensitive, afterSensitive bool) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
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

func ValidateComputed(before ValidateChangeFunc) ValidateChangeFunc {
	return func(t *testing.T, change Change) {
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
