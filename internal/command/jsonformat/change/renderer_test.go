package change

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/plans"
)

func TestRenderers(t *testing.T) {
	strptr := func(in string) *string {
		return &in
	}

	colorize := colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}

	tcs := map[string]struct {
		change   Change
		expected string
		opts     RenderOpts
	}{
		"primitive_create": {
			change: Change{
				renderer: Primitive(nil, strptr("1")),
				action:   plans.Create,
			},
			expected: "1",
		},
		"primitive_delete": {
			change: Change{
				renderer: Primitive(strptr("1"), nil),
				action:   plans.Delete,
			},
			expected: "1 -> null",
		},
		"primitive_delete_override": {
			change: Change{
				renderer: Primitive(strptr("1"), nil),
				action:   plans.Delete,
			},
			opts:     RenderOpts{overrideNullSuffix: true},
			expected: "1",
		},
		"primitive_update_to_null": {
			change: Change{
				renderer: Primitive(strptr("1"), nil),
				action:   plans.Update,
			},
			expected: "1 -> null",
		},
		"primitive_update_from_null": {
			change: Change{
				renderer: Primitive(nil, strptr("1")),
				action:   plans.Update,
			},
			expected: "null -> 1",
		},
		"primitive_update": {
			change: Change{
				renderer: Primitive(strptr("0"), strptr("1")),
				action:   plans.Update,
			},
			expected: "0 -> 1",
		},
		"primitive_update_replace": {
			change: Change{
				renderer: Primitive(strptr("0"), strptr("1")),
				action:   plans.Update,
				replace:  true,
			},
			expected: "0 -> 1 # forces replacement",
		},
		"sensitive_update": {
			change: Change{
				renderer: Sensitive("0", "1", true, true),
				action:   plans.Update,
			},
			expected: "(sensitive)",
		},
		"sensitive_update_replace": {
			change: Change{
				renderer: Sensitive("0", "1", true, true),
				action:   plans.Update,
				replace:  true,
			},
			expected: "(sensitive) # forces replacement",
		},
		"computed_create": {
			change: Change{
				renderer: Computed(Change{}),
				action:   plans.Create,
			},
			expected: "(known after apply)",
		},
		"computed_update": {
			change: Change{
				renderer: Computed(Change{
					renderer: Primitive(strptr("0"), nil),
					action:   plans.Delete,
				}),
				action: plans.Update,
			},
			expected: "0 -> (known after apply)",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			actual := colorize.Color(tc.change.Render(0, tc.opts))
			if diff := cmp.Diff(tc.expected, actual); len(diff) > 0 {
				t.Fatalf("\nexpected:\n%s\nactual:\n%s\ndiff:\n%s\n", tc.expected, actual, diff)
			}
		})
	}

}
