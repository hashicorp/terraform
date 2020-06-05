package blocktoattr

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/dynblock"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	hcljson "github.com/hashicorp/hcl/v2/json"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestFixUpBlockAttrs(t *testing.T) {
	fooSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type: cty.List(cty.Object(map[string]cty.Type{
					"bar": cty.String,
				})),
				Optional: true,
			},
		},
	}

	tests := map[string]struct {
		src      string
		json     bool
		schema   *configschema.Block
		want     cty.Value
		wantErrs bool
	}{
		"empty": {
			src:    ``,
			schema: &configschema.Block{},
			want:   cty.EmptyObjectVal,
		},
		"empty JSON": {
			src:    `{}`,
			json:   true,
			schema: &configschema.Block{},
			want:   cty.EmptyObjectVal,
		},
		"unset": {
			src:    ``,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(fooSchema.Attributes["foo"].Type),
			}),
		},
		"unset JSON": {
			src:    `{}`,
			json:   true,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(fooSchema.Attributes["foo"].Type),
			}),
		},
		"no fixup required, with one value": {
			src: `
foo = [
  {
    bar = "baz"
  },
]
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("baz"),
					}),
				}),
			}),
		},
		"no fixup required, with two values": {
			src: `
foo = [
  {
    bar = "baz"
  },
  {
    bar = "boop"
  },
]
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("baz"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
					}),
				}),
			}),
		},
		"no fixup required, with values, JSON": {
			src:    `{"foo": [{"bar": "baz"}]}`,
			json:   true,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("baz"),
					}),
				}),
			}),
		},
		"no fixup required, empty": {
			src: `
foo = []
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListValEmpty(fooSchema.Attributes["foo"].Type.ElementType()),
			}),
		},
		"no fixup required, empty, JSON": {
			src:    `{"foo":[]}`,
			json:   true,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListValEmpty(fooSchema.Attributes["foo"].Type.ElementType()),
			}),
		},
		"fixup one block": {
			src: `
foo {
  bar = "baz"
}
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("baz"),
					}),
				}),
			}),
		},
		"fixup one block omitting attribute": {
			src: `
foo {}
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.NullVal(cty.String),
					}),
				}),
			}),
		},
		"fixup two blocks": {
			src: `
foo {
  bar = baz
}
foo {
  bar = "boop"
}
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("baz value"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
					}),
				}),
			}),
		},
		"interaction with dynamic block generation": {
			src: `
dynamic "foo" {
  for_each = ["baz", beep]
  content {
    bar = foo.value
  }
}
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("baz"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("beep value"),
					}),
				}),
			}),
		},
		"dynamic block with empty iterator": {
			src: `
dynamic "foo" {
  for_each = []
  content {
    bar = foo.value
  }
}
`,
			schema: fooSchema,
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(fooSchema.Attributes["foo"].Type),
			}),
		},
		"both attribute and block syntax": {
			src: `
foo = []
foo {
  bar = "baz"
}
`,
			schema:   fooSchema,
			wantErrs: true, // Unsupported block type (user must be consistent about whether they consider foo to be a block type or an attribute)
			want: cty.ObjectVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("baz"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"bar": cty.StringVal("boop"),
					}),
				}),
			}),
		},
		"fixup inside block": {
			src: `
container {
  foo {
    bar = "baz"
  }
  foo {
    bar = "boop"
  }
}
container {
  foo {
    bar = beep
  }
}
`,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"container": {
						Nesting: configschema.NestingList,
						Block:   *fooSchema,
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"container": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("baz"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("boop"),
							}),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("beep value"),
							}),
						}),
					}),
				}),
			}),
		},
		"fixup inside attribute-as-block": {
			src: `
container {
  foo {
    bar = "baz"
  }
  foo {
    bar = "boop"
  }
}
container {
  foo {
    bar = beep
  }
}
`,
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"container": {
						Type: cty.List(cty.Object(map[string]cty.Type{
							"foo": cty.List(cty.Object(map[string]cty.Type{
								"bar": cty.String,
							})),
						})),
						Optional: true,
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"container": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("baz"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("boop"),
							}),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("beep value"),
							}),
						}),
					}),
				}),
			}),
		},
		"nested fixup with dynamic block generation": {
			src: `
container {
  dynamic "foo" {
    for_each = ["baz", beep]
    content {
      bar = foo.value
    }
  }
}
`,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"container": {
						Nesting: configschema.NestingList,
						Block:   *fooSchema,
					},
				},
			},
			want: cty.ObjectVal(map[string]cty.Value{
				"container": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"foo": cty.ListVal([]cty.Value{
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("baz"),
							}),
							cty.ObjectVal(map[string]cty.Value{
								"bar": cty.StringVal("beep value"),
							}),
						}),
					}),
				}),
			}),
		},
	}

	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"bar":  cty.StringVal("bar value"),
			"baz":  cty.StringVal("baz value"),
			"beep": cty.StringVal("beep value"),
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

			// We'll expand dynamic blocks in the body first, to mimic how
			// we process this fixup when using the main "lang" package API.
			spec := test.schema.DecoderSpec()
			body := dynblock.Expand(f.Body, ctx)

			body = FixUpBlockAttrs(body, test.schema)
			got, diags := hcldec.Decode(body, spec, ctx)

			if test.wantErrs {
				if !diags.HasErrors() {
					t.Errorf("succeeded, but want error\ngot: %#v", got)
				}
				return
			}

			if !test.want.RawEquals(got) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
			for _, diag := range diags {
				t.Errorf("unexpected diagnostic: %s", diag)
			}
		})
	}
}
