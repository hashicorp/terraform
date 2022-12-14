package change

import (
	"strings"
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
		"object_created": {
			change: Change{
				renderer: Object(map[string]Change{}),
				action:   plans.Create,
			},
			expected: "{}",
		},
		"object_created_with_attributes": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Primitive(nil, strptr("0")),
						action:   plans.Create,
					},
				}),
				action: plans.Create,
			},
			expected: `
{
      + attribute_one = 0
    }
`,
		},
		"object_deleted": {
			change: Change{
				renderer: Object(map[string]Change{}),
				action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"object_deleted_with_attributes": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Primitive(strptr("0"), nil),
						action:   plans.Delete,
					},
				}),
				action: plans.Delete,
			},
			expected: `
{
      - attribute_one = 0
    } -> null
`,
		},
		"nested_object_deleted": {
			change: Change{
				renderer: NestedObject(map[string]Change{}),
				action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"nested_object_deleted_with_attributes": {
			change: Change{
				renderer: NestedObject(map[string]Change{
					"attribute_one": {
						renderer: Primitive(strptr("0"), nil),
						action:   plans.Delete,
					},
				}),
				action: plans.Delete,
			},
			expected: `
{
      - attribute_one = 0 -> null
    } -> null
`,
		},
		"object_create_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Primitive(nil, strptr("0")),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      + attribute_one = 0
    }
`,
		},
		"object_update_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Primitive(strptr("0"), strptr("1")),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = 0 -> 1
    }
`,
		},
		"object_update_attribute_forces_replacement": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Primitive(strptr("0"), strptr("1")),
						action:   plans.Update,
					},
				}),
				action:  plans.Update,
				replace: true,
			},
			expected: `
{ # forces replacement
      ~ attribute_one = 0 -> 1
    }
`,
		},
		"object_delete_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Primitive(strptr("0"), nil),
						action:   plans.Delete,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      - attribute_one = 0
    }
`,
		},
		"object_ignore_unchanged_attributes": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Primitive(strptr("0"), strptr("1")),
						action:   plans.Update,
					},
					"attribute_two": {
						renderer: Primitive(strptr("0"), strptr("0")),
						action:   plans.NoOp,
					},
					"attribute_three": {
						renderer: Primitive(nil, strptr("1")),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      ~ attribute_one   = 0 -> 1
      + attribute_three = 1
        # (1 unchanged attribute hidden)
    }
`,
		},
		"object_create_sensitive_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Sensitive(nil, 1, false, true),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      + attribute_one = (sensitive)
    }
`,
		},
		"object_update_sensitive_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Sensitive(nil, 1, false, true),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = (sensitive)
    }
`,
		},
		"object_delete_sensitive_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Sensitive(nil, 1, false, true),
						action:   plans.Delete,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      - attribute_one = (sensitive)
    }
`,
		},
		"object_create_computed_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Computed(Change{renderer: nil}),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      + attribute_one = (known after apply)
    }
`,
		},
		"object_update_computed_attribute": {
			change: Change{
				renderer: Object(map[string]Change{
					"attribute_one": {
						renderer: Computed(Change{
							renderer: Primitive(strptr("1"), nil),
							action:   plans.Delete,
						}),
						action: plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = 1 -> (known after apply)
    }
`,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			expected := strings.TrimSpace(tc.expected)
			actual := colorize.Color(tc.change.Render(0, tc.opts))
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Fatalf("\nexpected:\n%s\nactual:\n%s\ndiff:\n%s\n", expected, actual, diff)
			}
		})
	}

}
