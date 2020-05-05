package jsonstate

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
)

func TestMarshalOutputs(t *testing.T) {
	tests := []struct {
		Outputs map[string]*states.OutputValue
		Want    map[string]output
		Err     bool
	}{
		{
			nil,
			nil,
			false,
		},
		{
			map[string]*states.OutputValue{
				"test": {
					Sensitive: true,
					Value:     cty.StringVal("sekret"),
				},
			},
			map[string]output{
				"test": {
					Sensitive: true,
					Value:     json.RawMessage(`"sekret"`),
				},
			},
			false,
		},
		{
			map[string]*states.OutputValue{
				"test": {
					Sensitive: false,
					Value:     cty.StringVal("not_so_sekret"),
				},
			},
			map[string]output{
				"test": {
					Sensitive: false,
					Value:     json.RawMessage(`"not_so_sekret"`),
				},
			},
			false,
		},
	}

	for _, test := range tests {
		got, err := marshalOutputs(test.Outputs)
		if test.Err {
			if err == nil {
				t.Fatal("succeeded; want error")
			}
			return
		} else if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		eq := reflect.DeepEqual(got, test.Want)
		if !eq {
			// printing the output isn't terribly useful, but it does help indicate which test case failed
			t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}

func TestMarshalAttributeValues(t *testing.T) {
	tests := []struct {
		Attr   cty.Value
		Schema *configschema.Block
		Want   attributeValues
	}{
		{
			cty.NilVal,
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			nil,
		},
		{
			cty.NullVal(cty.String),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			nil,
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
			}),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			attributeValues{"foo": json.RawMessage(`"bar"`)},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.NullVal(cty.String),
			}),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			attributeValues{"foo": json.RawMessage(`null`)},
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"bar": cty.MapVal(map[string]cty.Value{
					"hello": cty.StringVal("world"),
				}),
				"baz": cty.ListVal([]cty.Value{
					cty.StringVal("goodnight"),
					cty.StringVal("moon"),
				}),
			}),
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"bar": {
						Type:     cty.Map(cty.String),
						Required: true,
					},
					"baz": {
						Type:     cty.List(cty.String),
						Optional: true,
					},
				},
			},
			attributeValues{
				"bar": json.RawMessage(`{"hello":"world"}`),
				"baz": json.RawMessage(`["goodnight","moon"]`),
			},
		},
	}

	for _, test := range tests {
		got := marshalAttributeValues(test.Attr, test.Schema)
		eq := reflect.DeepEqual(got, test.Want)
		if !eq {
			t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}

func TestMarshalResources(t *testing.T) {
	deposedKey := states.NewDeposedKey()
	tests := map[string]struct {
		Resources map[string]*states.Resource
		Schemas   *terraform.Schemas
		Want      []resource
		Err       bool
	}{
		"nil": {
			nil,
			nil,
			nil,
			false,
		},
		"single resource": {
			map[string]*states.Resource{
				"test_thing.baz": {
					Addr: addrs.AbsResource{
						Resource: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "bar",
						},
					},
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.NoKey: {
							Current: &states.ResourceInstanceObjectSrc{
								SchemaVersion: 1,
								Status:        states.ObjectReady,
								AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
							},
						},
					},
					ProviderConfig: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:       "test_thing.bar",
					Mode:          "managed",
					Type:          "test_thing",
					Name:          "bar",
					Index:         addrs.InstanceKey(nil),
					ProviderName:  "registry.terraform.io/hashicorp/test",
					SchemaVersion: 1,
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
		"resource with count": {
			map[string]*states.Resource{
				"test_thing.bar": {
					Addr: addrs.AbsResource{
						Resource: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "bar",
						},
					},
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.IntKey(0): {
							Current: &states.ResourceInstanceObjectSrc{
								SchemaVersion: 1,
								Status:        states.ObjectReady,
								AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
							},
						},
					},
					ProviderConfig: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:       "test_thing.bar[0]",
					Mode:          "managed",
					Type:          "test_thing",
					Name:          "bar",
					Index:         addrs.IntKey(0),
					ProviderName:  "registry.terraform.io/hashicorp/test",
					SchemaVersion: 1,
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
		"resource with for_each": {
			map[string]*states.Resource{
				"test_thing.bar": {
					Addr: addrs.AbsResource{
						Resource: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "bar",
						},
					},
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.StringKey("rockhopper"): {
							Current: &states.ResourceInstanceObjectSrc{
								SchemaVersion: 1,
								Status:        states.ObjectReady,
								AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
							},
						},
					},
					ProviderConfig: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:       "test_thing.bar[\"rockhopper\"]",
					Mode:          "managed",
					Type:          "test_thing",
					Name:          "bar",
					Index:         addrs.StringKey("rockhopper"),
					ProviderName:  "registry.terraform.io/hashicorp/test",
					SchemaVersion: 1,
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
		"deposed resource": {
			map[string]*states.Resource{
				"test_thing.baz": {
					Addr: addrs.AbsResource{
						Resource: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "bar",
						},
					},
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.NoKey: {
							Deposed: map[states.DeposedKey]*states.ResourceInstanceObjectSrc{
								states.DeposedKey(deposedKey): &states.ResourceInstanceObjectSrc{
									SchemaVersion: 1,
									Status:        states.ObjectReady,
									AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
								},
							},
						},
					},
					ProviderConfig: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:      "test_thing.bar",
					Mode:         "managed",
					Type:         "test_thing",
					Name:         "bar",
					Index:        addrs.InstanceKey(nil),
					ProviderName: "registry.terraform.io/hashicorp/test",
					DeposedKey:   deposedKey.String(),
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
		"deposed and current resource": {
			map[string]*states.Resource{
				"test_thing.baz": {
					Addr: addrs.AbsResource{
						Resource: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "bar",
						},
					},
					Instances: map[addrs.InstanceKey]*states.ResourceInstance{
						addrs.NoKey: {
							Deposed: map[states.DeposedKey]*states.ResourceInstanceObjectSrc{
								states.DeposedKey(deposedKey): &states.ResourceInstanceObjectSrc{
									SchemaVersion: 1,
									Status:        states.ObjectReady,
									AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
								},
							},
							Current: &states.ResourceInstanceObjectSrc{
								SchemaVersion: 1,
								Status:        states.ObjectReady,
								AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
							},
						},
					},
					ProviderConfig: addrs.AbsProviderConfig{
						Provider: addrs.NewDefaultProvider("test"),
						Module:   addrs.RootModule,
					},
				},
			},
			testSchemas(),
			[]resource{
				resource{
					Address:       "test_thing.bar",
					Mode:          "managed",
					Type:          "test_thing",
					Name:          "bar",
					Index:         addrs.InstanceKey(nil),
					ProviderName:  "registry.terraform.io/hashicorp/test",
					SchemaVersion: 1,
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
				resource{
					Address:      "test_thing.bar",
					Mode:         "managed",
					Type:         "test_thing",
					Name:         "bar",
					Index:        addrs.InstanceKey(nil),
					ProviderName: "registry.terraform.io/hashicorp/test",
					DeposedKey:   deposedKey.String(),
					AttributeValues: attributeValues{
						"foozles": json.RawMessage(`null`),
						"woozles": json.RawMessage(`"confuzles"`),
					},
				},
			},
			false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := marshalResources(test.Resources, addrs.RootModuleInstance, test.Schemas)
			if test.Err {
				if err == nil {
					t.Fatal("succeeded; want error")
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			eq := reflect.DeepEqual(got, test.Want)
			if !eq {
				t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
			}
		})
	}
}

func TestMarshalModules_basic(t *testing.T) {
	childModule, _ := addrs.ParseModuleInstanceStr("module.child")
	subModule, _ := addrs.ParseModuleInstanceStr("module.submodule")
	testState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(childModule),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   childModule.Module(),
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(subModule),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   subModule.Module(),
			},
		)
	})
	moduleMap := make(map[string][]addrs.ModuleInstance)
	moduleMap[""] = []addrs.ModuleInstance{childModule, subModule}

	got, err := marshalModules(testState, testSchemas(), moduleMap[""], moduleMap)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if len(got) != 2 {
		t.Fatalf("wrong result! got %d modules, expected 2", len(got))
	}

	if got[0].Address != "module.child" || got[1].Address != "module.submodule" {
		t.Fatalf("wrong result! got %#v\n", got)
	}

}

func TestMarshalModules_nested(t *testing.T) {
	childModule, _ := addrs.ParseModuleInstanceStr("module.child")
	subModule, _ := addrs.ParseModuleInstanceStr("module.child.module.submodule")
	testState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(childModule),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   childModule.Module(),
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(subModule),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo","foo":"value","bar":"value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   subModule.Module(),
			},
		)
	})
	moduleMap := make(map[string][]addrs.ModuleInstance)
	moduleMap[""] = []addrs.ModuleInstance{childModule}
	moduleMap[childModule.String()] = []addrs.ModuleInstance{subModule}

	got, err := marshalModules(testState, testSchemas(), moduleMap[""], moduleMap)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if len(got) != 1 {
		t.Fatalf("wrong result! got %d modules, expected 1", len(got))
	}

	if got[0].Address != "module.child" {
		t.Fatalf("wrong result! got %#v\n", got)
	}

	if got[0].ChildModules[0].Address != "module.child.module.submodule" {
		t.Fatalf("wrong result! got %#v\n", got)
	}
}

func testSchemas() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[addrs.Provider]*terraform.ProviderSchema{
			addrs.NewDefaultProvider("test"): &terraform.ProviderSchema{
				ResourceTypes: map[string]*configschema.Block{
					"test_thing": {
						Attributes: map[string]*configschema.Attribute{
							"woozles": {Type: cty.String, Optional: true, Computed: true},
							"foozles": {Type: cty.String, Optional: true},
						},
					},
					"test_instance": {
						Attributes: map[string]*configschema.Attribute{
							"id":  {Type: cty.String, Optional: true, Computed: true},
							"foo": {Type: cty.String, Optional: true},
							"bar": {Type: cty.String, Optional: true},
						},
					},
				},
			},
		},
	}
}
