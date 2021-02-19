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

func TestMarshalResources(t *testing.T) {
	deposedKey := states.NewDeposedKey()
	tests := map[string]struct {
		Resources map[string]*states.Resource
		Want      []resource
		Err       bool
	}{
		"nil": {
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
			[]resource{
				resource{
					Address:         "test_thing.bar",
					Mode:            "managed",
					Type:            "test_thing",
					Name:            "bar",
					Index:           addrs.InstanceKey(nil),
					ProviderName:    "registry.terraform.io/hashicorp/test",
					SchemaVersion:   1,
					AttributeValues: json.RawMessage(`{"woozles":"confuzles"}`),
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
			[]resource{
				resource{
					Address:         "test_thing.bar[0]",
					Mode:            "managed",
					Type:            "test_thing",
					Name:            "bar",
					Index:           addrs.IntKey(0),
					ProviderName:    "registry.terraform.io/hashicorp/test",
					SchemaVersion:   1,
					AttributeValues: json.RawMessage(`{"woozles":"confuzles"}`),
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
			[]resource{
				resource{
					Address:         "test_thing.bar[\"rockhopper\"]",
					Mode:            "managed",
					Type:            "test_thing",
					Name:            "bar",
					Index:           addrs.StringKey("rockhopper"),
					ProviderName:    "registry.terraform.io/hashicorp/test",
					SchemaVersion:   1,
					AttributeValues: json.RawMessage(`{"woozles":"confuzles"}`),
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
			[]resource{
				resource{
					Address:         "test_thing.bar",
					Mode:            "managed",
					Type:            "test_thing",
					Name:            "bar",
					Index:           addrs.InstanceKey(nil),
					ProviderName:    "registry.terraform.io/hashicorp/test",
					DeposedKey:      deposedKey.String(),
					AttributeValues: json.RawMessage(`{"woozles":"confuzles"}`),
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
			[]resource{
				resource{
					Address:         "test_thing.bar",
					Mode:            "managed",
					Type:            "test_thing",
					Name:            "bar",
					Index:           addrs.InstanceKey(nil),
					ProviderName:    "registry.terraform.io/hashicorp/test",
					SchemaVersion:   1,
					AttributeValues: json.RawMessage(`{"woozles":"confuzles"}`),
				},
				resource{
					Address:         "test_thing.bar",
					Mode:            "managed",
					Type:            "test_thing",
					Name:            "bar",
					Index:           addrs.InstanceKey(nil),
					ProviderName:    "registry.terraform.io/hashicorp/test",
					DeposedKey:      deposedKey.String(),
					AttributeValues: json.RawMessage(`{"woozles":"confuzles"}`),
				},
			},
			false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := marshalResources(test.Resources, addrs.RootModuleInstance)
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

	got, err := marshalModules(testState, moduleMap[""], moduleMap)

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

	got, err := marshalModules(testState, moduleMap[""], moduleMap)

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

func TestMarshalModules_parent_no_resources(t *testing.T) {
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
	got, err := marshalRootModule(testState)

	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if len(got.ChildModules) != 1 {
		t.Fatalf("wrong result! got %d modules, expected 1", len(got.ChildModules))
	}

	if got.ChildModules[0].Address != "module.child" {
		t.Fatalf("wrong result! got %#v\n", got)
	}

	if got.ChildModules[0].ChildModules[0].Address != "module.child.module.submodule" {
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
