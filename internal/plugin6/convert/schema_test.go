package convert

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
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
					{
						Name: "nested_type",
						NestedType: &proto.Schema_Object{
							Nesting: proto.Schema_Object_SINGLE,
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
						Required: true,
					},
					{
						Name: "deeply_nested_type",
						NestedType: &proto.Schema_Object{
							Nesting: proto.Schema_Object_SINGLE,
							Attributes: []*proto.Schema_Attribute{
								{
									Name: "first_level",
									NestedType: &proto.Schema_Object{
										Nesting: proto.Schema_Object_SINGLE,
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
									Computed: true,
								},
							},
						},
						Required: true,
					},
					{
						Name: "nested_list",
						NestedType: &proto.Schema_Object{
							Nesting: proto.Schema_Object_LIST,
							Attributes: []*proto.Schema_Attribute{
								{
									Name:     "required",
									Type:     []byte(`"string"`),
									Computed: true,
								},
							},
						},
						Required: true,
					},
					{
						Name: "nested_set",
						NestedType: &proto.Schema_Object{
							Nesting: proto.Schema_Object_SET,
							Attributes: []*proto.Schema_Attribute{
								{
									Name:     "required",
									Type:     []byte(`"string"`),
									Computed: true,
								},
							},
						},
						Required: true,
					},
					{
						Name: "nested_map",
						NestedType: &proto.Schema_Object{
							Nesting: proto.Schema_Object_MAP,
							Attributes: []*proto.Schema_Attribute{
								{
									Name:     "required",
									Type:     []byte(`"string"`),
									Computed: true,
								},
							},
						},
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
					"nested_type": {
						NestedType: &configschema.Object{
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
							Nesting: configschema.NestingSingle,
						},
						Required: true,
					},
					"deeply_nested_type": {
						NestedType: &configschema.Object{
							Attributes: map[string]*configschema.Attribute{
								"first_level": {
									NestedType: &configschema.Object{
										Nesting: configschema.NestingSingle,
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
									Computed: true,
								},
							},
							Nesting: configschema.NestingSingle,
						},
						Required: true,
					},
					"nested_list": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingList,
							Attributes: map[string]*configschema.Attribute{
								"required": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
						Required: true,
					},
					"nested_map": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingMap,
							Attributes: map[string]*configschema.Attribute{
								"required": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
						Required: true,
					},
					"nested_set": {
						NestedType: &configschema.Object{
							Nesting: configschema.NestingSet,
							Attributes: map[string]*configschema.Attribute{
								"required": {
									Type:     cty.String,
									Computed: true,
								},
							},
						},
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
			converted := ProtoToConfigSchema(tc.Block)
			if !cmp.Equal(converted, tc.Want, typeComparer, valueComparer, equateEmpty) {
				t.Fatal(cmp.Diff(converted, tc.Want, typeComparer, valueComparer, equateEmpty))
			}
		})
	}
}

// Test that we can convert configschema to protobuf types and back again.
func TestConvertProtoSchemaBlocks(t *testing.T) {
	tests := map[string]struct {
		Want  *proto.Schema_Block
		Block *configschema.Block
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
			converted := ConfigSchemaToProto(tc.Block)
			if !cmp.Equal(converted, tc.Want, typeComparer, equateEmpty, ignoreUnexported) {
				t.Fatal(cmp.Diff(converted, tc.Want, typeComparer, equateEmpty, ignoreUnexported))
			}
		})
	}
}
