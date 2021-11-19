package format

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/mitchellh/colorstring"
	"github.com/zclconf/go-cty/cty"
)

func TestResourceChange_primitiveTypes(t *testing.T) {
	testCases := map[string]testCase{
		"creation": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + id = (known after apply)
    }
`,
		},
		"creation (null string)": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal("null"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"string": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + string = "null"
    }
`,
		},
		"creation (null string with extra whitespace)": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"string": cty.StringVal("null "),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"string": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + string = "null "
    }
`,
		},
		"deletion": {
			Action: plans.Delete,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
			}),
			After: cty.NullVal(cty.EmptyObject),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be destroyed
  - resource "test_instance" "example" {
      - id = "i-02ae66f368e8518a9" -> null
    }
`,
		},
		"deletion of deposed object": {
			Action:     plans.Delete,
			Mode:       addrs.ManagedResourceMode,
			DeposedKey: states.DeposedKey("byebye"),
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
			}),
			After: cty.NullVal(cty.EmptyObject),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Computed: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example (deposed object byebye) will be destroyed
  # (left over from a partially-failed replacement of this instance)
  - resource "test_instance" "example" {
      - id = "i-02ae66f368e8518a9" -> null
    }
`,
		},
		"deletion (empty string)": {
			Action: plans.Delete,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":                 cty.StringVal("i-02ae66f368e8518a9"),
				"intentionally_long": cty.StringVal(""),
			}),
			After: cty.NullVal(cty.EmptyObject),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":                 {Type: cty.String, Computed: true},
					"intentionally_long": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be destroyed
  - resource "test_instance" "example" {
      - id = "i-02ae66f368e8518a9" -> null
    }
`,
		},
		"string in-place update": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"
    }
`,
		},
		"string force-new update": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "ami"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER" # forces replacement
        id  = "i-02ae66f368e8518a9"
    }
`,
		},
		"string in-place update (null values)": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.StringVal("i-02ae66f368e8518a9"),
				"ami":       cty.StringVal("ami-BEFORE"),
				"unchanged": cty.NullVal(cty.String),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.StringVal("i-02ae66f368e8518a9"),
				"ami":       cty.StringVal("ami-AFTER"),
				"unchanged": cty.NullVal(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"unchanged": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"
    }
`,
		},
		"in-place update of multi-line string field": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"more_lines": cty.StringVal(`original
long
multi-line
string
field
`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"more_lines": cty.StringVal(`original
extremely long
multi-line
string
field
`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"more_lines": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ more_lines = <<-EOT
            original
          - long
          + extremely long
            multi-line
            string
            field
        EOT
    }
`,
		},
		"addition of multi-line string field": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"more_lines": cty.NullVal(cty.String),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"more_lines": cty.StringVal(`original
new line
`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"more_lines": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + more_lines = <<-EOT
            original
            new line
        EOT
    }
`,
		},
		"force-new update of multi-line string field": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"more_lines": cty.StringVal(`original
`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"more_lines": cty.StringVal(`original
new line
`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"more_lines": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "more_lines"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ more_lines = <<-EOT # forces replacement
            original
          + new line
        EOT
    }
`,
		},

		// Sensitive

		"creation with sensitive field": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":       cty.UnknownVal(cty.String),
				"password": cty.StringVal("top-secret"),
				"conn_info": cty.ObjectVal(map[string]cty.Value{
					"user":     cty.StringVal("not-secret"),
					"password": cty.StringVal("top-secret"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":       {Type: cty.String, Computed: true},
					"password": {Type: cty.String, Optional: true, Sensitive: true},
					"conn_info": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"user":     {Type: cty.String, Optional: true},
								"password": {Type: cty.String, Optional: true, Sensitive: true},
							},
						},
					},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + conn_info = {
          + password = (sensitive value)
          + user     = "not-secret"
        }
      + id        = (known after apply)
      + password  = (sensitive value)
    }
`,
		},
		"update with equal sensitive field": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":       cty.StringVal("blah"),
				"str":      cty.StringVal("before"),
				"password": cty.StringVal("top-secret"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":       cty.UnknownVal(cty.String),
				"str":      cty.StringVal("after"),
				"password": cty.StringVal("top-secret"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":       {Type: cty.String, Computed: true},
					"str":      {Type: cty.String, Optional: true},
					"password": {Type: cty.String, Optional: true, Sensitive: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id       = "blah" -> (known after apply)
      ~ str      = "before" -> "after"
        # (1 unchanged attribute hidden)
    }
`,
		},

		// tainted objects
		"replace tainted resource": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseTainted,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-AFTER"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "ami"},
			}),
			ExpectedOutput: `  # test_instance.example is tainted, so must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER" # forces replacement
      ~ id  = "i-02ae66f368e8518a9" -> (known after apply)
    }
`,
		},
		"force replacement with empty before value": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("name"),
				"forced": cty.NullVal(cty.String),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("name"),
				"forced": cty.StringVal("example"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"name":   {Type: cty.String, Optional: true},
					"forced": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "forced"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      + forced = "example" # forces replacement
        name   = "name"
    }
`,
		},
		"force replacement with empty before value legacy": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("name"),
				"forced": cty.StringVal(""),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"name":   cty.StringVal("name"),
				"forced": cty.StringVal("example"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"name":   {Type: cty.String, Optional: true},
					"forced": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "forced"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      + forced = "example" # forces replacement
        name   = "name"
    }
`,
		},
		"show all identifying attributes even if unchanged": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":   cty.StringVal("i-02ae66f368e8518a9"),
				"ami":  cty.StringVal("ami-BEFORE"),
				"bar":  cty.StringVal("bar"),
				"foo":  cty.StringVal("foo"),
				"name": cty.StringVal("alice"),
				"tags": cty.MapVal(map[string]cty.Value{
					"name": cty.StringVal("bob"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":   cty.StringVal("i-02ae66f368e8518a9"),
				"ami":  cty.StringVal("ami-AFTER"),
				"bar":  cty.StringVal("bar"),
				"foo":  cty.StringVal("foo"),
				"name": cty.StringVal("alice"),
				"tags": cty.MapVal(map[string]cty.Value{
					"name": cty.StringVal("bob"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":   {Type: cty.String, Optional: true, Computed: true},
					"ami":  {Type: cty.String, Optional: true},
					"bar":  {Type: cty.String, Optional: true},
					"foo":  {Type: cty.String, Optional: true},
					"name": {Type: cty.String, Optional: true},
					"tags": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami  = "ami-BEFORE" -> "ami-AFTER"
        id   = "i-02ae66f368e8518a9"
        name = "alice"
        tags = {
            "name" = "bob"
        }
        # (2 unchanged attributes hidden)
    }
`,
		},
	}

	runTestCases(t, testCases)
}

func TestResourceChange_JSON(t *testing.T) {
	testCases := map[string]testCase{
		"creation": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{
					"str": "value",
					"list":["a","b", 234, true],
					"obj": {"key": "val"}
				}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + id         = (known after apply)
      + json_field = jsonencode(
            {
              + list = [
                  + "a",
                  + "b",
                  + 234,
                  + true,
                ]
              + obj  = {
                  + key = "val"
                }
              + str  = "value"
            }
        )
    }
`,
		},
		"in-place update of object": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": "value","ccc": 5}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"aaa": "value", "bbb": "new_value"}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              + bbb = "new_value"
              - ccc = 5 -> null
                # (1 unchanged element hidden)
            }
        )
    }
`,
		},
		"in-place update (from empty tuple)": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": []}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"aaa": ["value"]}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              ~ aaa = [
                  + "value",
                ]
            }
        )
    }
`,
		},
		"in-place update (to empty tuple)": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": ["value"]}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"aaa": []}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              ~ aaa = [
                  - "value",
                ]
            }
        )
    }
