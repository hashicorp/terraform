package schema

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
)

var (
	equateEmpty  = cmpopts.EquateEmpty()
	typeComparer = cmp.Comparer(cty.Type.Equals)
)

// add the implicit "id" attribute for test resources
func testResource(block *configschema.Block) *configschema.Block {
	if block.Attributes == nil {
		block.Attributes = make(map[string]*configschema.Attribute)
	}

	if block.BlockTypes == nil {
		block.BlockTypes = make(map[string]*configschema.NestedBlock)
	}

	if block.Attributes["id"] == nil {
		block.Attributes["id"] = &configschema.Attribute{
			Type:     cty.String,
			Optional: true,
			Computed: true,
		}
	}
	return block
}

func TestSchemaMapCoreConfigSchema(t *testing.T) {
	tests := map[string]struct {
		Schema map[string]*Schema
		Want   *configschema.Block
	}{
		"empty": {
			map[string]*Schema{},
			testResource(&configschema.Block{}),
		},
		"primitives": {
			map[string]*Schema{
				"int": {
					Type:        TypeInt,
					Required:    true,
					Description: "foo bar baz",
				},
				"float": {
					Type:     TypeFloat,
					Optional: true,
				},
				"bool": {
					Type:     TypeBool,
					Computed: true,
				},
				"string": {
					Type:     TypeString,
					Optional: true,
					Computed: true,
				},
			},
			testResource(&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"int": {
						Type:        cty.Number,
						Required:    true,
						Description: "foo bar baz",
					},
					"float": {
						Type:     cty.Number,
						Optional: true,
					},
					"bool": {
						Type:     cty.Bool,
						Computed: true,
					},
					"string": {
						Type:     cty.String,
						Optional: true,
						Computed: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{},
			}),
		},
		"simple collections": {
			map[string]*Schema{
				"list": {
					Type:     TypeList,
					Required: true,
					Elem: &Schema{
						Type: TypeInt,
					},
				},
				"set": {
					Type:     TypeSet,
					Optional: true,
					Elem: &Schema{
						Type: TypeString,
					},
				},
				"map": {
					Type:     TypeMap,
					Optional: true,
					Elem: &Schema{
						Type: TypeBool,
					},
				},
				"map_default_type": {
					Type:     TypeMap,
					Optional: true,
					// Maps historically don't have elements because we
					// assumed they would be strings, so this needs to work
					// for pre-existing schemas.
				},
			},
			testResource(&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"list": {
						Type:     cty.List(cty.Number),
						Required: true,
					},
					"set": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
					"map": {
						Type:     cty.Map(cty.Bool),
						Optional: true,
					},
					"map_default_type": {
						Type:     cty.Map(cty.String),
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{},
			}),
		},
		"incorrectly-specified collections": {
			// Historically we tolerated setting a type directly as the Elem
			// attribute, rather than a Schema object. This is common enough
			// in existing provider code that we must support it as an alias
			// for a schema object with the given type.
			map[string]*Schema{
				"list": {
					Type:     TypeList,
					Required: true,
					Elem:     TypeInt,
				},
				"set": {
					Type:     TypeSet,
					Optional: true,
					Elem:     TypeString,
				},
				"map": {
					Type:     TypeMap,
					Optional: true,
					Elem:     TypeBool,
				},
			},
			testResource(&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"list": {
						Type:     cty.List(cty.Number),
						Required: true,
					},
					"set": {
						Type:     cty.Set(cty.String),
						Optional: true,
					},
					"map": {
						Type:     cty.Map(cty.Bool),
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{},
			}),
		},
		"sub-resource collections": {
			map[string]*Schema{
				"list": {
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{},
					},
					MinItems: 1,
					MaxItems: 2,
				},
				"set": {
					Type:     TypeSet,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{},
					},
				},
				"map": {
					Type:     TypeMap,
					Optional: true,
					Elem: &Resource{
						Schema: map[string]*Schema{},
					},
				},
			},
			testResource(&configschema.Block{
				Attributes: map[string]*configschema.Attribute{},
				BlockTypes: map[string]*configschema.NestedBlock{
					"list": {
						Nesting:  configschema.NestingList,
						Block:    configschema.Block{},
						MinItems: 1,
						MaxItems: 2,
					},
					"set": {
						Nesting:  configschema.NestingSet,
						Block:    configschema.Block{},
						MinItems: 1, // because schema is Required
					},
					"map": {
						Nesting: configschema.NestingMap,
						Block:   configschema.Block{},
					},
				},
			}),
		},
		"nested attributes and blocks": {
			map[string]*Schema{
				"foo": {
					Type:     TypeList,
					Required: true,
					Elem: &Resource{
						Schema: map[string]*Schema{
							"bar": {
								Type:     TypeList,
								Required: true,
								Elem: &Schema{
									Type: TypeList,
									Elem: &Schema{
										Type: TypeString,
									},
								},
							},
							"baz": {
								Type:     TypeSet,
								Optional: true,
								Elem: &Resource{
									Schema: map[string]*Schema{},
								},
							},
						},
					},
				},
			},
			testResource(&configschema.Block{
				Attributes: map[string]*configschema.Attribute{},
				BlockTypes: map[string]*configschema.NestedBlock{
					"foo": &configschema.NestedBlock{
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"bar": {
									Type:     cty.List(cty.List(cty.String)),
									Required: true,
								},
							},
							BlockTypes: map[string]*configschema.NestedBlock{
								"baz": {
									Nesting: configschema.NestingSet,
									Block:   configschema.Block{},
								},
							},
						},
						MinItems: 1, // because schema is Required
					},
				},
			}),
		},
		"sensitive": {
			map[string]*Schema{
				"string": {
					Type:      TypeString,
					Optional:  true,
					Sensitive: true,
				},
			},
			testResource(&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"string": {
						Type:      cty.String,
						Optional:  true,
						Sensitive: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{},
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := schemaMap(test.Schema).CoreConfigSchema()
			if !cmp.Equal(got, test.Want, typeComparer) {
				cmp.Diff(got, test.Want, typeComparer)
			}
		})
	}
}
