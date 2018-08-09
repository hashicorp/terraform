package plugin

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var (
	equateEmpty   = cmpopts.EquateEmpty()
	typeComparer  = cmp.Comparer(cty.Type.Equals)
	valueComparer = cmp.Comparer(cty.Value.RawEquals)
)

// Test that we can convert configschema to protobuf types and back again.
func TestConvertSchemaBlocks(t *testing.T) {
	tests := map[string]struct {
		Block *proto.Schema_Block
		Want  *configschema.Block
	}{
		"attributes": {
			&proto.Schema_Block{
				Attributes: []*proto.Schema_Attribute{
					{
						Name:     "computed",
						Type:     []byte(`["list","bool"]`),
						Computed: true,
					},
					{
						Name:     "optional",
						Type:     []byte(`"string"`),
						Optional: true,
					},
					{
						Name:     "optional_computed",
						Type:     []byte(`["map","bool"]`),
						Optional: true,
						Computed: true,
					},
					{
						Name:     "required",
						Type:     []byte(`"number"`),
						Required: true,
					},
				},
			},
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"computed": {
						Type:     cty.List(cty.Bool),
						Computed: true,
					},
					"optional": {
						Type:     cty.String,
						Optional: true,
					},
					"optional_computed": {
						Type:     cty.Map(cty.Bool),
						Optional: true,
						Computed: true,
					},
					"required": {
						Type:     cty.Number,
						Required: true,
					},
				},
			},
		},
		"blocks": {
			&proto.Schema_Block{
				BlockTypes: []*proto.Schema_NestedBlock{
					{
						TypeName: "list",
						Nesting:  proto.Schema_NestedBlock_LIST,
						Block:    &proto.Schema_Block{},
					},
					{
						TypeName: "map",
						Nesting:  proto.Schema_NestedBlock_MAP,
						Block:    &proto.Schema_Block{},
					},
					{
						TypeName: "set",
						Nesting:  proto.Schema_NestedBlock_SET,
						Block:    &proto.Schema_Block{},
					},
					{
						TypeName: "single",
						Nesting:  proto.Schema_NestedBlock_SINGLE,
						Block: &proto.Schema_Block{
							Attributes: []*proto.Schema_Attribute{
								{
									Name:     "foo",
									Type:     []byte(`"dynamic"`),
									Required: true,
								},
							},
						},
					},
				},
			},
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"list": &configschema.NestedBlock{
						Nesting: configschema.NestingList,
					},
					"map": &configschema.NestedBlock{
						Nesting: configschema.NestingMap,
					},
					"set": &configschema.NestedBlock{
						Nesting: configschema.NestingSet,
					},
					"single": &configschema.NestedBlock{
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:     cty.DynamicPseudoType,
									Required: true,
								},
							},
						},
					},
				},
			},
		},
		"deep block nesting": {
			&proto.Schema_Block{
				BlockTypes: []*proto.Schema_NestedBlock{
					{
						TypeName: "single",
						Nesting:  proto.Schema_NestedBlock_SINGLE,
						Block: &proto.Schema_Block{
							BlockTypes: []*proto.Schema_NestedBlock{
								{
									TypeName: "list",
									Nesting:  proto.Schema_NestedBlock_LIST,
									Block: &proto.Schema_Block{
										BlockTypes: []*proto.Schema_NestedBlock{
											{
												TypeName: "set",
												Nesting:  proto.Schema_NestedBlock_SET,
												Block:    &proto.Schema_Block{},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			&configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"single": &configschema.NestedBlock{
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							BlockTypes: map[string]*configschema.NestedBlock{
								"list": &configschema.NestedBlock{
									Nesting: configschema.NestingList,
									Block: configschema.Block{
										BlockTypes: map[string]*configschema.NestedBlock{
											"set": &configschema.NestedBlock{
												Nesting: configschema.NestingSet,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			converted := schemaBlock(tc.Block)
			if !cmp.Equal(converted, tc.Want, typeComparer, valueComparer, equateEmpty) {
				t.Fatal(cmp.Diff(converted, tc.Want, typeComparer, valueComparer, equateEmpty))
			}
		})
	}
}

func TestDiagnostics(t *testing.T) {
	type diagFlat struct {
		Severity tfdiags.Severity
		Attr     []interface{}
		Summary  string
		Detail   string
	}

	tests := map[string]struct {
		Cons func([]*proto.Diagnostic) []*proto.Diagnostic
		Want []diagFlat
	}{
		"nil": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return diags
			},
			nil,
		},
		"error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "simple error",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "simple error",
				},
			},
		},
		"detailed error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "simple error",
					Detail:   "detailed error",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "simple error",
					Detail:   "detailed error",
				},
			},
		},
		"warning": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_WARNING,
					Summary:  "simple warning",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Warning,
					Summary:  "simple warning",
				},
			},
		},
		"detailed warning": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_WARNING,
					Summary:  "simple warning",
					Detail:   "detailed warning",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Warning,
					Summary:  "simple warning",
					Detail:   "detailed warning",
				},
			},
		},
		"multi error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "first error",
				}, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "second error",
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "first error",
				},
				{
					Severity: tfdiags.Error,
					Summary:  "second error",
				},
			},
		},
		"warning and error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_WARNING,
					Summary:  "warning",
				}, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "error",
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Warning,
					Summary:  "warning",
				},
				{
					Severity: tfdiags.Error,
					Summary:  "error",
				},
			},
		},
		"attr error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "error",
					Detail:   "error detail",
					Attribute: &proto.AttributePath{
						Steps: []*proto.AttributePath_Step{
							{
								Selector: &proto.AttributePath_Step_AttributeName{
									AttributeName: "attribute_name",
								},
							},
						},
					},
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "error",
					Detail:   "error detail",
					Attr:     []interface{}{"attribute_name"},
				},
			},
		},
		"multi attr": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags,
					&proto.Diagnostic{
						Severity: proto.Diagnostic_ERROR,
						Summary:  "error 1",
						Detail:   "error 1 detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
							},
						},
					},
					&proto.Diagnostic{
						Severity: proto.Diagnostic_ERROR,
						Summary:  "error 2",
						Detail:   "error 2 detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "sub",
									},
								},
							},
						},
					},
					&proto.Diagnostic{
						Severity: proto.Diagnostic_WARNING,
						Summary:  "warning",
						Detail:   "warning detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
								{
									Selector: &proto.AttributePath_Step_ElementKeyInt{
										ElementKeyInt: 1,
									},
								},
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "sub",
									},
								},
							},
						},
					},
					&proto.Diagnostic{
						Severity: proto.Diagnostic_ERROR,
						Summary:  "error 3",
						Detail:   "error 3 detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
								{
									Selector: &proto.AttributePath_Step_ElementKeyString{
										ElementKeyString: "idx",
									},
								},
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "sub",
									},
								},
							},
						},
					},
				)

				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "error 1",
					Detail:   "error 1 detail",
					Attr:     []interface{}{"attr"},
				},
				{
					Severity: tfdiags.Error,
					Summary:  "error 2",
					Detail:   "error 2 detail",
					Attr:     []interface{}{"attr", "sub"},
				},
				{
					Severity: tfdiags.Warning,
					Summary:  "warning",
					Detail:   "warning detail",
					Attr:     []interface{}{"attr", 1, "sub"},
				},
				{
					Severity: tfdiags.Error,
					Summary:  "error 3",
					Detail:   "error 3 detail",
					Attr:     []interface{}{"attr", "idx", "sub"},
				},
			},
		},
	}

	flattenTFDiags := func(ds tfdiags.Diagnostics) []diagFlat {
		var flat []diagFlat
		for _, item := range ds {
			desc := item.Description()

			var attr []interface{}

			for _, a := range tfdiags.GetAttribute(item) {
				switch step := a.(type) {
				case cty.GetAttrStep:
					attr = append(attr, step.Name)
				case cty.IndexStep:
					switch step.Key.Type() {
					case cty.Number:
						i, _ := step.Key.AsBigFloat().Int64()
						attr = append(attr, int(i))
					case cty.String:
						attr = append(attr, step.Key.AsString())
					}
				}
			}

			flat = append(flat, diagFlat{
				Severity: item.Severity(),
				Attr:     attr,
				Summary:  desc.Summary,
				Detail:   desc.Detail,
			})
		}
		return flat
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// we take the
			tfDiags := ProtoToDiagnostics(tc.Cons(nil))

			flat := flattenTFDiags(tfDiags)

			if !cmp.Equal(flat, tc.Want, typeComparer, valueComparer, equateEmpty) {
				t.Fatal(cmp.Diff(flat, tc.Want, typeComparer, valueComparer, equateEmpty))
			}
		})
	}
}
