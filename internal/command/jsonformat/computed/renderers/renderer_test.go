// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package renderers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/plans"
)

func TestRenderers_Human(t *testing.T) {
	colorize := colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}

	tcs := map[string]struct {
		diff     computed.Diff
		expected string
		opts     computed.RenderHumanOpts
	}{
		// We're using the string "null" in these tests to demonstrate the
		// difference between rendering an actual string and rendering a null
		// value.
		"primitive_create_string": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "null", cty.String),
				Action:   plans.Create,
			},
			expected: "\"null\"",
		},
		"primitive_delete_string": {
			diff: computed.Diff{
				Renderer: Primitive("null", nil, cty.String),
				Action:   plans.Delete,
			},
			expected: "\"null\" -> null",
		},
		"primitive_update_string_to_null": {
			diff: computed.Diff{
				Renderer: Primitive("null", nil, cty.String),
				Action:   plans.Update,
			},
			expected: "\"null\" -> null",
		},
		"primitive_update_string_from_null": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "null", cty.String),
				Action:   plans.Update,
			},
			expected: "null -> \"null\"",
		},
		"primitive_update_multiline_string_to_null": {
			diff: computed.Diff{
				Renderer: Primitive("nu\nll", nil, cty.String),
				Action:   plans.Update,
			},
			expected: `
<<-EOT
      - nu
      - ll
      + null
    EOT
`,
		},
		"primitive_update_multiline_string_from_null": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "nu\nll", cty.String),
				Action:   plans.Update,
			},
			expected: `
<<-EOT
      - null
      + nu
      + ll
    EOT
`,
		},
		"primitive_update_json_string_to_null": {
			diff: computed.Diff{
				Renderer: Primitive("[null]", nil, cty.String),
				Action:   plans.Update,
			},
			expected: `
jsonencode(
        [
          - null,
        ]
    ) -> null
`,
		},
		"primitive_update_json_string_from_null": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "[null]", cty.String),
				Action:   plans.Update,
			},
			expected: `
null -> jsonencode(
        [
          + null,
        ]
    )
`,
		},
		"primitive_create_fake_json": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "[\"hello\"] and some more", cty.String),
				Action:   plans.Create,
			},
			expected: `"[\"hello\"] and some more"`,
		},
		"primitive_create_null_string": {
			diff: computed.Diff{
				Renderer: Primitive(nil, nil, cty.String),
				Action:   plans.Create,
			},
			expected: "null",
		},
		"primitive_delete_null_string": {
			diff: computed.Diff{
				Renderer: Primitive(nil, nil, cty.String),
				Action:   plans.Delete,
			},
			expected: "null",
		},
		"primitive_create": {
			diff: computed.Diff{
				Renderer: Primitive(nil, json.Number("1"), cty.Number),
				Action:   plans.Create,
			},
			expected: "1",
		},
		"primitive_delete": {
			diff: computed.Diff{
				Renderer: Primitive(json.Number("1"), nil, cty.Number),
				Action:   plans.Delete,
			},
			expected: "1 -> null",
		},
		"primitive_delete_override": {
			diff: computed.Diff{
				Renderer: Primitive(json.Number("1"), nil, cty.Number),
				Action:   plans.Delete,
			},
			opts:     computed.RenderHumanOpts{OverrideNullSuffix: true},
			expected: "1",
		},
		"primitive_update_to_null": {
			diff: computed.Diff{
				Renderer: Primitive(json.Number("1"), nil, cty.Number),
				Action:   plans.Update,
			},
			expected: "1 -> null",
		},
		"primitive_update_from_null": {
			diff: computed.Diff{
				Renderer: Primitive(nil, json.Number("1"), cty.Number),
				Action:   plans.Update,
			},
			expected: "null -> 1",
		},
		"primitive_update": {
			diff: computed.Diff{
				Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
				Action:   plans.Update,
			},
			expected: "0 -> 1",
		},
		"primitive_update_long_float": {
			diff: computed.Diff{
				Renderer: Primitive(json.Number("123456789"), json.Number("987654321"), cty.Number),
				Action:   plans.Update,
			},
			expected: "123456789 -> 987654321",
		},
		"primitive_update_long_float_with_decimals": {
			diff: computed.Diff{
				Renderer: Primitive(json.Number("1.23456789"), json.Number("9.87654321"), cty.Number),
				Action:   plans.Update,
			},
			expected: "1.23456789 -> 9.87654321",
		},
		"primitive_create_very_long_float": {
			diff: computed.Diff{
				Renderer: Primitive(nil, json.Number("9223372036854773256"), cty.Number),
				Action:   plans.Create,
			},
			expected: "9223372036854773256",
		},
		"primitive_update_replace": {
			diff: computed.Diff{
				Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
				Action:   plans.Update,
				Replace:  true,
			},
			expected: "0 -> 1 # forces replacement",
		},
		"primitive_multiline_string_create": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "hello\nworld", cty.String),
				Action:   plans.Create,
			},
			expected: `
<<-EOT
        hello
        world
    EOT
`,
		},
		"primitive_multiline_string_delete": {
			diff: computed.Diff{
				Renderer: Primitive("hello\nworld", nil, cty.String),
				Action:   plans.Delete,
			},
			expected: `
<<-EOT
        hello
        world
    EOT -> null
`,
		},
		"primitive_multiline_string_update": {
			diff: computed.Diff{
				Renderer: Primitive("hello\nold\nworld", "hello\nnew\nworld", cty.String),
				Action:   plans.Update,
			},
			expected: `
<<-EOT
        hello
      - old
      + new
        world
    EOT
`,
		},
		"primitive_json_string_create": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", cty.String),
				Action:   plans.Create,
			},
			expected: `
jsonencode(
        {
          + key_one = "value_one"
          + key_two = "value_two"
        }
    )
`,
		},
		"primitive_json_string_delete": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", nil, cty.String),
				Action:   plans.Delete,
			},
			expected: `
jsonencode(
        {
          - key_one = "value_one"
          - key_two = "value_two"
        }
    ) -> null
`,
		},
		"primitive_json_string_update": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", "{\"key_one\": \"value_one\",\"key_two\":\"value_two\",\"key_three\":\"value_three\"}", cty.String),
				Action:   plans.Update,
			},
			expected: `
jsonencode(
      ~ {
          + key_three = "value_three"
            # (2 unchanged attributes hidden)
        }
    )
`,
		},
		"primitive_json_explicit_nulls": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\":\"value_one\",\"key_two\":\"value_two\"}", "{\"key_one\":null}", cty.String),
				Action:   plans.Update,
			},
			expected: `
jsonencode(
      ~ {
          ~ key_one = "value_one" -> null
          - key_two = "value_two"
        }
    )
`,
		},
		"primitive_fake_json_string_update": {
			diff: computed.Diff{
				// This isn't valid JSON, our renderer should be okay with it.
				Renderer: Primitive("{\"key_one\": \"value_one\",\"key_two\":\"value_two\"", "{\"key_one\": \"value_one\",\"key_two\":\"value_two\",\"key_three\":\"value_three\"", cty.String),
				Action:   plans.Update,
			},
			expected: "\"{\\\"key_one\\\": \\\"value_one\\\",\\\"key_two\\\":\\\"value_two\\\"\" -> \"{\\\"key_one\\\": \\\"value_one\\\",\\\"key_two\\\":\\\"value_two\\\",\\\"key_three\\\":\\\"value_three\\\"\"",
		},
		"primitive_multiline_to_json_update": {
			diff: computed.Diff{
				Renderer: Primitive("hello\nworld", "{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", cty.String),
				Action:   plans.Update,
			},
			expected: `
<<-EOT
        hello
        world
    EOT -> jsonencode(
        {
          + key_one = "value_one"
          + key_two = "value_two"
        }
    )
`,
		},
		"primitive_json_to_multiline_update": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", "hello\nworld", cty.String),
				Action:   plans.Update,
			},
			expected: `
jsonencode(
        {
          - key_one = "value_one"
          - key_two = "value_two"
        }
    ) -> <<-EOT
        hello
        world
    EOT
`,
		},
		"primitive_json_to_string_update": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", "hello world", cty.String),
				Action:   plans.Update,
			},
			expected: `
jsonencode(
        {
          - key_one = "value_one"
          - key_two = "value_two"
        }
    ) -> "hello world"
`,
		},
		"primitive_string_to_json_update": {
			diff: computed.Diff{
				Renderer: Primitive("hello world", "{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", cty.String),
				Action:   plans.Update,
			},
			expected: `
"hello world" -> jsonencode(
        {
          + key_one = "value_one"
          + key_two = "value_two"
        }
    )
`,
		},
		"primitive_multi_to_single_update": {
			diff: computed.Diff{
				Renderer: Primitive("hello\nworld", "hello world", cty.String),
				Action:   plans.Update,
			},
			expected: `
<<-EOT
      - hello
      - world
      + hello world
    EOT
`,
		},
		"primitive_single_to_multi_update": {
			diff: computed.Diff{
				Renderer: Primitive("hello world", "hello\nworld", cty.String),
				Action:   plans.Update,
			},
			expected: `
<<-EOT
      - hello world
      + hello
      + world
    EOT
`,
		},
		"sensitive_update": {
			diff: computed.Diff{
				Renderer: Sensitive(computed.Diff{
					Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
					Action:   plans.Update,
				}, true, true),
				Action: plans.Update,
			},
			expected: "(sensitive value)",
		},
		"sensitive_update_replace": {
			diff: computed.Diff{
				Renderer: Sensitive(computed.Diff{
					Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
					Action:   plans.Update,
					Replace:  true,
				}, true, true),
				Action:  plans.Update,
				Replace: true,
			},
			expected: "(sensitive value) # forces replacement",
		},
		"computed_create": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "(known after apply)",
		},
		"computed_update": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{
					Renderer: Primitive(json.Number("0"), nil, cty.Number),
					Action:   plans.Delete,
				}),
				Action: plans.Update,
			},
			expected: "0 -> (known after apply)",
		},
		"computed_update_from_null": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{}),
				Action:   plans.Update,
			},
			expected: "(known after apply)",
		},
		"computed_create_forces_replacement": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{}),
				Action:   plans.Create,
				Replace:  true,
			},
			expected: "(known after apply) # forces replacement",
		},
		"computed_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{
					Renderer: Primitive(json.Number("0"), nil, cty.Number),
					Action:   plans.Delete,
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: "0 -> (known after apply) # forces replacement",
		},
		"object_created": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "{}",
		},
		"object_created_with_attributes": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(nil, json.Number("0"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
{
      + attribute_one = 0
    }
`,
		},
		"object_deleted": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"object_deleted_with_attributes": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("0"), nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - attribute_one = 0
    } -> null
`,
		},
		"nested_object_deleted": {
			diff: computed.Diff{
				Renderer: NestedObject(map[string]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"nested_object_deleted_with_attributes": {
			diff: computed.Diff{
				Renderer: NestedObject(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("0"), nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - attribute_one = 0 -> null
    } -> null
`,
		},
		"object_create_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(nil, json.Number("0"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + attribute_one = 0
    }
`,
		},
		"object_update_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = 0 -> 1
    }
`,
		},
		"object_update_attribute_forces_replacement": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
{ # forces replacement
      ~ attribute_one = 0 -> 1
    }
`,
		},
		"object_delete_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("0"), nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - attribute_one = 0
    }
`,
		},
		"object_ignore_unchanged_attributes": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
						Action:   plans.Update,
					},
					"attribute_two": {
						Renderer: Primitive(json.Number("0"), json.Number("0"), cty.Number),
						Action:   plans.NoOp,
					},
					"attribute_three": {
						Renderer: Primitive(nil, json.Number("1"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
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
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, json.Number("1"), cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + attribute_one = (sensitive value)
    }
`,
		},
		"object_update_sensitive_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = (sensitive value)
    }
