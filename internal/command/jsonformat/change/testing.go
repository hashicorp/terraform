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
		primitive, ok := change.renderer.(primitiveRenderer)
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