`,
		},
		"in-place update (tuple of different types)": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": [42, {"foo":"bar"}, "value"]}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"aaa": [42, {"foo":"baz"}, "value"]}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              ~ aaa = [
                    42,
                  ~ {
                      ~ foo = "bar" -> "baz"
                    },
                    "value",
                ]
            }
        )
    }
`,
		},
		"force-new update": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": "value"}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"aaa": "value", "bbb": "new_value"}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "json_field"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              + bbb = "new_value"
                # (1 unchanged element hidden)
            } # forces replacement
        )
    }
`,
		},
		"in-place update (whitespace change)": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": "value", "bbb": "another"}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"aaa":"value",
					"bbb":"another"}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode( # whitespace changes
            {
                aaa = "value"
                bbb = "another"
            }
        )
    }
`,
		},
		"force-new update (whitespace change)": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": "value", "bbb": "another"}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"aaa":"value",
					"bbb":"another"}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "json_field"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode( # whitespace changes force replacement
            {
                aaa = "value"
                bbb = "another"
            }
        )
    }
`,
		},
		"creation (empty)": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + id         = (known after apply)
      + json_field = jsonencode({})
    }
`,
		},
		"JSON list item removal": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`["first","second","third"]`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`["first","second"]`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ [
                # (1 unchanged element hidden)
                "second",
              - "third",
            ]
        )
    }
`,
		},
		"JSON list item addition": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`["first","second"]`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`["first","second","third"]`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ [
                # (1 unchanged element hidden)
                "second",
              + "third",
            ]
        )
    }
`,
		},
		"JSON list object addition": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"first":"111"}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"first":"111","second":"222"}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              + second = "222"
                # (1 unchanged element hidden)
            }
        )
    }
`,
		},
		"JSON object with nested list": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{
		  "Statement": ["first"]
		}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{
		  "Statement": ["first", "second"]
		}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              ~ Statement = [
                    "first",
                  + "second",
                ]
            }
        )
    }
`,
		},
		"JSON list of objects - adding item": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`[{"one": "111"}]`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`[{"one": "111"}, {"two": "222"}]`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ [
                {
                    one = "111"
                },
              + {
                  + two = "222"
                },
            ]
        )
    }
`,
		},
		"JSON list of objects - removing item": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`[{"one": "111"}, {"two": "222"}, {"three": "333"}]`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`[{"one": "111"}, {"three": "333"}]`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ [
                {
                    one = "111"
                },
              - {
                  - two = "222"
                },
                {
                    three = "333"
                },
            ]
        )
    }
`,
		},
		"JSON object with list of objects": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"parent":[{"one": "111"}]}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"parent":[{"one": "111"}, {"two": "222"}]}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              ~ parent = [
                    {
                        one = "111"
                    },
                  + {
                      + two = "222"
                    },
                ]
            }
        )
    }
