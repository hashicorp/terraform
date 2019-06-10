package states

import (
	"testing"

	"github.com/go-test/deep"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
)

func TestState(t *testing.T) {
	// This basic tests exercises the main mutation methods to construct
	// a state. It is not fully comprehensive, so other tests should visit
	// more esoteric codepaths.

	state := NewState()

	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}

	rootModule.SetLocalValue("foo", cty.StringVal("foo value"))
	rootModule.SetOutputValue("bar", cty.StringVal("bar value"), false)
	rootModule.SetOutputValue("secret", cty.StringVal("secret value"), true)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		&ResourceInstanceObjectSrc{
			Status:        ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)

	childModule := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	childModule.SetOutputValue("pizza", cty.StringVal("hawaiian"), false)

	want := &State{
		Modules: map[string]*Module{
			"": {
				Addr: addrs.RootModuleInstance,
				LocalValues: map[string]cty.Value{
					"foo": cty.StringVal("foo value"),
				},
				OutputValues: map[string]*OutputValue{
					"bar": {
						Value:     cty.StringVal("bar value"),
						Sensitive: false,
					},
					"secret": {
						Value:     cty.StringVal("secret value"),
						Sensitive: true,
					},
				},
				Resources: map[string]*Resource{
					"test_thing.baz": {
						Addr: addrs.Resource{
							Mode: addrs.ManagedResourceMode,
							Type: "test_thing",
							Name: "baz",
						},
						EachMode: EachList,
						Instances: map[addrs.InstanceKey]*ResourceInstance{
							addrs.IntKey(0): {
								Current: &ResourceInstanceObjectSrc{
									SchemaVersion: 1,
									Status:        ObjectReady,
									AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
								},
								Deposed: map[DeposedKey]*ResourceInstanceObjectSrc{},
							},
						},
						ProviderConfig: addrs.ProviderConfig{
							Type: "test",
						}.Absolute(addrs.RootModuleInstance),
					},
				},
			},
			"module.child": {
				Addr:        addrs.RootModuleInstance.Child("child", addrs.NoKey),
				LocalValues: map[string]cty.Value{},
				OutputValues: map[string]*OutputValue{
					"pizza": {
						Value:     cty.StringVal("hawaiian"),
						Sensitive: false,
					},
				},
				Resources: map[string]*Resource{},
			},
		},
	}

	{
		// Our structure goes deep, so we need to temporarily override the
		// deep package settings to ensure that we visit the full structure.
		oldDeepDepth := deep.MaxDepth
		oldDeepCompareUnexp := deep.CompareUnexportedFields
		deep.MaxDepth = 50
		deep.CompareUnexportedFields = true
		defer func() {
			deep.MaxDepth = oldDeepDepth
			deep.CompareUnexportedFields = oldDeepCompareUnexp
		}()
	}

	for _, problem := range deep.Equal(state, want) {
		t.Error(problem)
	}
}

func TestStateDeepCopy(t *testing.T) {
	state := NewState()

	rootModule := state.RootModule()
	if rootModule == nil {
		t.Errorf("root module is nil; want valid object")
	}

	rootModule.SetLocalValue("foo", cty.StringVal("foo value"))
	rootModule.SetOutputValue("bar", cty.StringVal("bar value"), false)
	rootModule.SetOutputValue("secret", cty.StringVal("secret value"), true)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "baz",
		}.Instance(addrs.IntKey(0)),
		&ResourceInstanceObjectSrc{
			Status:        ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
			Private:       []byte("private data"),
			Dependencies:  []addrs.Referenceable{},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "bar",
		}.Instance(addrs.IntKey(0)),
		&ResourceInstanceObjectSrc{
			Status:        ObjectReady,
			SchemaVersion: 1,
			AttrsJSON:     []byte(`{"woozles":"confuzles"}`),
			Private:       []byte("private data"),
			Dependencies: []addrs.Referenceable{addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "baz",
			}},
		},
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	)

	childModule := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	childModule.SetOutputValue("pizza", cty.StringVal("hawaiian"), false)

	stateCopy := state.DeepCopy()
	if !state.Equal(stateCopy) {
		t.Fatalf("\nexpected:\n%q\ngot:\n%q\n", state, stateCopy)
	}
}
