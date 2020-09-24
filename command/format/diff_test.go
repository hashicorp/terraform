package format

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/experiment"
	"github.com/hashicorp/terraform/plans"
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be destroyed
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
			Tainted:         false,
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"
    }
`,
		},
		"string force-new update": {
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
					"ami": {Type: cty.String, Optional: true},
				},
			},
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "ami"},
			}),
			Tainted: false,
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
			Tainted:         false,
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ more_lines = <<~EOT
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + more_lines = <<~EOT
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
			Tainted: false,
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ more_lines = <<~EOT # forces replacement
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
			}),
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":       {Type: cty.String, Computed: true},
					"password": {Type: cty.String, Optional: true, Sensitive: true},
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + id       = (known after apply)
      + password = (sensitive value)
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id       = "blah" -> (known after apply)
      ~ str      = "before" -> "after"
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id       = "blah" -> (known after apply)
        password = (sensitive value)
      ~ str      = "before" -> "after"
    }
`,
		},

		// tainted resources
		"replace tainted resource": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
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
			Tainted: true,
			ExpectedOutput: `  # test_instance.example is tainted, so must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER" # forces replacement
      ~ id  = "i-02ae66f368e8518a9" -> (known after apply)
    }
`,
		},
		"force replacement with empty before value": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
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
			Tainted: false,
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      + forced = "example" # forces replacement
        name   = "name"
    }
`,
		},
		"force replacement with empty before value legacy": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
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
			Tainted: false,
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami  = "ami-BEFORE" -> "ami-AFTER"
        bar  = "bar"
        foo  = "foo"
        id   = "i-02ae66f368e8518a9"
        name = "alice"
        tags = {
            "name" = "bob"
        }
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
			Tainted:         false,
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
			Tainted:         false,
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

			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
                aaa = "value"
              + bbb = "new_value"
              - ccc = 5 -> null
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
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
			Tainted: false,
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
			VerboseOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
                aaa = "value"
              + bbb = "new_value"
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
			Tainted:         false,
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
			Action: plans.DeleteThenCreate,
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
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "json_field"},
			}),
			Tainted: false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ [
                "first",
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
			Tainted:         false,
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

			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ [
                "first",
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + list_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + list_field = [
          + "new-element",
        ]
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          + "new-element",
        ]
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
            "aaaa",
            "bbbb",
          + "cccc",
            "dddd",
            "eeee",
            "ffff",
        ]
    }
`,
		},
		"force-new update - insertion": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
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
			Tainted: false,
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
			VerboseOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [ # forces replacement
            "aaaa",
          + "bbbb",
            "cccc",
        ]
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          - "aaaa",
            "bbbb",
          - "cccc",
            "dddd",
            "eeee",
        ]
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
			Tainted:         false,
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          - "aaaa",
          - "bbbb",
          - "cccc",
        ]
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + list_field = []
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      + list_field = []
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
            "aaaa",
          - "bbbb",
          + (known after apply),
            "cccc",
        ]
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
            "aaaa",
          - "bbbb",
          + (known after apply),
          + (known after apply),
            "cccc",
            "dddd",
            "eeee",
        ]
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        id          = "i-02ae66f368e8518a9"
      ~ tuple_field = [
            "aaaa",
            "bbbb",
          - "dddd",
          + "cccc",
            "eeee",
            "ffff",
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + set_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + set_field = [
          + "new-element",
        ]
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          + "new-element",
        ]
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          + "new-element",
        ]
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
            "aaaa",
          + "bbbb",
            "cccc",
        ]
    }
