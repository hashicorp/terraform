package states

import (
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
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
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)

	childModule := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	childModule.SetOutputValue("pizza", cty.StringVal("hawaiian"), false)
	multiModA := state.EnsureModule(addrs.RootModuleInstance.Child("multi", addrs.StringKey("a")))
	multiModA.SetOutputValue("pizza", cty.StringVal("cheese"), false)
	multiModB := state.EnsureModule(addrs.RootModuleInstance.Child("multi", addrs.StringKey("b")))
	multiModB.SetOutputValue("pizza", cty.StringVal("sausage"), false)

	want := &State{
		Modules: map[string]*Module{
			"": {
				Addr: addrs.RootModuleInstance,
				LocalValues: map[string]cty.Value{
					"foo": cty.StringVal("foo value"),
				},
				OutputValues: map[string]*OutputValue{
					"bar": {
						Addr: addrs.AbsOutputValue{
							OutputValue: addrs.OutputValue{
								Name: "bar",
							},
						},
						Value:     cty.StringVal("bar value"),
						Sensitive: false,
					},
					"secret": {
						Addr: addrs.AbsOutputValue{
							OutputValue: addrs.OutputValue{
								Name: "secret",
							},
						},
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
						}.Absolute(addrs.RootModuleInstance),

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
						ProviderConfig: addrs.AbsProviderConfig{
							Provider: addrs.NewDefaultProvider("test"),
							Module:   addrs.RootModule,
						},
					},
				},
			},
			"module.child": {
				Addr:        addrs.RootModuleInstance.Child("child", addrs.NoKey),
				LocalValues: map[string]cty.Value{},
				OutputValues: map[string]*OutputValue{
					"pizza": {
						Addr: addrs.AbsOutputValue{
							Module: addrs.RootModuleInstance.Child("child", addrs.NoKey),
							OutputValue: addrs.OutputValue{
								Name: "pizza",
							},
						},
						Value:     cty.StringVal("hawaiian"),
						Sensitive: false,
					},
				},
				Resources: map[string]*Resource{},
			},
			`module.multi["a"]`: {
				Addr:        addrs.RootModuleInstance.Child("multi", addrs.StringKey("a")),
				LocalValues: map[string]cty.Value{},
				OutputValues: map[string]*OutputValue{
					"pizza": {
						Addr: addrs.AbsOutputValue{
							Module: addrs.RootModuleInstance.Child("multi", addrs.StringKey("a")),
							OutputValue: addrs.OutputValue{
								Name: "pizza",
							},
						},
						Value:     cty.StringVal("cheese"),
						Sensitive: false,
					},
				},
				Resources: map[string]*Resource{},
			},
			`module.multi["b"]`: {
				Addr:        addrs.RootModuleInstance.Child("multi", addrs.StringKey("b")),
				LocalValues: map[string]cty.Value{},
				OutputValues: map[string]*OutputValue{
					"pizza": {
						Addr: addrs.AbsOutputValue{
							Module: addrs.RootModuleInstance.Child("multi", addrs.StringKey("b")),
							OutputValue: addrs.OutputValue{
								Name: "pizza",
							},
						},
						Value:     cty.StringVal("sausage"),
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

	expectedOutputs := map[string]string{
		`module.multi["a"].output.pizza`: "cheese",
		`module.multi["b"].output.pizza`: "sausage",
	}

	for _, o := range state.ModuleOutputs(addrs.RootModuleInstance, addrs.ModuleCall{Name: "multi"}) {
		addr := o.Addr.String()
		expected := expectedOutputs[addr]
		delete(expectedOutputs, addr)

		if expected != o.Value.AsString() {
			t.Fatalf("expected %q:%q, got %q", addr, expected, o.Value.AsString())
		}
	}

	for addr, o := range expectedOutputs {
		t.Fatalf("missing output %q:%q", addr, o)
	}
}

func TestStateDeepCopyObject(t *testing.T) {
	obj := &ResourceInstanceObject{
		Value: cty.ObjectVal(map[string]cty.Value{
			"id": cty.StringVal("id"),
		}),
		Private: []byte("private"),
		Status:  ObjectReady,
		Dependencies: []addrs.ConfigResource{
			{
				Module: addrs.RootModule,
				Resource: addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "test_instance",
					Name: "bar",
				},
			},
		},
		CreateBeforeDestroy: true,
	}

	objCopy := obj.DeepCopy()
	if !reflect.DeepEqual(obj, objCopy) {
		t.Fatalf("not equal\n%#v\n%#v", obj, objCopy)
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
			Status:              ObjectReady,
			SchemaVersion:       1,
			AttrsJSON:           []byte(`{"woozles":"confuzles"}`),
			Private:             []byte("private data"),
			Dependencies:        []addrs.ConfigResource{},
			CreateBeforeDestroy: true,
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
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
			// Sensitive path at "woozles"
			AttrSensitivePaths: []cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "woozles"}},
					Marks: cty.NewValueMarks("sensitive"),
				},
			},
			Private: []byte("private data"),
			Dependencies: []addrs.ConfigResource{
				{
					Module: addrs.RootModule,
					Resource: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "baz",
					},
				},
			},
		},
		addrs.AbsProviderConfig{
			Provider: addrs.NewDefaultProvider("test"),
			Module:   addrs.RootModule,
		},
	)

	childModule := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	childModule.SetOutputValue("pizza", cty.StringVal("hawaiian"), false)

	stateCopy := state.DeepCopy()
	if !state.Equal(stateCopy) {
		t.Fatalf("\nexpected:\n%q\ngot:\n%q\n", state, stateCopy)
	}
}