`,
		},
		"JSON object double nested lists": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"parent":[{"another_list": ["111"]}]}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`{"parent":[{"another_list": ["111", "222"]}]}`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              ~ parent = [
                  ~ {
                      ~ another_list = [
                            "111",
                          + "222",
                        ]
                    },
                ]
            }
        )
    }
`,
		},
		"in-place update from object to tuple": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"json_field": cty.StringVal(`{"aaa": [42, {"foo":"bar"}, "value"]}`),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"json_field": cty.StringVal(`["aaa", 42, "something"]`),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"json_field": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
              - aaa = [
                  - 42,
                  - {
                      - foo = "bar"
                    },
                  - "value",
                ]
            } -> [
              + "aaa",
              + 42,
              + "something",
            ]
        )
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_primitiveList(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"ami":        cty.StringVal("ami-STATIC"),
				"list_field": cty.NullVal(cty.List(cty.String)),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("new-element"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + list_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - first addition": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"ami":        cty.StringVal("ami-STATIC"),
				"list_field": cty.ListValEmpty(cty.String),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("new-element"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("dddd"),
					cty.StringVal("eeee"),
					cty.StringVal("ffff"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
					cty.StringVal("dddd"),
					cty.StringVal("eeee"),
					cty.StringVal("ffff"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
            # (1 unchanged element hidden)
            "bbbb",
          + "cccc",
            "dddd",
            # (2 unchanged elements hidden)
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"force-new update - insertion": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "list_field"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [ # forces replacement
            "aaaa",
          + "bbbb",
            "cccc",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - deletion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
					cty.StringVal("dddd"),
					cty.StringVal("eeee"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("bbbb"),
					cty.StringVal("dddd"),
					cty.StringVal("eeee"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          - "aaaa",
            "bbbb",
          - "cccc",
            "dddd",
            # (1 unchanged element hidden)
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"creation - empty list": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"ami":        cty.StringVal("ami-STATIC"),
				"list_field": cty.ListValEmpty(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + ami        = "ami-STATIC"
      + id         = (known after apply)
      + list_field = []
    }
`,
		},
		"in-place update - full to empty": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"ami":        cty.StringVal("ami-STATIC"),
				"list_field": cty.ListValEmpty(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          - "aaaa",
          - "bbbb",
          - "cccc",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - null to empty": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.StringVal("i-02ae66f368e8518a9"),
				"ami":        cty.StringVal("ami-STATIC"),
				"list_field": cty.NullVal(cty.List(cty.String)),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":         cty.UnknownVal(cty.String),
				"ami":        cty.StringVal("ami-STATIC"),
				"list_field": cty.ListValEmpty(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + list_field = []
        # (1 unchanged attribute hidden)
    }
`,
		},
		"update to unknown element": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.UnknownVal(cty.String),
					cty.StringVal("cccc"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
            "aaaa",
          - "bbbb",
          + (known after apply),
            "cccc",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"update - two new unknown elements": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
					cty.StringVal("dddd"),
					cty.StringVal("eeee"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.UnknownVal(cty.String),
					cty.UnknownVal(cty.String),
					cty.StringVal("cccc"),
					cty.StringVal("dddd"),
					cty.StringVal("eeee"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
            "aaaa",
          - "bbbb",
          + (known after apply),
          + (known after apply),
            "cccc",
            # (2 unchanged elements hidden)
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_primitiveTuple(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"tuple_field": cty.TupleVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("dddd"),
					cty.StringVal("eeee"),
					cty.StringVal("ffff"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"tuple_field": cty.TupleVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
					cty.StringVal("eeee"),
					cty.StringVal("ffff"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":          {Type: cty.String, Required: true},
					"tuple_field": {Type: cty.Tuple([]cty.Type{cty.String, cty.String, cty.String, cty.String, cty.String}), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        id          = "i-02ae66f368e8518a9"
      ~ tuple_field = [
            # (1 unchanged element hidden)
            "bbbb",
          - "dddd",
          + "cccc",
            "eeee",
            # (1 unchanged element hidden)
        ]
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_primitiveSet(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.StringVal("i-02ae66f368e8518a9"),
				"ami":       cty.StringVal("ami-STATIC"),
				"set_field": cty.NullVal(cty.Set(cty.String)),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("new-element"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + set_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - first insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.StringVal("i-02ae66f368e8518a9"),
				"ami":       cty.StringVal("ami-STATIC"),
				"set_field": cty.SetValEmpty(cty.String),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("new-element"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          + "bbbb",
            # (2 unchanged elements hidden)
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"force-new update - insertion": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "set_field"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [ # forces replacement
          + "bbbb",
            # (2 unchanged elements hidden)
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - deletion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
					cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("bbbb"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          - "aaaa",
          - "cccc",
            # (1 unchanged element hidden)
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"creation - empty set": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.UnknownVal(cty.String),
				"ami":       cty.StringVal("ami-STATIC"),
				"set_field": cty.SetValEmpty(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + ami       = "ami-STATIC"
      + id        = (known after apply)
      + set_field = []
    }
`,
		},
		"in-place update - full to empty set": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.UnknownVal(cty.String),
				"ami":       cty.StringVal("ami-STATIC"),
				"set_field": cty.SetValEmpty(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          - "aaaa",
          - "bbbb",
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - null to empty set": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.StringVal("i-02ae66f368e8518a9"),
				"ami":       cty.StringVal("ami-STATIC"),
				"set_field": cty.NullVal(cty.Set(cty.String)),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.UnknownVal(cty.String),
				"ami":       cty.StringVal("ami-STATIC"),
				"set_field": cty.SetValEmpty(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + set_field = []
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update to unknown": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.UnknownVal(cty.String),
				"ami":       cty.StringVal("ami-STATIC"),
				"set_field": cty.UnknownVal(cty.Set(cty.String)),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          - "aaaa",
          - "bbbb",
        ] -> (known after apply)
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update to unknown element": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.StringVal("bbbb"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"set_field": cty.SetVal([]cty.Value{
					cty.StringVal("aaaa"),
					cty.UnknownVal(cty.String),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"set_field": {Type: cty.Set(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          - "bbbb",
          ~ (known after apply),
            # (1 unchanged element hidden)
        ]
        # (1 unchanged attribute hidden)
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_map(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.StringVal("i-02ae66f368e8518a9"),
				"ami":       cty.StringVal("ami-STATIC"),
				"map_field": cty.NullVal(cty.Map(cty.String)),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"new-key": cty.StringVal("new-element"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"map_field": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + map_field = {
          + "new-key" = "new-element"
        }
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - first insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.StringVal("i-02ae66f368e8518a9"),
				"ami":       cty.StringVal("ami-STATIC"),
				"map_field": cty.MapValEmpty(cty.String),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"new-key": cty.StringVal("new-element"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"map_field": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
          + "new-key" = "new-element"
        }
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("aaaa"),
					"c": cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("aaaa"),
					"b": cty.StringVal("bbbb"),
					"c": cty.StringVal("cccc"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"map_field": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
          + "b" = "bbbb"
            # (2 unchanged elements hidden)
        }
        # (1 unchanged attribute hidden)
    }
`,
		},
		"force-new update - insertion": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("aaaa"),
					"c": cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("aaaa"),
					"b": cty.StringVal("bbbb"),
					"c": cty.StringVal("cccc"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"map_field": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "map_field"},
			}),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = { # forces replacement
          + "b" = "bbbb"
            # (2 unchanged elements hidden)
        }
        # (1 unchanged attribute hidden)
    }
`,
		},
		"in-place update - deletion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("aaaa"),
					"b": cty.StringVal("bbbb"),
					"c": cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"b": cty.StringVal("bbbb"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"map_field": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
          - "a" = "aaaa" -> null
          - "c" = "cccc" -> null
            # (1 unchanged element hidden)
        }
        # (1 unchanged attribute hidden)
    }
`,
		},
		"creation - empty": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":        cty.UnknownVal(cty.String),
				"ami":       cty.StringVal("ami-STATIC"),
				"map_field": cty.MapValEmpty(cty.String),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"map_field": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + ami       = "ami-STATIC"
      + id        = (known after apply)
      + map_field = {}
    }
`,
		},
		"update to unknown element": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("aaaa"),
					"b": cty.StringVal("bbbb"),
					"c": cty.StringVal("cccc"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"map_field": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("aaaa"),
					"b": cty.UnknownVal(cty.String),
					"c": cty.StringVal("cccc"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":        {Type: cty.String, Optional: true, Computed: true},
					"ami":       {Type: cty.String, Optional: true},
					"map_field": {Type: cty.Map(cty.String), Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
          ~ "b" = "bbbb" -> (known after apply)
            # (2 unchanged elements hidden)
        }
        # (1 unchanged attribute hidden)
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_nestedList(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update - equal": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
        id    = "i-02ae66f368e8518a9"
        # (1 unchanged attribute hidden)

        # (1 unchanged block hidden)
    }
`,
		},
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
				"disks": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
					"mount_point": cty.StringVal("/var/diska"),
					"size":        cty.StringVal("50GB"),
				})}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.NullVal(cty.String),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          + {
              + mount_point = "/var/diska"
              + size        = "50GB"
            },
        ]
        id    = "i-02ae66f368e8518a9"

      + root_block_device {}
    }
`,
		},
		"in-place update - first insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
				"disks": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.NullVal(cty.String),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          + {
              + mount_point = "/var/diska"
            },
        ]
        id    = "i-02ae66f368e8518a9"

      + root_block_device {
          + volume_type = "gp2"
        }
    }
`,
		},
		"in-place update - insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.NullVal(cty.String),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          ~ {
              + size        = "50GB"
                # (1 unchanged attribute hidden)
            },
            # (1 unchanged element hidden)
        ]
        id    = "i-02ae66f368e8518a9"

      ~ root_block_device {
          + new_field   = "new_value"
            # (1 unchanged attribute hidden)
        }
    }
`,
		},
		"force-new update (inside blocks)": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("different"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(
				cty.Path{
					cty.GetAttrStep{Name: "root_block_device"},
					cty.IndexStep{Key: cty.NumberIntVal(0)},
					cty.GetAttrStep{Name: "volume_type"},
				},
				cty.Path{
					cty.GetAttrStep{Name: "disks"},
					cty.IndexStep{Key: cty.NumberIntVal(0)},
					cty.GetAttrStep{Name: "mount_point"},
				},
			),
			Schema: testSchema(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          ~ {
              ~ mount_point = "/var/diska" -> "/var/diskb" # forces replacement
                # (1 unchanged attribute hidden)
            },
        ]
        id    = "i-02ae66f368e8518a9"

      ~ root_block_device {
          ~ volume_type = "gp2" -> "different" # forces replacement
        }
    }
`,
		},
		"force-new update (whole block)": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("different"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(
				cty.Path{cty.GetAttrStep{Name: "root_block_device"}},
				cty.Path{cty.GetAttrStep{Name: "disks"}},
			),
			Schema: testSchema(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [ # forces replacement
          ~ {
              ~ mount_point = "/var/diska" -> "/var/diskb"
                # (1 unchanged attribute hidden)
            },
        ]
        id    = "i-02ae66f368e8518a9"

      ~ root_block_device { # forces replacement
          ~ volume_type = "gp2" -> "different"
        }
    }
`,
		},
		"in-place update - deletion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
				"root_block_device": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          - {
              - mount_point = "/var/diska" -> null
              - size        = "50GB" -> null
            },
        ]
        id    = "i-02ae66f368e8518a9"

      - root_block_device {
          - volume_type = "gp2" -> null
        }
    }
`,
		},
		"with dynamically-typed attribute": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"block": cty.EmptyTupleVal,
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"block": cty.TupleVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.StringVal("foo"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"attr": cty.True,
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"attr": {Type: cty.DynamicPseudoType, Optional: true},
							},
						},
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      + block {
          + attr = "foo"
        }
      + block {
          + attr = true
        }
    }
