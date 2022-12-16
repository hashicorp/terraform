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
		"map_create_empty": {
			change: Change{
				renderer: Map(map[string]Change{}),
				action:   plans.Create,
			},
			expected: "{}",
		},
		"map_create": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Primitive(nil, strptr("new")),
						action:   plans.Create,
					},
				}),
				action: plans.Create,
			},
			expected: `
{
      + "element_one" = new
    }
`,
		},
		"map_delete_empty": {
			change: Change{
				renderer: Map(map[string]Change{}),
				action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"map_delete": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Primitive(strptr("old"), nil),
						action:   plans.Delete,
					},
				}),
				action: plans.Delete,
			},
			expected: `
{
      - "element_one" = old
    } -> null
`,
		},
		"map_create_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Primitive(nil, strptr("new")),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      + "element_one" = new
    }
`,
		},
		"map_update_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Primitive(strptr("old"), strptr("new")),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = old -> new
    }
`,
		},
		"map_delete_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Primitive(strptr("old"), nil),
						action:   plans.Delete,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      - "element_one" = old -> null
    }
`,
		},
		"map_update_forces_replacement": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Primitive(strptr("old"), strptr("new")),
						action:   plans.Update,
					},
				}),
				action:  plans.Update,
				replace: true,
			},
			expected: `
{ # forces replacement
      ~ "element_one" = old -> new
    }
`,
		},
		"map_ignore_unchanged_elements": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Primitive(nil, strptr("new")),
						action:   plans.Create,
					},
					"element_two": {
						renderer: Primitive(strptr("old"), strptr("old")),
						action:   plans.NoOp,
					},
					"element_three": {
						renderer: Primitive(strptr("old"), strptr("new")),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      + "element_one"   = new
      ~ "element_three" = old -> new
        # (1 unchanged element hidden)
    }
`,
		},
		"map_create_sensitive_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Sensitive(nil, 1, false, true),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      + "element_one" = (sensitive)
    }
`,
		},
		"map_update_sensitive_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Sensitive(0, 1, true, true),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = (sensitive)
    }
`,
		},
		"map_update_sensitive_element_status": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Sensitive(0, 0, true, false),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change. The value is unchanged.
      ~ "element_one" = (sensitive)
    }
`,
		},
		"map_delete_sensitive_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Sensitive(0, nil, true, false),
						action:   plans.Delete,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      - "element_one" = (sensitive) -> null
    }
`,
		},
		"map_create_computed_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
						renderer: Computed(Change{}),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
{
      + "element_one" = (known after apply)
    }
`,
		},
		"map_update_computed_element": {
			change: Change{
				renderer: Map(map[string]Change{
					"element_one": {
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
      ~ "element_one" = 1 -> (known after apply)
    }
`,
		},
		"list_create_empty": {
			change: Change{
				renderer: List([]Change{}),
				action:   plans.Create,
			},
			expected: "[]",
		},
		"list_create": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(nil, strptr("1")),
						action:   plans.Create,
					},
				}),
				action: plans.Create,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"list_delete_empty": {
			change: Change{
				renderer: List([]Change{}),
				action:   plans.Delete,
			},
			expected: "[] -> null",
		},
		"list_delete": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(strptr("1"), nil),
						action:   plans.Delete,
					},
				}),
				action: plans.Delete,
			},
			expected: `
[
      - 1,
    ] -> null
`,
		},
		"list_create_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(nil, strptr("1")),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"list_update_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(strptr("0"), strptr("1")),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      ~ 0 -> 1,
    ]
`,
		},
		"list_replace_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(strptr("0"), nil),
						action:   plans.Delete,
					},
					{
						renderer: Primitive(nil, strptr("1")),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      - 0,
      + 1,
    ]
`,
		},
		"list_delete_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(strptr("0"), nil),
						action:   plans.Delete,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      - 0,
    ]
`,
		},
		"list_update_forces_replacement": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(strptr("0"), strptr("1")),
						action:   plans.Update,
					},
				}),
				action:  plans.Update,
				replace: true,
			},
			expected: `
[ # forces replacement
      ~ 0 -> 1,
    ]
`,
		},
		"list_update_ignores_unchanged": {
			change: Change{
				renderer: NestedList([]Change{
					{
						renderer: Primitive(strptr("0"), strptr("0")),
						action:   plans.NoOp,
					},
					{
						renderer: Primitive(strptr("1"), strptr("1")),
						action:   plans.NoOp,
					},
					{
						renderer: Primitive(strptr("2"), strptr("5")),
						action:   plans.Update,
					},
					{
						renderer: Primitive(strptr("3"), strptr("3")),
						action:   plans.NoOp,
					},
					{
						renderer: Primitive(strptr("4"), strptr("4")),
						action:   plans.NoOp,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      ~ 2 -> 5,
        # (4 unchanged elements hidden)
    ]
`,
		},
		"list_update_ignored_unchanged_with_context": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Primitive(strptr("0"), strptr("0")),
						action:   plans.NoOp,
					},
					{
						renderer: Primitive(strptr("1"), strptr("1")),
						action:   plans.NoOp,
					},
					{
						renderer: Primitive(strptr("2"), strptr("5")),
						action:   plans.Update,
					},
					{
						renderer: Primitive(strptr("3"), strptr("3")),
						action:   plans.NoOp,
					},
					{
						renderer: Primitive(strptr("4"), strptr("4")),
						action:   plans.NoOp,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
        # (1 unchanged element hidden)
        1,
      ~ 2 -> 5,
        3,
        # (1 unchanged element hidden)
    ]
`,
		},
		"list_create_sensitive_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Sensitive(nil, 1, false, true),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      + (sensitive),
    ]
`,
		},
		"list_delete_sensitive_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Sensitive(1, nil, true, false),
						action:   plans.Delete,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      - (sensitive),
    ]
`,
		},
		"list_update_sensitive_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Sensitive(nil, 1, false, true),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      ~ (sensitive),
    ]
`,
		},
		"list_update_sensitive_element_status": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Sensitive(1, 1, false, true),
						action:   plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      # Warning: this attribute value will be marked as sensitive and will not
      # display in UI output after applying this change. The value is unchanged.
      ~ (sensitive),
    ]
`,
		},
		"list_create_computed_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Computed(Change{}),
						action:   plans.Create,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      + (known after apply),
    ]
`,
		},
		"list_update_computed_element": {
			change: Change{
				renderer: List([]Change{
					{
						renderer: Computed(Change{
							renderer: Primitive(strptr("0"), nil),
							action:   plans.Delete,
						}),
						action: plans.Update,
					},
				}),
				action: plans.Update,
			},
			expected: `
[
      ~ 0 -> (known after apply),
    ]
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