`,
		},
		"object_delete_sensitive_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("0"), nil, cty.Number),
							Action:   plans.Delete,
						}, true, false),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - attribute_one = (sensitive value)
    }
`,
		},
		"object_create_computed_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Unknown(computed.Diff{Renderer: nil}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + attribute_one = (known after apply)
    }
`,
		},
		"object_update_computed_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(json.Number("1"), nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ attribute_one = 1 -> (known after apply)
    }
`,
		},
		"object_escapes_attribute_keys": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
						Action:   plans.Update,
					},
					"attribute:two": {
						Renderer: Primitive(json.Number("2"), json.Number("3"), cty.Number),
						Action:   plans.Update,
					},
					"attribute_six": {
						Renderer: Primitive(json.Number("3"), json.Number("4"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "attribute:two" = 2 -> 3
      ~ attribute_one   = 1 -> 2
      ~ attribute_six   = 3 -> 4
    }
`,
		},
		"map_create_empty": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "{}",
		},
		"map_create": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(nil, "new", cty.String),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
{
      + "element_one" = "new"
    }
`,
		},
		"map_delete_empty": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "{} -> null",
		},
		"map_delete": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", nil, cty.String),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - "element_one" = "old"
    } -> null
`,
		},
		"map_create_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(nil, "new", cty.String),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one" = "new"
    }
`,
		},
		"map_update_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", "new", cty.String),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = "old" -> "new"
    }