`,
		},
		"in-place sequence update - deletion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{"attr": cty.StringVal("x")}),
					cty.ObjectVal(map[string]cty.Value{"attr": cty.StringVal("y")}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{"attr": cty.StringVal("y")}),
					cty.ObjectVal(map[string]cty.Value{"attr": cty.StringVal("z")}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"list": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"attr": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ list {
          ~ attr = "x" -> "y"
        }
      ~ list {
          ~ attr = "y" -> "z"
        }
    }
`,
		},
		"in-place update - unknown": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				}))),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          - {
              - mount_point = "/var/diska" -> null
              - size        = "50GB" -> null
            },
        ] -> (known after apply)
        id    = "i-02ae66f368e8518a9"

        # (1 unchanged block hidden)
    }
`,
		},
		"in-place update - modification": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("50GB"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskc"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("75GB"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskc"),
						"size":        cty.StringVal("25GB"),
					}),
				}),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingList),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          ~ {
              ~ size        = "50GB" -> "75GB"
                # (1 unchanged attribute hidden)
            },
          ~ {
              ~ size        = "50GB" -> "25GB"
                # (1 unchanged attribute hidden)
            },
            # (1 unchanged element hidden)
        ]
        id    = "i-02ae66f368e8518a9"

        # (1 unchanged block hidden)
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_nestedSet(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
				"root_block_device": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.NullVal(cty.String),
					}),
				}),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingSet),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          + {
              + mount_point = "/var/diska"
            },
        ]
        id    = "i-02ae66f368e8518a9"

      + root_block_device {
          + volume_type = "gp2"
        }
    }
`,
		},
		"in-place update - insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.NullVal(cty.String),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("100GB"),
					}),
				}),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.NullVal(cty.String),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("100GB"),
					}),
				}),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingSet),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          + {
              + mount_point = "/var/diska"
              + size        = "50GB"
            },
          - {
              - mount_point = "/var/diska" -> null
            },
            # (1 unchanged element hidden)
        ]
        id    = "i-02ae66f368e8518a9"

      + root_block_device {
          + new_field   = "new_value"
          + volume_type = "gp2"
        }
      - root_block_device {
          - volume_type = "gp2" -> null
        }
    }
`,
		},
		"force-new update (whole block)": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("different"),
					}),
				}),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diskb"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(
				cty.Path{cty.GetAttrStep{Name: "root_block_device"}},
				cty.Path{cty.GetAttrStep{Name: "disks"}},
			),
			Schema: testSchema(configschema.NestingSet),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          - { # forces replacement
              - mount_point = "/var/diska" -> null
              - size        = "50GB" -> null
            },
          + { # forces replacement
              + mount_point = "/var/diskb"
              + size        = "50GB"
            },
        ]
        id    = "i-02ae66f368e8518a9"

      + root_block_device { # forces replacement
          + volume_type = "different"
        }
      - root_block_device { # forces replacement
          - volume_type = "gp2" -> null
        }
    }
`,
		},
		"in-place update - deletion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
					"new_field":   cty.String,
				})),
				"disks": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingSet),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          - {
              - mount_point = "/var/diska" -> null
              - size        = "50GB" -> null
            },
        ]
        id    = "i-02ae66f368e8518a9"

      - root_block_device {
          - new_field   = "new_value" -> null
          - volume_type = "gp2" -> null
        }
    }
`,
		},
		"in-place update - empty nested sets": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				}))),
				"root_block_device": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
				"root_block_device": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingSet),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      + disks = [
        ]
        id    = "i-02ae66f368e8518a9"
    }
`,
		},
		"in-place update - null insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				}))),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.NullVal(cty.String),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingSet),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      + disks = [
          + {
              + mount_point = "/var/diska"
              + size        = "50GB"
            },
        ]
        id    = "i-02ae66f368e8518a9"

      + root_block_device {
          + new_field   = "new_value"
          + volume_type = "gp2"
        }
      - root_block_device {
          - volume_type = "gp2" -> null
        }
    }
`,
		},
		"in-place update - unknown": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.UnknownVal(cty.Set(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				}))),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingSet),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = [
          - {
              - mount_point = "/var/diska" -> null
              - size        = "50GB" -> null
            },
        ] -> (known after apply)
        id    = "i-02ae66f368e8518a9"

        # (1 unchanged block hidden)
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_nestedMap(t *testing.T) {
	testCases := map[string]testCase{
		"creation from null": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.NullVal(cty.String),
				"ami": cty.NullVal(cty.String),
				"disks": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				}))),
				"root_block_device": cty.NullVal(cty.Map(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				}))),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.NullVal(cty.String),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      + ami   = "ami-AFTER"
      + disks = {
          + "disk_a" = {
              + mount_point = "/var/diska"
            },
        }
      + id    = "i-02ae66f368e8518a9"

      + root_block_device "a" {
          + volume_type = "gp2"
        }
    }
`,
		},
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
				"root_block_device": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.NullVal(cty.String),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchema(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = {
          + "disk_a" = {
              + mount_point = "/var/diska"
            },
        }
        id    = "i-02ae66f368e8518a9"

      + root_block_device "a" {
          + volume_type = "gp2"
        }
    }
`,
		},
		"in-place update - change attr": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.NullVal(cty.String),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.NullVal(cty.String),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = {
          ~ "disk_a" = {
              + size        = "50GB"
                # (1 unchanged attribute hidden)
            },
        }
        id    = "i-02ae66f368e8518a9"

      ~ root_block_device "a" {
          + new_field   = "new_value"
            # (1 unchanged attribute hidden)
        }
    }
`,
		},
		"in-place update - insertion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.NullVal(cty.String),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
					"disk_2": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/disk2"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.NullVal(cty.String),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = {
          + "disk_2" = {
              + mount_point = "/var/disk2"
              + size        = "50GB"
            },
            # (1 unchanged element hidden)
        }
        id    = "i-02ae66f368e8518a9"

      + root_block_device "b" {
          + new_field   = "new_value"
          + volume_type = "gp2"
        }
        # (1 unchanged block hidden)
    }
