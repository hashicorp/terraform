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
      ~ ami = "ami-BEFORE" -> "ami-AFTER"
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
