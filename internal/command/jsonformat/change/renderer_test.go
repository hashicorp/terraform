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
				replace:  false,
			},
			expected: "1",
		},
		"primitive_delete": {
			change: Change{
				renderer: Primitive(strptr("1"), nil),
				action:   plans.Delete,
				replace:  false,
			},
			expected: "1 -> null",
		},
		"primitive_delete_override": {
			change: Change{
				renderer: Primitive(strptr("1"), nil),
				action:   plans.Delete,
				replace:  false,
			},
			opts:     RenderOpts{overrideNullSuffix: true},
			expected: "1",
		},
		"primitive_update_to_null": {
			change: Change{
				renderer: Primitive(strptr("1"), nil),
				action:   plans.Update,
				replace:  false,
			},
			expected: "1 -> null",
		},
		"primitive_update_from_null": {
			change: Change{
				renderer: Primitive(nil, strptr("1")),
				action:   plans.Update,
				replace:  false,
			},
			expected: "null -> 1",
		},
		"primitive_update": {
			change: Change{
				renderer: Primitive(strptr("0"), strptr("1")),
				action:   plans.Update,
				replace:  false,
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
				replace:  false,
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
