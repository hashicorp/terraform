package schema

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/terraform/config/configschema"
)

func TestSchemaMapCoreConfigSchema(t *testing.T) {
	tests := map[string]struct {
		Schema map[string]*Schema
		Want   *configschema.Block
	}{
		"empty": {
			map[string]*Schema{},
			&configschema.Block{},
		},
		"primitives": {
			map[string]*Schema{
				"int": {
					Type:     TypeInt,
					Required: true,
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
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"int": {
						Type:     cty.Number,
						Required: true,
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
			},
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
			&configschema.Block{
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
			},
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
			&configschema.Block{
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
			},
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
			&configschema.Block{
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
			},
		},
		"sensitive": {
			map[string]*Schema{
				"string": {
					Type:      TypeString,
					Optional:  true,
					Sensitive: true,
				},
			},
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"string": {
						Type:      cty.String,
						Optional:  true,
						Sensitive: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := schemaMap(test.Schema).CoreConfigSchema()
			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(test.Want))
			}
		})
	}
}