`,
		},
		"force-new update - insertion": {
			Action: plans.DeleteThenCreate,
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
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "set_field"},
			}),
			Tainted: false,
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
			VerboseOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [ # forces replacement
            "aaaa",
          + "bbbb",
            "cccc",
        ]
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          - "aaaa",
            "bbbb",
          - "cccc",
        ]
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          - "aaaa",
          - "bbbb",
        ]
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + set_field = []
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + set_field = []
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
          - "aaaa",
          - "bbbb",
        ] -> (known after apply)
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ set_field = [
            "aaaa",
          - "bbbb",
          ~ (known after apply),
        ]
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + map_field = {
          + "new-key" = "new-element"
        }
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      + map_field = {
          + "new-key" = "new-element"
        }
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
			Tainted:         false,
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
          + "new-key" = "new-element"
        }
        # (1 unchanged attribute hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
          + "new-key" = "new-element"
        }
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
            "a" = "aaaa"
          + "b" = "bbbb"
            "c" = "cccc"
        }
    }
`,
		},
		"force-new update - insertion": {
			Action: plans.DeleteThenCreate,
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
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "map_field"},
			}),
			Tainted: false,
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
			VerboseOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = { # forces replacement
            "a" = "aaaa"
          + "b" = "bbbb"
            "c" = "cccc"
        }
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
          - "a" = "aaaa" -> null
            "b" = "bbbb"
          - "c" = "cccc" -> null
        }
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
			Tainted:         false,
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
			Tainted:         false,
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
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami       = "ami-STATIC"
      ~ id        = "i-02ae66f368e8518a9" -> (known after apply)
      ~ map_field = {
            "a" = "aaaa"
          ~ "b" = "bbbb" -> (known after apply)
            "c" = "cccc"
        }
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
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

        # (1 unchanged block hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

        root_block_device {
            volume_type = "gp2"
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
				"root_block_device": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.NullVal(cty.String),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

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
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

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
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      ~ root_block_device {
          + new_field   = "new_value"
            # (1 unchanged attribute hidden)
        }
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      ~ root_block_device {
          + new_field   = "new_value"
            volume_type = "gp2"
        }
    }
`,
		},
		"force-new update (inside block)": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("different"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "root_block_device"},
				cty.IndexStep{Key: cty.NumberIntVal(0)},
				cty.GetAttrStep{Name: "volume_type"},
			}),
			Tainted: false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      ~ root_block_device {
          ~ volume_type = "gp2" -> "different" # forces replacement
        }
    }
`,
		},
		"force-new update (whole block)": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("different"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "root_block_device"},
			}),
			Tainted: false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

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
				"root_block_device": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
					"new_field":   cty.String,
				})),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingList,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      - root_block_device {
          - new_field   = "new_value" -> null
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
			Tainted:         false,
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
				"root_block_device": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

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
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

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
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
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
			}),
			RequiredReplace: cty.NewPathSet(cty.Path{
				cty.GetAttrStep{Name: "root_block_device"},
			}),
			Tainted: false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

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
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
					"new_field":   cty.String,
				})),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingSet,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      - root_block_device {
          - new_field   = "new_value" -> null
          - volume_type = "gp2" -> null
        }
    }
`,
		},
	}
	runTestCases(t, testCases)
}

func TestResourceChange_nestedMap(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
				})),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingMap,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

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
				"root_block_device": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"volume_type": cty.StringVal("gp2"),
						"new_field":   cty.StringVal("new_value"),
					}),
				}),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingMap,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      ~ root_block_device "a" {
          + new_field   = "new_value"
            # (1 unchanged attribute hidden)
        }
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      ~ root_block_device "a" {
          + new_field   = "new_value"
            volume_type = "gp2"
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
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingMap,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      + root_block_device "b" {
          + new_field   = "new_value"
          + volume_type = "gp2"
        }
        # (1 unchanged block hidden)
    }
`,
			VerboseOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

        root_block_device "a" {
            volume_type = "gp2"
        }
      + root_block_device "b" {
          + new_field   = "new_value"
          + volume_type = "gp2"
        }
    }
