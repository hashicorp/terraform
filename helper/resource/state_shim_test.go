package resource

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
)

// TestStateShim is meant to be a fairly comprehensive test, checking for dependencies, root outputs,
func TestStateShim(t *testing.T) {
	state := states.NewState()

	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}

	rootModule.SetOutputValue("bar", cty.ListVal([]cty.Value{cty.StringVal("bar"), cty.StringVal("value")}), false)
	rootModule.SetOutputValue("secret", cty.StringVal("secret value"), true)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "foo",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:        states.ObjectReady,
			AttrsFlat:     map[string]string{"id": "foo", "bazzle": "dazzle"},
			SchemaVersion: 7,
			DependsOn: []addrs.Referenceable{
				addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: 'M',
						Type: "test_thing",
						Name: "baz",
					},
				},
			},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "baz",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsFlat: map[string]string{"id": "baz", "bazzle": "dazzle"},
			DependsOn: []addrs.Referenceable{},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)

	childInstance := addrs.RootModuleInstance.Child("child", addrs.NoKey)
	childModule := state.EnsureModule(childInstance)
	childModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.DataResourceMode,
			Type: "test_data_thing",
			Name: "foo",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id": "bar", "fuzzle":"wuzzle"}`),
			DependsOn: []addrs.Referenceable{},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(childInstance),
	)
	childModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "baz",
		}.Instance(addrs.NoKey),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id": "bar", "fizzle":"wizzle"}`),
			DependsOn: []addrs.Referenceable{
				addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: 'D',
						Type: "test_data_thing",
						Name: "foo",
					},
				},
			},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(childInstance),
	)

	childModule.SetResourceInstanceDeposed(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "baz",
		}.Instance(addrs.NoKey),
		"00000001",
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsFlat: map[string]string{"id": "old", "fizzle": "wizzle"},
			DependsOn: []addrs.Referenceable{
				addrs.ResourceInstance{
					Resource: addrs.Resource{
						Mode: 'D',
						Type: "test_data_thing",
						Name: "foo",
					},
				},
			},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(childInstance),
	)

	childModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "lots",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsFlat: map[string]string{"id": "0", "bazzle": "dazzle"},
			DependsOn: []addrs.Referenceable{},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(childInstance),
	)
	childModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "lots",
		}.Instance(addrs.IntKey(1)),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectTainted,
			AttrsFlat: map[string]string{"id": "1", "bazzle": "dazzle"},
			DependsOn: []addrs.Referenceable{},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(childInstance),
	)

	childModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "single_count",
		}.Instance(addrs.IntKey(0)),
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id": "single", "bazzle":"dazzle"}`),
			DependsOn: []addrs.Referenceable{},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(childInstance),
	)

	expected := &terraform.State{
		Version: 3,
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Outputs: map[string]*terraform.OutputState{
					"bar": {
						Type:  "list",
						Value: []interface{}{"bar", "value"},
					},
					"secret": {
						Sensitive: true,
						Type:      "string",
						Value:     "secret value",
					},
				},
				Resources: map[string]*terraform.ResourceState{
					"test_thing.baz": &terraform.ResourceState{
						Type:     "test_thing",
						Provider: "provider.test",
						Primary: &terraform.InstanceState{
							ID: "baz",
							Attributes: map[string]string{
								"id":     "baz",
								"bazzle": "dazzle",
							},
						},
					},
					"test_thing.foo": &terraform.ResourceState{
						Type:     "test_thing",
						Provider: "provider.test",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"id":     "foo",
								"bazzle": "dazzle",
							},
							Meta: map[string]interface{}{
								"schema_version": 7,
							},
						},
						Dependencies: []string{"test_thing.baz"},
					},
				},
			},
			&terraform.ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*terraform.ResourceState{
					"test_thing.baz": &terraform.ResourceState{
						Type:     "test_thing",
						Provider: "module.child.provider.test",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"id":     "bar",
								"fizzle": "wizzle",
							},
						},
						Deposed: []*terraform.InstanceState{
							{
								ID: "old",
								Attributes: map[string]string{
									"id":     "old",
									"fizzle": "wizzle",
								},
							},
						},
						Dependencies: []string{"data.test_data_thing.foo"},
					},
					"data.test_data_thing.foo": &terraform.ResourceState{
						Type:     "test_data_thing",
						Provider: "module.child.provider.test",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"id":     "bar",
								"fuzzle": "wuzzle",
							},
						},
					},
					"test_thing.lots.0": &terraform.ResourceState{
						Type:     "test_thing",
						Provider: "module.child.provider.test",
						Primary: &terraform.InstanceState{
							ID: "0",
							Attributes: map[string]string{
								"id":     "0",
								"bazzle": "dazzle",
							},
						},
					},
					"test_thing.lots.1": &terraform.ResourceState{
						Type:     "test_thing",
						Provider: "module.child.provider.test",
						Primary: &terraform.InstanceState{
							ID: "1",
							Attributes: map[string]string{
								"id":     "1",
								"bazzle": "dazzle",
							},
							Tainted: true,
						},
					},
					"test_thing.single_count": &terraform.ResourceState{
						Type:     "test_thing",
						Provider: "module.child.provider.test",
						Primary: &terraform.InstanceState{
							ID: "single",
							Attributes: map[string]string{
								"id":     "single",
								"bazzle": "dazzle",
							},
						},
					},
				},
			},
		},
	}

	providers := map[string]terraform.ResourceProvider{
		"test": &schema.Provider{
			ResourcesMap: map[string]*schema.Resource{
				"test_thing": &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":     {Type: schema.TypeString, Computed: true},
						"fizzle": {Type: schema.TypeString, Optional: true},
						"bazzle": {Type: schema.TypeString, Optional: true},
					},
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"test_data_thing": &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":     {Type: schema.TypeString, Computed: true},
						"fuzzle": {Type: schema.TypeString, Optional: true},
					},
				},
			},
		},
	}

	shimmed, err := shimNewState(state, providers)
	if err != nil {
		t.Fatal(err)
	}

	if !expected.Equal(shimmed) {
		t.Fatalf("wrong result state\ngot:\n%s\n\nwant:\n%s", expected, shimmed)
	}
}