`,
		},
		"force-new update (whole block)": {
			Action:       plans.DeleteThenCreate,
			ActionReason: plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:         addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("standard"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("100GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("different"),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("standard"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "root_block_device"},
				cty.IndexStep{Key: cty.StringVal("a")},
			},
				cty.Path{cty.GetAttrStep{Name: "disks"}},
			),
			Schema: testSchema(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = {
          ~ "disk_a" = { # forces replacement
              ~ size        = "50GB" -> "100GB"
                # (1 unchanged attribute hidden)
            },
        }
        id    = "i-02ae66f368e8518a9"

      ~ root_block_device "a" { # forces replacement
          ~ volume_type = "gp2" -> "different"
        }
        # (1 unchanged block hidden)
    }
`,
		},
		"in-place update - deletion": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
				"root_block_device": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
					"new_field":   cty.String,
				})),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = {
          - "disk_a" = {
              - mount_point = "/var/diska" -> null
              - size        = "50GB" -> null
            },
        }
        id    = "i-02ae66f368e8518a9"

      - root_block_device "a" {
          - new_field   = "new_value" -> null
          - volume_type = "gp2" -> null
        }
    }
`,
		},
		"in-place update - unknown": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.UnknownVal(cty.Map(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				}))),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = {
          - "disk_a" = {
              - mount_point = "/var/diska" -> null
              - size        = "50GB" -> null
            },
        } -> (known after apply)
        id    = "i-02ae66f368e8518a9"

        # (1 unchanged block hidden)
    }
`,
		},
		"in-place update - insertion sensitive": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"disks": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"mount_point": cty.String,
					"size":        cty.String,
				})),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"disks": cty.MapVal(map[string]cty.Value{
					"disk_a": cty.ObjectVal(map[string]cty.Value{
						"mount_point": cty.StringVal("/var/diska"),
						"size":        cty.StringVal("50GB"),
					}),
				}),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			AfterValMarks: []cty.PathValueMarks{
				{
					Path: cty.Path{cty.GetAttrStep{Name: "disks"},
						cty.IndexStep{Key: cty.StringVal("disk_a")},
						cty.GetAttrStep{Name: "mount_point"},
					},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Schema:          testSchemaPlus(configschema.NestingMap),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami   = "ami-BEFORE" -> "ami-AFTER"
      ~ disks = {
          + "disk_a" = {
              + mount_point = (sensitive)
              + size        = "50GB"
            },
        }
        id    = "i-02ae66f368e8518a9"

        # (1 unchanged block hidden)
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_actionReason(t *testing.T) {
	emptySchema := &configschema.Block{}
	nullVal := cty.NullVal(cty.EmptyObject)
	emptyVal := cty.EmptyObjectVal

	testCases := map[string]testCase{
		"delete for no particular reason": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceChangeNoReason,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be destroyed
  - resource "test_instance" "example" {}
`,
		},
		"delete because of wrong repetition mode (NoKey)": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceDeleteBecauseWrongRepetition,
			Mode:            addrs.ManagedResourceMode,
			InstanceKey:     addrs.NoKey,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be destroyed
  # (because resource uses count or for_each)
  - resource "test_instance" "example" {}
`,
		},
		"delete because of wrong repetition mode (IntKey)": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceDeleteBecauseWrongRepetition,
			Mode:            addrs.ManagedResourceMode,
			InstanceKey:     addrs.IntKey(1),
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example[1] will be destroyed
  # (because resource does not use count)
  - resource "test_instance" "example" {}
`,
		},
		"delete because of wrong repetition mode (StringKey)": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceDeleteBecauseWrongRepetition,
			Mode:            addrs.ManagedResourceMode,
			InstanceKey:     addrs.StringKey("a"),
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example["a"] will be destroyed
  # (because resource does not use for_each)
  - resource "test_instance" "example" {}
`,
		},
		"delete because no resource configuration": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceDeleteBecauseNoResourceConfig,
			ModuleInst:      addrs.RootModuleInstance.Child("foo", addrs.NoKey),
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # module.foo.test_instance.example will be destroyed
  # (because test_instance.example is not in configuration)
  - resource "test_instance" "example" {}
`,
		},
		"delete because no module": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceDeleteBecauseNoModule,
			ModuleInst:      addrs.RootModuleInstance.Child("foo", addrs.IntKey(1)),
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # module.foo[1].test_instance.example will be destroyed
  # (because module.foo[1] is not in configuration)
  - resource "test_instance" "example" {}
`,
		},
		"delete because out of range for count": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceDeleteBecauseCountIndex,
			Mode:            addrs.ManagedResourceMode,
			InstanceKey:     addrs.IntKey(1),
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example[1] will be destroyed
  # (because index [1] is out of range for count)
  - resource "test_instance" "example" {}
`,
		},
		"delete because out of range for for_each": {
			Action:          plans.Delete,
			ActionReason:    plans.ResourceInstanceDeleteBecauseEachKey,
			Mode:            addrs.ManagedResourceMode,
			InstanceKey:     addrs.StringKey("boop"),
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example["boop"] will be destroyed
  # (because key ["boop"] is not in for_each map)
  - resource "test_instance" "example" {}
`,
		},
		"replace for no particular reason (delete first)": {
			Action:          plans.DeleteThenCreate,
			ActionReason:    plans.ResourceInstanceChangeNoReason,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {}
`,
		},
		"replace for no particular reason (create first)": {
			Action:          plans.CreateThenDelete,
			ActionReason:    plans.ResourceInstanceChangeNoReason,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example must be replaced
+/- resource "test_instance" "example" {}
`,
		},
		"replace by request (delete first)": {
			Action:          plans.DeleteThenCreate,
			ActionReason:    plans.ResourceInstanceReplaceByRequest,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be replaced, as requested
-/+ resource "test_instance" "example" {}
`,
		},
		"replace by request (create first)": {
			Action:          plans.CreateThenDelete,
			ActionReason:    plans.ResourceInstanceReplaceByRequest,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be replaced, as requested
+/- resource "test_instance" "example" {}
`,
		},
		"replace because tainted (delete first)": {
			Action:          plans.DeleteThenCreate,
			ActionReason:    plans.ResourceInstanceReplaceBecauseTainted,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example is tainted, so must be replaced
-/+ resource "test_instance" "example" {}
`,
		},
		"replace because tainted (create first)": {
			Action:          plans.CreateThenDelete,
			ActionReason:    plans.ResourceInstanceReplaceBecauseTainted,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example is tainted, so must be replaced
+/- resource "test_instance" "example" {}
`,
		},
		"replace because cannot update (delete first)": {
			Action:          plans.DeleteThenCreate,
			ActionReason:    plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			// This one has no special message, because the fuller explanation
			// typically appears inline as a "# forces replacement" comment.
			// (not shown here)
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {}
`,
		},
		"replace because cannot update (create first)": {
			Action:          plans.CreateThenDelete,
			ActionReason:    plans.ResourceInstanceReplaceBecauseCannotUpdate,
			Mode:            addrs.ManagedResourceMode,
			Before:          emptyVal,
			After:           nullVal,
			Schema:          emptySchema,
			RequiredReplace: cty.NewPathSet(),
			// This one has no special message, because the fuller explanation
			// typically appears inline as a "# forces replacement" comment.
			// (not shown here)
			ExpectedOutput: `  # test_instance.example must be replaced
+/- resource "test_instance" "example" {}
`,
		},
	}

	runTestCases(t, testCases)
}

func TestResourceChange_sensitiveVariable(t *testing.T) {
	testCases := map[string]testCase{
		"creation": {
			Action: plans.Create,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.NullVal(cty.EmptyObject),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-123"),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(2000),
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("pizza"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
					cty.StringVal("!"),
				}),
				"nested_block_list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
						"another": cty.StringVal("not secret"),
					}),
				}),
				"nested_block_set": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
						"another": cty.StringVal("not secret"),
					}),
				}),
			}),
			AfterValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(1)}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_whole"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_key"}, cty.IndexStep{Key: cty.StringVal("dinner")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					// Nested blocks/sets will mark the whole set/block as sensitive
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_list"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_set"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"map_whole":  {Type: cty.Map(cty.String), Optional: true},
					"map_key":    {Type: cty.Map(cty.Number), Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_block_list": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
								"another": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingList,
					},
					"nested_block_set": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
								"another": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + ami        = (sensitive)
      + id         = "i-02ae66f368e8518a9"
      + list_field = [
          + "hello",
          + (sensitive),
          + "!",
        ]
      + map_key    = {
          + "breakfast" = 800
          + "dinner"    = (sensitive)
        }
      + map_whole  = (sensitive)

      + nested_block_list {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }

      + nested_block_set {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }
`,
		},
		"in-place update - before sensitive": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":          cty.StringVal("i-02ae66f368e8518a9"),
				"ami":         cty.StringVal("ami-BEFORE"),
				"special":     cty.BoolVal(true),
				"some_number": cty.NumberIntVal(1),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
					cty.StringVal("!"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(2000), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("pizza"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
					}),
				}),
				"nested_block_set": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":          cty.StringVal("i-02ae66f368e8518a9"),
				"ami":         cty.StringVal("ami-AFTER"),
				"special":     cty.BoolVal(false),
				"some_number": cty.NumberIntVal(2),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
					cty.StringVal("."),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(1900),
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("cereal"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("changed"),
					}),
				}),
				"nested_block_set": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("changed"),
					}),
				}),
			}),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "special"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "some_number"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(2)}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_key"}, cty.IndexStep{Key: cty.StringVal("dinner")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_whole"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_set"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":          {Type: cty.String, Optional: true, Computed: true},
					"ami":         {Type: cty.String, Optional: true},
					"list_field":  {Type: cty.List(cty.String), Optional: true},
					"special":     {Type: cty.Bool, Optional: true},
					"some_number": {Type: cty.Number, Optional: true},
					"map_key":     {Type: cty.Map(cty.Number), Optional: true},
					"map_whole":   {Type: cty.Map(cty.String), Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingList,
					},
					"nested_block_set": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change.
      ~ ami         = (sensitive)
        id          = "i-02ae66f368e8518a9"
      ~ list_field  = [
            # (1 unchanged element hidden)
            "friends",
          - (sensitive),
          + ".",
        ]
      ~ map_key     = {
          # Warning: this attribute value will no longer be marked as sensitive
          # after applying this change.
          ~ "dinner"    = (sensitive)
            # (1 unchanged element hidden)
        }
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change.
      ~ map_whole   = (sensitive)
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change.
      ~ some_number = (sensitive)
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change.
      ~ special     = (sensitive)

      # Warning: this block will no longer be marked as sensitive
      # after applying this change.
      ~ nested_block {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }

      # Warning: this block will no longer be marked as sensitive
      # after applying this change.
      ~ nested_block_set {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }
`,
		},
		"in-place update - after sensitive": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(2000), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("pizza"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block_single": cty.ObjectVal(map[string]cty.Value{
					"an_attr": cty.StringVal("original"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("goodbye"),
					cty.StringVal("friends"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(700),
					"dinner":    cty.NumberIntVal(2100), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("cereal"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block_single": cty.ObjectVal(map[string]cty.Value{
					"an_attr": cty.StringVal("changed"),
				}),
			}),
			AfterValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "tags"}, cty.IndexStep{Key: cty.StringVal("address")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(0)}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_key"}, cty.IndexStep{Key: cty.StringVal("dinner")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_whole"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_single"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
					"map_key":    {Type: cty.Map(cty.Number), Optional: true},
					"map_whole":  {Type: cty.Map(cty.String), Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_block_single": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingSingle,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        id         = "i-02ae66f368e8518a9"
      ~ list_field = [
          - "hello",
          + (sensitive),
            "friends",
        ]
      ~ map_key    = {
          ~ "breakfast" = 800 -> 700
          # Warning: this attribute value will be marked as sensitive and will not
          # display in UI output after applying this change.
          ~ "dinner"    = (sensitive)
        }
      # Warning: this attribute value will be marked as sensitive and will not
      # display in UI output after applying this change.
      ~ map_whole  = (sensitive)

      # Warning: this block will be marked as sensitive and will not
      # display in UI output after applying this change.
      ~ nested_block_single {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }
`,
		},
		"in-place update - both sensitive": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(2000), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("pizza"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block_map": cty.MapVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("original"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("goodbye"),
					cty.StringVal("friends"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(1800), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("cereal"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block_map": cty.MapVal(map[string]cty.Value{
					"foo": cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.UnknownVal(cty.String),
					}),
				}),
			}),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(0)}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_key"}, cty.IndexStep{Key: cty.StringVal("dinner")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_whole"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_map"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			AfterValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(0)}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_key"}, cty.IndexStep{Key: cty.StringVal("dinner")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_whole"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_map"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
					"map_key":    {Type: cty.Map(cty.Number), Optional: true},
					"map_whole":  {Type: cty.Map(cty.String), Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_block_map": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingMap,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami        = (sensitive)
        id         = "i-02ae66f368e8518a9"
      ~ list_field = [
          - (sensitive),
          + (sensitive),
            "friends",
        ]
      ~ map_key    = {
          ~ "dinner"    = (sensitive)
            # (1 unchanged element hidden)
        }
      ~ map_whole  = (sensitive)

      ~ nested_block_map {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }
`,
		},
		"in-place update - value unchanged, sensitivity changes": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":          cty.StringVal("i-02ae66f368e8518a9"),
				"ami":         cty.StringVal("ami-BEFORE"),
				"special":     cty.BoolVal(true),
				"some_number": cty.NumberIntVal(1),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
					cty.StringVal("!"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(2000), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("pizza"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
					}),
				}),
				"nested_block_set": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":          cty.StringVal("i-02ae66f368e8518a9"),
				"ami":         cty.StringVal("ami-BEFORE"),
				"special":     cty.BoolVal(true),
				"some_number": cty.NumberIntVal(1),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
					cty.StringVal("!"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(2000), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("pizza"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
					}),
				}),
				"nested_block_set": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secretval"),
					}),
				}),
			}),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "special"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "some_number"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(2)}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_key"}, cty.IndexStep{Key: cty.StringVal("dinner")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_whole"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_set"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":          {Type: cty.String, Optional: true, Computed: true},
					"ami":         {Type: cty.String, Optional: true},
					"list_field":  {Type: cty.List(cty.String), Optional: true},
					"special":     {Type: cty.Bool, Optional: true},
					"some_number": {Type: cty.Number, Optional: true},
					"map_key":     {Type: cty.Map(cty.Number), Optional: true},
					"map_whole":   {Type: cty.Map(cty.String), Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_block": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingList,
					},
					"nested_block_set": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change. The value is unchanged.
      ~ ami         = (sensitive)
        id          = "i-02ae66f368e8518a9"
      ~ list_field  = [
            # (1 unchanged element hidden)
            "friends",
          - (sensitive),
          + "!",
        ]
      ~ map_key     = {
          # Warning: this attribute value will no longer be marked as sensitive
          # after applying this change. The value is unchanged.
          ~ "dinner"    = (sensitive)
            # (1 unchanged element hidden)
        }
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change. The value is unchanged.
      ~ map_whole   = (sensitive)
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change. The value is unchanged.
      ~ some_number = (sensitive)
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change. The value is unchanged.
      ~ special     = (sensitive)

      # Warning: this block will no longer be marked as sensitive
      # after applying this change.
      ~ nested_block {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }

      # Warning: this block will no longer be marked as sensitive
      # after applying this change.
      ~ nested_block_set {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }
`,
		},
		"deletion": {
			Action: plans.Delete,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
				}),
				"map_key": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.NumberIntVal(800),
					"dinner":    cty.NumberIntVal(2000), // sensitive key
				}),
				"map_whole": cty.MapVal(map[string]cty.Value{
					"breakfast": cty.StringVal("pizza"),
					"dinner":    cty.StringVal("pizza"),
				}),
				"nested_block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secret"),
						"another": cty.StringVal("not secret"),
					}),
				}),
				"nested_block_set": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secret"),
						"another": cty.StringVal("not secret"),
					}),
				}),
			}),
			After: cty.NullVal(cty.EmptyObject),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(1)}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_key"}, cty.IndexStep{Key: cty.StringVal("dinner")}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "map_whole"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "nested_block_set"}},
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
					"map_key":    {Type: cty.Map(cty.Number), Optional: true},
					"map_whole":  {Type: cty.Map(cty.String), Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_block_set": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Optional: true},
								"another": {Type: cty.String, Optional: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be destroyed
  - resource "test_instance" "example" {
      - ami        = (sensitive) -> null
      - id         = "i-02ae66f368e8518a9" -> null
      - list_field = [
          - "hello",
          - (sensitive),
        ] -> null
      - map_key    = {
          - "breakfast" = 800
          - "dinner"    = (sensitive)
        } -> null
      - map_whole  = (sensitive) -> null

      - nested_block_set {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }
`,
		},
		"update with sensitive value forcing replacement": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"nested_block_set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("secret"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"nested_block_set": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"an_attr": cty.StringVal("changed"),
					}),
				}),
			}),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.GetAttrPath("ami"),
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.GetAttrPath("nested_block_set"),
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			AfterValMarks: []cty.PathValueMarks{
				{
					Path:  cty.GetAttrPath("ami"),
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
				{
					Path:  cty.GetAttrPath("nested_block_set"),
					Marks: cty.NewValueMarks(marks.Sensitive),
				},
			},
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"nested_block_set": {
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"an_attr": {Type: cty.String, Required: true},
							},
						},
						Nesting: configschema.NestingSet,
					},
				},
			},
			RequiredReplace: cty.NewPathSet(
				cty.GetAttrPath("ami"),
				cty.GetAttrPath("nested_block_set"),
			),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = (sensitive) # forces replacement
        id  = "i-02ae66f368e8518a9"

      ~ nested_block_set { # forces replacement
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }
`,
		},
		"update with sensitive attribute forcing replacement": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true, Computed: true, Sensitive: true},
				},
			},
			RequiredReplace: cty.NewPathSet(
				cty.GetAttrPath("ami"),
			),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = (sensitive value) # forces replacement
        id  = "i-02ae66f368e8518a9"
    }