`,
		},
		"map_delete_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", nil, cty.String),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - "element_one" = "old" -> null
    }
`,
		},
		"map_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive("old", "new", cty.String),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
{ # forces replacement
      ~ "element_one" = "old" -> "new"
    }
`,
		},
		"map_ignore_unchanged_elements": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(nil, "new", cty.String),
						Action:   plans.Create,
					},
					"element_two": {
						Renderer: Primitive("old", "old", cty.String),
						Action:   plans.NoOp,
					},
					"element_three": {
						Renderer: Primitive("old", "new", cty.String),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one"   = "new"
      ~ "element_three" = "old" -> "new"
        # (1 unchanged element hidden)
    }
`,
		},
		"map_create_sensitive_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, json.Number("1"), cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one" = (sensitive value)
    }
`,
		},
		"map_update_sensitive_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = (sensitive value)
    }
`,
		},
		"map_update_sensitive_element_status": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("0"), json.Number("0"), cty.Number),
							Action:   plans.NoOp,
						}, true, false),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change. The value is unchanged.
      ~ "element_one" = (sensitive value)
    }
`,
		},
		"map_delete_sensitive_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("0"), nil, cty.Number),
							Action:   plans.Delete,
						}, true, false),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - "element_one" = (sensitive value) -> null
    }
`,
		},
		"map_create_computed_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Unknown(computed.Diff{}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + "element_one" = (known after apply)
    }
