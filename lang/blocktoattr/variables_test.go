package blocktoattr

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	hcljson "github.com/hashicorp/hcl2/hcl/json"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestExpandedVariables(t *testing.T) {
	fooSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type: cty.List(cty.Object(map[string]cty.Type{
					"bar": cty.String,
				})),
				Optional: true,
			},
			"bar": {
				Type:     cty.Map(cty.String),
				Optional: true,
			},
		},
	}

	tests := map[string]struct {
		src    string
		json   bool
		schema *configschema.Block
		want   []hcl.Traversal
	}{
		"empty": {
			src:    ``,
			schema: &configschema.Block{},
			want:   nil,
		},
		"attribute syntax": {
			src: `
foo = [
  {
    bar = baz
  },
]
`,
			schema: fooSchema,
			want: []hcl.Traversal{
				{
					hcl.TraverseRoot{
						Name: "baz",
						SrcRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 4, Column: 11, Byte: 23},
							End:      hcl.Pos{Line: 4, Column: 14, Byte: 26},
						},
					},
				},
			},
		},
		"block syntax": {
			src: `
foo {
  bar = baz
}
`,
			schema: fooSchema,
			want: []hcl.Traversal{
				{
					hcl.TraverseRoot{
						Name: "baz",
						SrcRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 3, Column: 9, Byte: 15},
							End:      hcl.Pos{Line: 3, Column: 12, Byte: 18},
						},
					},
				},
			},
		},
		"block syntax with nested blocks": {
			src: `
foo {
  bar {
    boop = baz
  }
}
`,
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type: cty.List(cty.Object(map[string]cty.Type{
							"bar": cty.List(cty.Object(map[string]cty.Type{
								"boop": cty.String,
							})),
						})),
						Optional: true,
					},
				},
			},
			want: []hcl.Traversal{
				{
					hcl.TraverseRoot{
						Name: "baz",
						SrcRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 4, Column: 12, Byte: 26},
							End:      hcl.Pos{Line: 4, Column: 15, Byte: 29},
						},
					},
				},
			},
		},
		"dynamic block syntax": {
			src: `
dynamic "foo" {
  for_each = beep
  content {
    bar = baz
  }
}
`,
			schema: fooSchema,
			want: []hcl.Traversal{
				{
					hcl.TraverseRoot{
						Name: "beep",
						SrcRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 3, Column: 14, Byte: 30},
							End:      hcl.Pos{Line: 3, Column: 18, Byte: 34},
						},
					},
				},
				{
					hcl.TraverseRoot{
						Name: "baz",
						SrcRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 5, Column: 11, Byte: 57},
							End:      hcl.Pos{Line: 5, Column: 14, Byte: 60},
						},
					},
				},
			},
		},
		"misplaced dynamic block": {
			src: `
dynamic "bar" {
  for_each = beep
  content {
    key = val
  }
}
`,
			schema: fooSchema,
			want: []hcl.Traversal{
				{
					hcl.TraverseRoot{
						Name: "beep",
						SrcRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 3, Column: 14, Byte: 30},
							End:      hcl.Pos{Line: 3, Column: 18, Byte: 34},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var f *hcl.File
			var diags hcl.Diagnostics
			if test.json {
				f, diags = hcljson.Parse([]byte(test.src), "test.tf.json")
			} else {
				f, diags = hclsyntax.ParseConfig([]byte(test.src), "test.tf", hcl.Pos{Line: 1, Column: 1})
			}
			if diags.HasErrors() {
				for _, diag := range diags {
					t.Errorf("unexpected diagnostic: %s", diag)
				}
				t.FailNow()
			}

			got := ExpandedVariables(f.Body, test.schema)

			co := cmpopts.IgnoreUnexported(hcl.TraverseRoot{})
			if !cmp.Equal(got, test.want, co) {
				t.Errorf("wrong result\n%s", cmp.Diff(test.want, got, co))
			}
		})
	}

}
