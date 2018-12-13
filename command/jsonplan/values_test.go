package jsonplan

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"

	"github.com/zclconf/go-cty/cty"
)

func TestMarshalStateModules(t *testing.T) {
	tests := []struct {
		Modules map[string]*states.Module
		Want    []module
		Err     bool
	}{
		{
			map[string]*states.Module{
				"root": &states.Module{
					Addr: addrs.RootModuleInstance,
					Resources: map[string]*states.Resource{
						"test-resource": &states.Resource{},
					},
				},
				"module": &states.Module{
					Addr: addrs.RootModuleInstance.Child("child", addrs.NoKey),
					Resources: map[string]*states.Resource{
						"test-module-resource": &states.Resource{},
					},
				},
			},
			[]module{
				module{
					Address: "",
					Resources: []resource{
						resource{Name: "test-resource"},
					},
					ChildModules: []module{
						module{
							Address: "modules/child",
							Resources: []resource{
								resource{Name: "test-module-resource"},
							},
						},
					},
				},
			},
			false,
		},
	}

	for _, test := range tests {
		got, err := marshalStateModules(test.Modules, testSchemas())

		if test.Err {
			if err == nil {
				t.Fatal("succeeded; want error")
			}
			return
		} else if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// TODO: write an actual comparison function if needed
		// (implementing proper sorting will help)
		if !reflect.DeepEqual(got, test.Want) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
		}
	}
}

func testProvider() *terraform.MockProvider {
	p := new(terraform.MockProvider)
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}

	p.GetSchemaReturn = testProviderSchema()

	return p
}

func testProviderSchema() *terraform.ProviderSchema {
	return &terraform.ProviderSchema{
		Provider: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"region": {Type: cty.String, Optional: true},
			},
		},
		ResourceTypes: map[string]*configschema.Block{
			"test_resource": {
				Attributes: map[string]*configschema.Attribute{
					"id":      {Type: cty.String, Computed: true},
					"foo":     {Type: cty.String, Optional: true},
					"woozles": {Type: cty.String, Optional: true},
				},
			},
		},
		DataSources: map[string]*configschema.Block{
			"test_data_source": {
				Attributes: map[string]*configschema.Attribute{
					"compute": {Type: cty.String, Optional: true},
					"value":   {Type: cty.String, Computed: true},
				},
			},
		},
	}
}

func testSchemas() *terraform.Schemas {
	provider := testProvider()
	return &terraform.Schemas{
		Providers: map[string]*terraform.ProviderSchema{
			"test": provider.GetSchemaReturn,
		},
	}
}