`,
		},
		"update with sensitive nested type attribute forcing replacement": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"conn_info": cty.ObjectVal(map[string]cty.Value{
					"user":     cty.StringVal("not-secret"),
					"password": cty.StringVal("top-secret"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"conn_info": cty.ObjectVal(map[string]cty.Value{
					"user":     cty.StringVal("not-secret"),
					"password": cty.StringVal("new-secret"),
				}),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id": {Type: cty.String, Optional: true, Computed: true},
					"conn_info": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSingle,
							Attributes: map[string]*configschema.Attribute{
								"user":     {Type: cty.String, Optional: true},
								"password": {Type: cty.String, Optional: true, Sensitive: true},
							},
						},
					},
				},
			},
			RequiredReplace: cty.NewPathSet(
				cty.GetAttrPath("conn_info"),
				cty.GetAttrPath("password"),
			),
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ conn_info = { # forces replacement
          ~ password = (sensitive value)
            # (1 unchanged attribute hidden)
        }
        id        = "i-02ae66f368e8518a9"
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_moved(t *testing.T) {
	prevRunAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "previous",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	testCases := map[string]testCase{
		"moved and updated": {
			PrevRunAddr: prevRunAddr,
			Action:      plans.Update,
			Mode:        addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("12345"),
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("baz"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("12345"),
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("boop"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Computed: true},
					"foo": {Type: cty.String, Optional: true},
					"bar": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  # (moved from test_instance.previous)
  ~ resource "test_instance" "example" {
      ~ bar = "baz" -> "boop"
        id  = "12345"
        # (1 unchanged attribute hidden)
    }
`,
		},
		"moved without changes": {
			PrevRunAddr: prevRunAddr,
			Action:      plans.NoOp,
			Mode:        addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("12345"),
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("baz"),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("12345"),
				"foo": cty.StringVal("hello"),
				"bar": cty.StringVal("baz"),
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Computed: true},
					"foo": {Type: cty.String, Optional: true},
					"bar": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.previous has moved to test_instance.example
    resource "test_instance" "example" {
        id  = "12345"
        # (2 unchanged attributes hidden)
    }