`,
		},
		"map_update_computed_element": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(json.Number("1"), nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = 1 -> (known after apply)
    }
`,
		},
		"map_aligns_key": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
						Action:   plans.Update,
					},
					"element_two": {
						Renderer: Primitive(json.Number("1"), json.Number("3"), cty.Number),
						Action:   plans.Update,
					},
					"element_three": {
						Renderer: Primitive(json.Number("1"), json.Number("4"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one"   = 1 -> 2
      ~ "element_three" = 1 -> 4
      ~ "element_two"   = 1 -> 3
    }
`,
		},
		"nested_map_does_not_align_keys": {
			diff: computed.Diff{
				Renderer: NestedMap(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
						Action:   plans.Update,
					},
					"element_two": {
						Renderer: Primitive(json.Number("1"), json.Number("3"), cty.Number),
						Action:   plans.Update,
					},
					"element_three": {
						Renderer: Primitive(json.Number("1"), json.Number("4"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "element_one" = 1 -> 2
      ~ "element_three" = 1 -> 4
      ~ "element_two" = 1 -> 3
    }
`,
		},
		"list_create_empty": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "[]",
		},
		"list_create": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(nil, json.Number("1"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"list_delete_empty": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "[] -> null",
		},
		"list_delete": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(json.Number("1"), nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
[
      - 1,
    ] -> null
`,
		},
		"list_create_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(nil, json.Number("1"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"list_update_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> 1,
    ]
`,
		},
		"list_replace_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), nil, cty.Number),
						Action:   plans.Delete,
					},
					{
						Renderer: Primitive(nil, json.Number("1"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
      + 1,
    ]
`,
		},
		"list_delete_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
    ]
`,
		},
		"list_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
[ # forces replacement
      ~ 0 -> 1,
    ]
`,
		},
		"list_update_ignores_unchanged": {
			diff: computed.Diff{
				Renderer: NestedList([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), json.Number("0"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("1"), json.Number("1"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("2"), json.Number("5"), cty.Number),
						Action:   plans.Update,
					},
					{
						Renderer: Primitive(json.Number("3"), json.Number("3"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("4"), json.Number("4"), cty.Number),
						Action:   plans.NoOp,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 2 -> 5,
        # (4 unchanged elements hidden)
    ]
`,
		},
		"list_update_ignored_unchanged_with_context": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), json.Number("0"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("1"), json.Number("1"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("2"), json.Number("5"), cty.Number),
						Action:   plans.Update,
					},
					{
						Renderer: Primitive(json.Number("3"), json.Number("3"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("4"), json.Number("4"), cty.Number),
						Action:   plans.NoOp,
					},
				}),
				Action: plans.Update,
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
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, json.Number("1"), cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (sensitive value),
    ]