`,
		},
		"force-new update (whole block)": {
			Action: plans.DeleteThenCreate,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-BEFORE"),
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
			}),
			Tainted: false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingMap,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      ~ root_block_device "a" { # forces replacement
          ~ volume_type = "gp2" -> "different"
        }
        # (1 unchanged block hidden)
    }
`,
			VerboseOutput: `  # test_instance.example must be replaced
-/+ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      ~ root_block_device "a" { # forces replacement
          ~ volume_type = "gp2" -> "different"
        }
        root_block_device "b" {
            volume_type = "standard"
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
				"root_block_device": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"volume_type": cty.String,
					"new_field":   cty.String,
				})),
			}),
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
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
						Nesting: configschema.NestingMap,
					},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
        id  = "i-02ae66f368e8518a9"

      - root_block_device "a" {
          - new_field   = "new_value" -> null
          - volume_type = "gp2" -> null
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
			Tainted:         false,
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
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
					cty.StringVal("!"),
				}),
			}),
			AfterValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(1)}},
					Marks: cty.NewValueMarks("sensitive"),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
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
			}),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "special"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "some_number"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(2)}},
					Marks: cty.NewValueMarks("sensitive"),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":          {Type: cty.String, Optional: true, Computed: true},
					"ami":         {Type: cty.String, Optional: true},
					"list_field":  {Type: cty.List(cty.String), Optional: true},
					"special":     {Type: cty.Bool, Optional: true},
					"some_number": {Type: cty.Number, Optional: true},
				},
			},
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change
      ~ ami         = (sensitive)
        id          = "i-02ae66f368e8518a9"
      ~ list_field  = [
            # (1 unchanged element hidden)
            "friends",
          - (sensitive),
          + ".",
        ]
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change
      ~ some_number = (sensitive)
      # Warning: this attribute value will no longer be marked as sensitive
      # after applying this change
      ~ special     = (sensitive)
    }
`,
		},
		"in-place update - after sensitive": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"tags": cty.MapVal(map[string]cty.Value{
					"name":    cty.StringVal("anna a"),
					"address": cty.StringVal("123 Main St"),
				}),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("friends"),
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("i-02ae66f368e8518a9"),
				"tags": cty.MapVal(map[string]cty.Value{
					"name":    cty.StringVal("anna b"),
					"address": cty.StringVal("123 Main Ave"),
				}),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("goodbye"),
					cty.StringVal("friends"),
				}),
			}),
			AfterValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "tags"}, cty.IndexStep{Key: cty.StringVal("address")}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(0)}},
					Marks: cty.NewValueMarks("sensitive"),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"tags":       {Type: cty.Map(cty.String), Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
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
      ~ tags       = {
          # Warning: this attribute value will be marked as sensitive and will
          # not display in UI output after applying this change
          ~ "address" = (sensitive)
          ~ "name"    = "anna a" -> "anna b"
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
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.StringVal("i-02ae66f368e8518a9"),
				"ami": cty.StringVal("ami-AFTER"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("goodbye"),
					cty.StringVal("friends"),
				}),
			}),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(0)}},
					Marks: cty.NewValueMarks("sensitive"),
				},
			},
			AfterValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(0)}},
					Marks: cty.NewValueMarks("sensitive"),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
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
			}),
			After: cty.NullVal(cty.EmptyObject),
			BeforeValMarks: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "ami"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "list_field"}, cty.IndexStep{Key: cty.NumberIntVal(1)}},
					Marks: cty.NewValueMarks("sensitive"),
				},
			},
			RequiredReplace: cty.NewPathSet(),
			Tainted:         false,
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"id":         {Type: cty.String, Optional: true, Computed: true},
					"ami":        {Type: cty.String, Optional: true},
					"list_field": {Type: cty.List(cty.String), Optional: true},
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
    }
`,
		},
	}
	runTestCases(t, testCases)
}

type testCase struct {
	Action          plans.Action
	Mode            addrs.ResourceMode
	Before          cty.Value
	BeforeValMarks  []cty.PathValueMarks
	AfterValMarks   []cty.PathValueMarks
	After           cty.Value
	Schema          *configschema.Block
	RequiredReplace cty.PathSet
	Tainted         bool
	ExpectedOutput  string

	// This field and all associated values can be removed if the concise diff
	// experiment succeeds.
	VerboseOutput string
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

			change := &plans.ResourceInstanceChangeSrc{
				Addr: addrs.Resource{
					Mode: tc.Mode,
					Type: "test_instance",
					Name: "example",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				ProviderAddr: addrs.AbsProviderConfig{
					Provider: addrs.NewLegacyProvider("test"),
					Module:   addrs.RootModule,
				},
				ChangeSrc: plans.ChangeSrc{
					Action:         tc.Action,
					Before:         before,
					After:          after,
					BeforeValMarks: tc.BeforeValMarks,
					AfterValMarks:  tc.AfterValMarks,
				},
				RequiredReplace: tc.RequiredReplace,
			}

			experiment.SetEnabled(experiment.X_concise_diff, true)
			output := ResourceChange(change, tc.Tainted, tc.Schema, color)
			if output != tc.ExpectedOutput {
				t.Errorf("Unexpected diff.\ngot:\n%s\nwant:\n%s\n", output, tc.ExpectedOutput)
				t.Errorf("%s", cmp.Diff(output, tc.ExpectedOutput))
			}

			// Temporary coverage for verbose diff behaviour. All lines below
			// in this function can be removed if the concise diff experiment
			// succeeds.
			if tc.VerboseOutput == "" {
				return
			}
			experiment.SetEnabled(experiment.X_concise_diff, false)
			output = ResourceChange(change, tc.Tainted, tc.Schema, color)
			if output != tc.VerboseOutput {
				t.Errorf("Unexpected diff.\ngot:\n%s\nwant:\n%s\n", output, tc.VerboseOutput)
				t.Errorf("%s", cmp.Diff(output, tc.VerboseOutput))
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
			experiment.SetEnabled(experiment.X_concise_diff, true)
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
