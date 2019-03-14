package jsonprovider

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
)

func TestMarshalProvider(t *testing.T) {
	tests := []struct {
		Input *terraform.ProviderSchema
		Want  *Provider
	}{
		{
			nil,
			&Provider{},
		},
		{
			testProvider(),
			&Provider{
				Provider: &schema{
					Block: &block{
						Attributes: map[string]*attribute{
							"region": {
								AttributeType: json.RawMessage(`"string"`),
								Required:      true,
							},
						},
					},
				},
				ResourceSchemas: map[string]*schema{
					"test_instance": {
						Version: 42,
						Block: &block{
							Attributes: map[string]*attribute{
								"id": {
									AttributeType: json.RawMessage(`"string"`),
									Optional:      true,
									Computed:      true,
								},
								"ami": {
									AttributeType: json.RawMessage(`"string"`),
									Optional:      true,
								},
							},
							BlockTypes: map[string]*blockType{
								"network_interface": {
									Block: &block{
										Attributes: map[string]*attribute{
											"device_index": {
												AttributeType: json.RawMessage(`"string"`),
												Optional:      true,
											},
											"description": {
												AttributeType: json.RawMessage(`"string"`),
												Optional:      true,
											},
										},
									},
									NestingMode: "list",
								},
							},
						},
					},
				},
				DataSourceSchemas: map[string]*schema{
					"test_data_source": {
						Version: 3,
						Block: &block{
							Attributes: map[string]*attribute{
								"id": {
									AttributeType: json.RawMessage(`"string"`),
									Optional:      true,
									Computed:      true,
								},
								"ami": {
									AttributeType: json.RawMessage(`"string"`),
									Optional:      true,
								},
							},
							BlockTypes: map[string]*blockType{
								"network_interface": {
									Block: &block{
										Attributes: map[string]*attribute{
											"device_index": {
												AttributeType: json.RawMessage(`"string"`),
												Optional:      true,
											},
											"description": {
												AttributeType: json.RawMessage(`"string"`),
												Optional:      true,
											},
										},
									},
									NestingMode: "list",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		got := marshalProvider(test.Input)
		if !cmp.Equal(got, test.Want) {
			t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, test.Want))
		}
	}
}

func testProviders() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[string]*terraform.ProviderSchema{
			"test": testProvider(),
		},
	}
}

func testProvider() *terraform.ProviderSchema {
	return &terraform.ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"region": {Type: cty.String, Required: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"network_interface": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"device_index": {Type: cty.String, Optional: true},
								"description":  {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"id":  {Type: cty.String, Optional: true, Computed: true},
					"ami": {Type: cty.String, Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"network_interface": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"device_index": {Type: cty.String, Optional: true},
								"description":  {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
		},

		ResourceTypeSchemaVersions: map[string]uint64{
			"test_instance":    42,
			"test_data_source": 3,
		},
	}
}