`,
		},
	}

	runTestCases(t, testCases)
}

type testCase struct {
	Action          plans.Action
	ActionReason    plans.ResourceInstanceChangeActionReason
	ModuleInst      addrs.ModuleInstance
	Mode            addrs.ResourceMode
	InstanceKey     addrs.InstanceKey
	DeposedKey      states.DeposedKey
	Before          cty.Value
	BeforeValMarks  []cty.PathValueMarks
	AfterValMarks   []cty.PathValueMarks
	After           cty.Value
	Schema          *configschema.Block
	RequiredReplace cty.PathSet
	ExpectedOutput  string
	PrevRunAddr     addrs.AbsResourceInstance
}

func runTestCases(t *testing.T, testCases map[string]testCase) {
	color := &colorstring.Colorize{Colors: colorstring.DefaultColors, Disable: true}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ty := tc.Schema.ImpliedType()

			beforeVal := tc.Before
			switch { // Some fixups to make the test cases a little easier to write
			case beforeVal.IsNull():
				beforeVal = cty.NullVal(ty) // allow mistyped nulls
			case !beforeVal.IsKnown():
				beforeVal = cty.UnknownVal(ty) // allow mistyped unknowns
			}
			before, err := plans.NewDynamicValue(beforeVal, ty)
			if err != nil {
				t.Fatal(err)
			}

			afterVal := tc.After
			switch { // Some fixups to make the test cases a little easier to write
			case afterVal.IsNull():
				afterVal = cty.NullVal(ty) // allow mistyped nulls
			case !afterVal.IsKnown():
				afterVal = cty.UnknownVal(ty) // allow mistyped unknowns
			}
			after, err := plans.NewDynamicValue(afterVal, ty)
			if err != nil {
				t.Fatal(err)
			}

			addr := addrs.Resource{
				Mode: tc.Mode,
				Type: "test_instance",
				Name: "example",
			}.Instance(tc.InstanceKey).Absolute(tc.ModuleInst)

			prevRunAddr := tc.PrevRunAddr
			// If no previous run address is given, reuse the current address
			// to make initialization easier
			if prevRunAddr.Resource.Resource.Type == "" {
				prevRunAddr = addr
			}

			change := &plans.ResourceInstanceChangeSrc{
				Addr:        addr,
				PrevRunAddr: prevRunAddr,
				DeposedKey:  tc.DeposedKey,
				ProviderAddr: addrs.AbsProviderConfig{
					Provider: addrs.NewDefaultProvider("test"),
					Module:   addrs.RootModule,
				},
				ChangeSrc: plans.ChangeSrc{
					Action:         tc.Action,
					Before:         before,
					After:          after,
					BeforeValMarks: tc.BeforeValMarks,
					AfterValMarks:  tc.AfterValMarks,
				},
				ActionReason:    tc.ActionReason,
				RequiredReplace: tc.RequiredReplace,
			}

			output := ResourceChange(change, tc.Schema, color, DiffLanguageProposedChange)
			if diff := cmp.Diff(output, tc.ExpectedOutput); diff != "" {
				t.Errorf("wrong output\n%s", diff)
			}
		})
	}
}

func TestOutputChanges(t *testing.T) {
	color := &colorstring.Colorize{Colors: colorstring.DefaultColors, Disable: true}

	testCases := map[string]struct {
		changes []*plans.OutputChangeSrc
		output  string
	}{
		"new output value": {
			[]*plans.OutputChangeSrc{
				outputChange(
					"foo",
					cty.NullVal(cty.DynamicPseudoType),
					cty.StringVal("bar"),
					false,
				),
			},
			`
  + foo = "bar"`,
		},
		"removed output": {
			[]*plans.OutputChangeSrc{
				outputChange(
					"foo",
					cty.StringVal("bar"),
					cty.NullVal(cty.DynamicPseudoType),
					false,
				),
			},
			`
  - foo = "bar" -> null`,
		},
		"single string change": {
			[]*plans.OutputChangeSrc{
				outputChange(
					"foo",
					cty.StringVal("bar"),
					cty.StringVal("baz"),
					false,
				),
			},
			`
  ~ foo = "bar" -> "baz"`,
		},
		"element added to list": {
			[]*plans.OutputChangeSrc{
				outputChange(
					"foo",
					cty.ListVal([]cty.Value{
						cty.StringVal("alpha"),
						cty.StringVal("beta"),
						cty.StringVal("delta"),
						cty.StringVal("epsilon"),
					}),
					cty.ListVal([]cty.Value{
						cty.StringVal("alpha"),
						cty.StringVal("beta"),
						cty.StringVal("gamma"),
						cty.StringVal("delta"),
						cty.StringVal("epsilon"),
					}),
					false,
				),
			},
			`
  ~ foo = [
        # (1 unchanged element hidden)
        "beta",
      + "gamma",
        "delta",
        # (1 unchanged element hidden)
    ]`,
		},
		"multiple outputs changed, one sensitive": {
			[]*plans.OutputChangeSrc{
				outputChange(
					"a",
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
					false,
				),
				outputChange(
					"b",
					cty.StringVal("hunter2"),
					cty.StringVal("correct-horse-battery-staple"),
					true,
				),
				outputChange(
					"c",
					cty.BoolVal(false),
					cty.BoolVal(true),
					false,
				),
			},
			`
  ~ a = 1 -> 2
  ~ b = (sensitive value)
  ~ c = false -> true`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			output := OutputChanges(tc.changes, color)
			if output != tc.output {
				t.Errorf("Unexpected diff.\ngot:\n%s\nwant:\n%s\n", output, tc.output)
			}
		})
	}
}

func outputChange(name string, before, after cty.Value, sensitive bool) *plans.OutputChangeSrc {
	addr := addrs.AbsOutputValue{
		OutputValue: addrs.OutputValue{Name: name},
	}

	change := &plans.OutputChange{
		Addr: addr, Change: plans.Change{
			Before: before,
			After:  after,
		},
		Sensitive: sensitive,
	}

	changeSrc, err := change.Encode()
	if err != nil {
		panic(fmt.Sprintf("failed to encode change for %s: %s", addr, err))
	}

	return changeSrc
}

// A basic test schema using a configurable NestingMode for one (NestedType) attribute and one block
func testSchema(nesting configschema.NestingMode) *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":  {Type: cty.String, Optional: true, Computed: true},
			"ami": {Type: cty.String, Optional: true},
			"disks": {
				NestedType: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"mount_point": {Type: cty.String, Optional: true},
						"size":        {Type: cty.String, Optional: true},
					},
					Nesting: nesting,
				},
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"root_block_device": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"volume_type": {
							Type:     cty.String,
							Optional: true,
							Computed: true,
						},
					},
				},
				Nesting: nesting,
			},
		},
	}
}

// similar to testSchema with the addition of a "new_field" block
func testSchemaPlus(nesting configschema.NestingMode) *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":  {Type: cty.String, Optional: true, Computed: true},
			"ami": {Type: cty.String, Optional: true},
			"disks": {
				NestedType: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"mount_point": {Type: cty.String, Optional: true},
						"size":        {Type: cty.String, Optional: true},
					},
					Nesting: nesting,
				},
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"root_block_device": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"volume_type": {
							Type:     cty.String,
							Optional: true,
							Computed: true,
						},
						"new_field": {
							Type:     cty.String,
							Optional: true,
							Computed: true,
						},
					},
				},
				Nesting: nesting,
			},
		},
	}
}