`,
		},
		"list_delete_sensitive_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("1"), nil, cty.Number),
							Action:   plans.Delete,
						}, true, false),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - (sensitive value),
    ]
`,
		},
		"list_update_sensitive_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ (sensitive value),
    ]
`,
		},
		"list_update_sensitive_element_status": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("1"), json.Number("1"), cty.Number),
							Action:   plans.NoOp,
						}, false, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      # Warning: this attribute value will be marked as sensitive and will not
      # display in UI output after applying this change. The value is unchanged.
      ~ (sensitive value),
    ]
`,
		},
		"list_create_computed_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (known after apply),
    ]
`,
		},
		"list_update_computed_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(json.Number("0"), nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> (known after apply),
    ]
`,
		},
		"set_create_empty": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{}),
				Action:   plans.Create,
			},
			expected: "[]",
		},
		"set_create": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(nil, json.Number("1"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Create,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"set_delete_empty": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{}),
				Action:   plans.Delete,
			},
			expected: "[] -> null",
		},
		"set_delete": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(json.Number("1"), nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Delete,
			},
			expected: `
[
      - 1,
    ] -> null
`,
		},
		"set_create_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(nil, json.Number("1"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + 1,
    ]
`,
		},
		"set_update_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> 1,
    ]
`,
		},
		"set_replace_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), nil, cty.Number),
						Action:   plans.Delete,
					},
					{
						Renderer: Primitive(nil, json.Number("1"), cty.Number),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
      + 1,
    ]
`,
		},
		"set_delete_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), nil, cty.Number),
						Action:   plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - 0,
    ]
`,
		},
		"set_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
						Action:   plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
[ # forces replacement
      ~ 0 -> 1,
    ]
`,
		},
		"nested_set_update_forces_replacement": {
			diff: computed.Diff{
				Renderer: NestedSet([]computed.Diff{
					{
						Renderer: Object(map[string]computed.Diff{
							"attribute_one": {
								Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
								Action:   plans.Update,
							},
						}),
						Action: plans.Update,
					},
				}),
				Action:  plans.Update,
				Replace: true,
			},
			expected: `
[
      ~ { # forces replacement
          ~ attribute_one = 0 -> 1
        },
    ]
`,
		},
		"set_update_ignores_unchanged": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(json.Number("0"), json.Number("0"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("1"), json.Number("1"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("2"), json.Number("5"), cty.Number),
						Action:   plans.Update,
					},
					{
						Renderer: Primitive(json.Number("3"), json.Number("3"), cty.Number),
						Action:   plans.NoOp,
					},
					{
						Renderer: Primitive(json.Number("4"), json.Number("4"), cty.Number),
						Action:   plans.NoOp,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 2 -> 5,
        # (4 unchanged elements hidden)
    ]
`,
		},
		"set_create_sensitive_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, json.Number("1"), cty.Number),
							Action:   plans.Create,
						}, false, true),
						Action: plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (sensitive value),
    ]
