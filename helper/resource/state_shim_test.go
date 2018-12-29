package resource

import (
	"testing"

	"github.com/hashicorp/terraform/configs/configschema"

	"github.com/hashicorp/terraform/addrs"
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
			Status:    states.ObjectReady,
			AttrsFlat: map[string]string{"id": "foo", "bazzle": "dazzle"},
			Dependencies: []addrs.Referenceable{
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
			Status:       states.ObjectReady,
			AttrsFlat:    map[string]string{"id": "baz", "bazzle": "dazzle"},
			Dependencies: []addrs.Referenceable{},
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
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id": "bar", "fuzzle":"wuzzle"}`),
			Dependencies: []addrs.Referenceable{},
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
			Dependencies: []addrs.Referenceable{
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
			Dependencies: []addrs.Referenceable{
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
			Status:       states.ObjectReady,
			AttrsFlat:    map[string]string{"id": "0", "bazzle": "dazzle"},
			Dependencies: []addrs.Referenceable{},
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
			Status:       states.ObjectTainted,
			AttrsFlat:    map[string]string{"id": "1", "bazzle": "dazzle"},
			Dependencies: []addrs.Referenceable{},
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
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id": "single", "bazzle":"dazzle"}`),
			Dependencies: []addrs.Referenceable{},
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
						Type: "test_thing",
						Primary: &terraform.InstanceState{
							ID: "baz",
							Attributes: map[string]string{
								"id":     "baz",
								"bazzle": "dazzle",
							},
						},
					},
					"test_thing.foo": &terraform.ResourceState{
						Type: "test_thing",
						Primary: &terraform.InstanceState{
							ID: "foo",
							Attributes: map[string]string{
								"id":     "foo",
								"bazzle": "dazzle",
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
						Type: "test_thing",
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
						Type: "test_data_thing",
						Primary: &terraform.InstanceState{
							ID: "bar",
							Attributes: map[string]string{
								"id":     "bar",
								"fuzzle": "wuzzle",
							},
						},
					},
					"test_thing.lots.0": &terraform.ResourceState{
						Type: "test_thing",
						Primary: &terraform.InstanceState{
							ID: "0",
							Attributes: map[string]string{
								"id":     "0",
								"bazzle": "dazzle",
							},
						},
					},
					"test_thing.lots.1": &terraform.ResourceState{
						Type: "test_thing",
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
						Type: "test_thing",
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

	schemas := &terraform.Schemas{
		Providers: map[string]*terraform.ProviderSchema{
			"test": {
				ResourceTypes: map[string]*configschema.Block{
					"test_thing": &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"id": {
								Type:     cty.String,
								Computed: true,
							},
							"fizzle": {
								Type:     cty.String,
								Optional: true,
							},
							"bazzle": {
								Type:     cty.String,
								Optional: true,
							},
						},
					},
				},
				DataSources: map[string]*configschema.Block{
					"test_data_thing": &configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"id": {
								Type:     cty.String,
								Computed: true,
							},
							"fuzzle": {
								Type:     cty.String,
								Optional: true,
							},
						},
					},
				},
			},
		},
	}

	shimmed, err := shimNewState(state, schemas)
	if err != nil {
		t.Fatal(err)
	}

	if !expected.Equal(shimmed) {
		t.Fatalf("\nexpected state:\n%s\n\ngot state:\n%s", expected, shimmed)
	}
}
