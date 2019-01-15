package format

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
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
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + id = (known after apply)
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
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ more_lines = <<~EOT
            original
          + new line
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
			ExpectedOutput: `  # test_instance.example will be created
  + resource "test_instance" "example" {
      + id       = (known after apply)
      + password = (sensitive value)
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
		"in-place update": {
			Action: plans.Update,
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
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
                aaa = "value"
              + bbb = "new_value"
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
			ExpectedOutput: `  # test_instance.example must be replaced
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ json_field = jsonencode(
          ~ {
                first  = "111"
              + second = "222"
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
		"JSON list of objects": {
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
		// TODO: JSON with unknown values inside
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			RequiredReplace: cty.NewPathSet(),
			ExpectedOutput: `  # test_instance.example will be updated in-place
  ~ resource "test_instance" "example" {
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
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
				}),
			}),
			After: cty.ObjectVal(map[string]cty.Value{
				"id":  cty.UnknownVal(cty.String),
				"ami": cty.StringVal("ami-STATIC"),
				"list_field": cty.ListVal([]cty.Value{
					cty.StringVal("bbbb"),
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
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
          - "aaaa",
            "bbbb",
          - "cccc",
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
        ami        = "ami-STATIC"
      ~ id         = "i-02ae66f368e8518a9" -> (known after apply)
      ~ list_field = [
            "aaaa",
          - "bbbb",
          + (known after apply),
          + (known after apply),
            "cccc",
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example must be replaced
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
			ExpectedOutput: `  # test_instance.example must be replaced
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
			ExpectedOutput: `  # test_instance.example will be updated in-place
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
				"id":                cty.StringVal("i-02ae66f368e8518a9"),
				"ami":               cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.ListValEmpty(cty.EmptyObject),
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
				"id":                cty.StringVal("i-02ae66f368e8518a9"),
				"ami":               cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.ListValEmpty(cty.EmptyObject),
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
				"id":                cty.StringVal("i-02ae66f368e8518a9"),
				"ami":               cty.StringVal("ami-AFTER"),
				"root_block_device": cty.ListValEmpty(cty.EmptyObject),
			}),
			RequiredReplace: cty.NewPathSet(),
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
	}
	runTestCases(t, testCases)
}

func TestResourceChange_nestedSet(t *testing.T) {
	testCases := map[string]testCase{
		"in-place update - creation": {
			Action: plans.Update,
			Mode:   addrs.ManagedResourceMode,
			Before: cty.ObjectVal(map[string]cty.Value{
				"id":                cty.StringVal("i-02ae66f368e8518a9"),
				"ami":               cty.StringVal("ami-BEFORE"),
				"root_block_device": cty.SetValEmpty(cty.EmptyObject),
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
          - volume_type = "gp2" -> null
        }
      + root_block_device {
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

      - root_block_device { # forces replacement
          - volume_type = "gp2" -> null
        }
      + root_block_device { # forces replacement
          + volume_type = "different"
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
				"id":                cty.StringVal("i-02ae66f368e8518a9"),
				"ami":               cty.StringVal("ami-AFTER"),
				"root_block_device": cty.SetValEmpty(cty.EmptyObject),
			}),
			RequiredReplace: cty.NewPathSet(),
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

type testCase struct {
	Action          plans.Action
	Mode            addrs.ResourceMode
	Before          cty.Value
	After           cty.Value
	Schema          *configschema.Block
	RequiredReplace cty.PathSet
	ExpectedOutput  string
}

func runTestCases(t *testing.T, testCases map[string]testCase) {
	color := &colorstring.Colorize{Colors: colorstring.DefaultColors, Disable: true}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			beforeVal := tc.Before
			before, err := plans.NewDynamicValue(beforeVal, beforeVal.Type())
			if err != nil {
				t.Fatal(err)
			}

			afterVal := tc.After
			after, err := plans.NewDynamicValue(afterVal, afterVal.Type())
			if err != nil {
				t.Fatal(err)
			}

			change := &plans.ResourceInstanceChangeSrc{
				Addr: addrs.Resource{
					Mode: tc.Mode,
					Type: "test_instance",
					Name: "example",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				ProviderAddr: addrs.ProviderConfig{Type: "test"}.Absolute(addrs.RootModuleInstance),
				ChangeSrc: plans.ChangeSrc{
					Action: tc.Action,
					Before: before,
					After:  after,
				},
				RequiredReplace: tc.RequiredReplace,
			}

			output := ResourceChange(change, tc.Schema, color)
			if output != tc.ExpectedOutput {
				t.Fatalf("Unexpected diff.\nExpected:\n%s\nGiven:\n%s\n", tc.ExpectedOutput, output)
			}
		})
	}
}