`,
		},
		"set_delete_sensitive_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("1"), nil, cty.Number),
							Action:   plans.Delete,
						}, false, true),
						Action: plans.Delete,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      - (sensitive value),
    ]
`,
		},
		"set_update_sensitive_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("0"), json.Number("1"), cty.Number),
							Action:   plans.Update,
						}, true, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ (sensitive value),
    ]
`,
		},
		"set_update_sensitive_element_status": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
							Action:   plans.Update,
						}, false, true),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      # Warning: this attribute value will be marked as sensitive and will not
      # display in UI output after applying this change.
      ~ (sensitive value),
    ]
`,
		},
		"set_create_computed_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{}),
						Action:   plans.Create,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      + (known after apply),
    ]
`,
		},
		"set_update_computed_element": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Unknown(computed.Diff{
							Renderer: Primitive(json.Number("0"), nil, cty.Number),
							Action:   plans.Delete,
						}),
						Action: plans.Update,
					},
				}),
				Action: plans.Update,
			},
			expected: `
[
      ~ 0 -> (known after apply),
    ]
`,
		},
		"create_empty_block": {
			diff: computed.Diff{
				Renderer: Block(nil, Blocks{}),
				Action:   plans.Create,
			},
			expected: "{}",
		},
		"create_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive(nil, "root", cty.String),
						Action:   plans.Create,
					},
					"boolean": {
						Renderer: Primitive(nil, true, cty.Bool),
						Action:   plans.Create,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "one", cty.String),
									Action:   plans.Create,
								},
							}, Blocks{}),
							Action: plans.Create,
						},
						"nested_block_two": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "two", cty.String),
									Action:   plans.Create,
								},
							}, Blocks{}),
							Action: plans.Create,
						},
					},
				}),
				Action: plans.Create,
			},
			expected: `
{
      + boolean = true
      + string  = "root"

      + nested_block {
          + string = "one"
        }

      + nested_block_two {
          + string = "two"
        }
    }`,
		},
		"update_empty_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive(nil, "root", cty.String),
						Action:   plans.Create,
					},
					"boolean": {
						Renderer: Primitive(nil, true, cty.Bool),
						Action:   plans.Create,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block": {

							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "one", cty.String),
									Action:   plans.Create,
								},
							}, Blocks{}),
							Action: plans.Create,
						},
						"nested_block_two": {

							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "two", cty.String),
									Action:   plans.Create,
								},
							}, Blocks{}),
							Action: plans.Create,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      + boolean = true
      + string  = "root"

      + nested_block {
          + string = "one"
        }

      + nested_block_two {
          + string = "two"
        }
    }`,
		},
		"update_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive(nil, "root", cty.String),
						Action:   plans.Create,
					},
					"boolean": {
						Renderer: Primitive(false, true, cty.Bool),
						Action:   plans.Update,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "one", cty.String),
									Action:   plans.NoOp,
								},
							}, Blocks{}),
							Action: plans.NoOp,
						},
						"nested_block_two": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "two", cty.String),
									Action:   plans.Create,
								},
							}, Blocks{}),
							Action: plans.Create,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ boolean = false -> true
      + string  = "root"

      + nested_block_two {
          + string = "two"
        }

        # (1 unchanged block hidden)
    }`,
		},
		"clear_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive("root", nil, cty.String),
						Action:   plans.Delete,
					},
					"boolean": {
						Renderer: Primitive(true, nil, cty.Bool),
						Action:   plans.Delete,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", nil, cty.String),
									Action:   plans.Delete,
								},
							}, Blocks{}),
							Action: plans.Delete,
						},
						"nested_block_two": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", nil, cty.String),
									Action:   plans.Delete,
								},
							}, Blocks{}),
							Action: plans.Delete,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      - boolean = true -> null
      - string  = "root" -> null

      - nested_block {
          - string = "one" -> null
        }

      - nested_block_two {
          - string = "two" -> null
        }
    }`,
		},
		"delete_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"string": {
						Renderer: Primitive("root", nil, cty.String),
						Action:   plans.Delete,
					},
					"boolean": {
						Renderer: Primitive(true, nil, cty.Bool),
						Action:   plans.Delete,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", nil, cty.String),
									Action:   plans.Delete,
								},
							}, Blocks{}),
							Action: plans.Delete,
						},
						"nested_block_two": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", nil, cty.String),
									Action:   plans.Delete,
								},
							}, Blocks{}),
							Action: plans.Delete,
						},
					},
				}),
				Action: plans.Delete,
			},
			expected: `
{
      - boolean = true -> null
      - string  = "root" -> null

      - nested_block {
          - string = "one" -> null
        }

      - nested_block_two {
          - string = "two" -> null
        }
    }`,
		},
		"list_block_update": {
			diff: computed.Diff{
				Renderer: Block(
					nil,
					Blocks{
						ListBlocks: map[string][]computed.Diff{
							"list_blocks": {
								{
									Renderer: Block(map[string]computed.Diff{
										"number": {
											Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
											Action:   plans.Update,
										},
										"string": {
											Renderer: Primitive(nil, "new", cty.String),
											Action:   plans.Create,
										},
									}, Blocks{}),
									Action: plans.Update,
								},
								{
									Renderer: Block(map[string]computed.Diff{
										"number": {
											Renderer: Primitive(json.Number("1"), nil, cty.Number),
											Action:   plans.Delete,
										},
										"string": {
											Renderer: Primitive("old", "new", cty.String),
											Action:   plans.Update,
										},
									}, Blocks{}),
									Action: plans.Update,
								},
							},
						},
					}),
			},
			expected: `
{
      ~ list_blocks {
          ~ number = 1 -> 2
          + string = "new"
        }
      ~ list_blocks {
          - number = 1 -> null
          ~ string = "old" -> "new"
        }
    }`,
		},
		"set_block_update": {
			diff: computed.Diff{
				Renderer: Block(
					nil,
					Blocks{
						SetBlocks: map[string][]computed.Diff{
							"set_blocks": {
								{
									Renderer: Block(map[string]computed.Diff{
										"number": {
											Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
											Action:   plans.Update,
										},
										"string": {
											Renderer: Primitive(nil, "new", cty.String),
											Action:   plans.Create,
										},
									}, Blocks{}),
									Action: plans.Update,
								},
								{
									Renderer: Block(map[string]computed.Diff{
										"number": {
											Renderer: Primitive(json.Number("1"), nil, cty.Number),
											Action:   plans.Delete,
										},
										"string": {
											Renderer: Primitive("old", "new", cty.String),
											Action:   plans.Update,
										},
									}, Blocks{}),
									Action: plans.Update,
								},
							},
						},
					}),
			},
			expected: `
{
      ~ set_blocks {
          ~ number = 1 -> 2
          + string = "new"
        }
      ~ set_blocks {
          - number = 1 -> null
          ~ string = "old" -> "new"
        }
    }`,
		},
		"map_block_update": {
			diff: computed.Diff{
				Renderer: Block(
					nil,
					Blocks{
						MapBlocks: map[string]map[string]computed.Diff{
							"list_blocks": {
								"key_one": {
									Renderer: Block(map[string]computed.Diff{
										"number": {
											Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
											Action:   plans.Update,
										},
										"string": {
											Renderer: Primitive(nil, "new", cty.String),
											Action:   plans.Create,
										},
									}, Blocks{}),
									Action: plans.Update,
								},
								"key:two": {
									Renderer: Block(map[string]computed.Diff{
										"number": {
											Renderer: Primitive(json.Number("1"), nil, cty.Number),
											Action:   plans.Delete,
										},
										"string": {
											Renderer: Primitive("old", "new", cty.String),
											Action:   plans.Update,
										},
									}, Blocks{}),
									Action: plans.Update,
								},
							},
						},
					}),
			},
			expected: `
{
      ~ list_blocks "key:two" {
          - number = 1 -> null
          ~ string = "old" -> "new"
        }
      ~ list_blocks "key_one" {
          ~ number = 1 -> 2
          + string = "new"
        }
    }
`,
		},
		"sensitive_block": {
			diff: computed.Diff{
				Renderer: SensitiveBlock(computed.Diff{
					Renderer: Block(nil, Blocks{}),
					Action:   plans.NoOp,
				}, true, true),
				Action: plans.Update,
			},
			expected: `
{
      # At least one attribute in this block is (or was) sensitive,
      # so its contents will not be displayed.
    }
`,
		},
		"delete_empty_block": {
			diff: computed.Diff{
				Renderer: Block(nil, Blocks{}),
				Action:   plans.Delete,
			},
			expected: "{}",
		},
		"block_escapes_keys": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(json.Number("1"), json.Number("2"), cty.Number),
						Action:   plans.Update,
					},
					"attribute:two": {
						Renderer: Primitive(json.Number("2"), json.Number("3"), cty.Number),
						Action:   plans.Update,
					},
					"attribute_six": {
						Renderer: Primitive(json.Number("3"), json.Number("4"), cty.Number),
						Action:   plans.Update,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block:one": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", "four", cty.String),
									Action:   plans.Update,
								},
							}, Blocks{}),
							Action: plans.Update,
						},
						"nested_block_two": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", "three", cty.String),
									Action:   plans.Update,
								},
							}, Blocks{}),
							Action: plans.Update,
						},
					},
				}),
				Action: plans.Update,
			},
			expected: `
{
      ~ "attribute:two" = 2 -> 3
      ~ attribute_one   = 1 -> 2
      ~ attribute_six   = 3 -> 4

      ~ "nested_block:one" {
          ~ string = "one" -> "four"
        }

      ~ nested_block_two {
          ~ string = "two" -> "three"
        }
    }`,
		},
		"block_always_includes_important_attributes": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"id": {
						Renderer: Primitive("root", "root", cty.String),
						Action:   plans.NoOp,
					},
					"boolean": {
						Renderer: Primitive(false, false, cty.Bool),
						Action:   plans.NoOp,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("one", "one", cty.String),
									Action:   plans.NoOp,
								},
							}, Blocks{}),
							Action: plans.NoOp,
						},
						"nested_block_two": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive("two", "two", cty.String),
									Action:   plans.NoOp,
								},
							}, Blocks{}),
							Action: plans.NoOp,
						},
					},
				}),
				Action: plans.NoOp,
			},
			expected: `
{
        id      = "root"
        # (1 unchanged attribute hidden)

        # (2 unchanged blocks hidden)
    }`,
		},
		"output_map_to_list": {
			diff: computed.Diff{
				Renderer: TypeChange(computed.Diff{
					Renderer: Map(map[string]computed.Diff{
						"element_one": {
							Renderer: Primitive(json.Number("0"), nil, cty.Number),
							Action:   plans.Delete,
						},
						"element_two": {
							Renderer: Primitive(json.Number("1"), nil, cty.Number),
							Action:   plans.Delete,
						},
					}),
					Action: plans.Delete,
				}, computed.Diff{
					Renderer: List([]computed.Diff{
						{
							Renderer: Primitive(nil, json.Number("0"), cty.Number),
							Action:   plans.Create,
						},
						{
							Renderer: Primitive(nil, json.Number("1"), cty.Number),
							Action:   plans.Create,
						},
					}),
					Action: plans.Create,
				}),
			},
			expected: `
{
      - "element_one" = 0
      - "element_two" = 1
    } -> [
      + 0,
      + 1,
    ]
`,
		},
		"json_string_no_symbols": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key\":\"value\"}", "{\"key\":\"value\"}", cty.String),
				Action:   plans.NoOp,
			},
			opts: computed.RenderHumanOpts{
				HideDiffActionSymbols: true,
				ShowUnchangedChildren: true,
			},
			expected: `
jsonencode(
    {
        key = "value"
    }
)
`,
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			opts := tc.opts.Clone()
			opts.Colorize = &colorize

			expected := strings.TrimSpace(tc.expected)
			actual := tc.diff.RenderHuman(0, opts)
			if diff := cmp.Diff(expected, actual); len(diff) > 0 {
				t.Fatalf("\nexpected:\n%s\nactual:\n%s\ndiff:\n%s\n", expected, actual, diff)
			}
		})
	}
}
