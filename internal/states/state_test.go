package states

import (
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
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
					Marks: cty.NewValueMarks(marks.Sensitive),
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

func TestState_MoveAbsResource(t *testing.T) {
	state := NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "foo",
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

	src := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "foo"}.Absolute(addrs.RootModuleInstance)
	dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "bar"}.Absolute(addrs.RootModuleInstance)

	state.MoveAbsResource(src, dst)

	if state.Empty() {
		t.Fatal("unexpected empty state")
	}

	if len(state.RootModule().Resources) != 1 {
		t.Fatalf("wrong number of resources in state; expected 1, found %d", len(state.RootModule().Resources))
	}

	got := state.Resource(dst)
	if got.Addr.Resource != dst.Resource {
		t.Fatalf("dst resource not in state")
	}
}

func TestState_MaybeMoveAbsResource(t *testing.T) {
	state := NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "foo",
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

	src := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "foo"}.Absolute(addrs.RootModuleInstance)
	dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "bar"}.Absolute(addrs.RootModuleInstance)

	// First move, success
	t.Run("first move", func(t *testing.T) {
		moved := state.MaybeMoveAbsResource(src, dst)
		if !moved {
			t.Fatal("wrong result")
		}
	})

	// Trying to move a resource that doesn't exist in state to a resource which does exist should be a noop.
	t.Run("noop", func(t *testing.T) {
		moved := state.MaybeMoveAbsResource(src, dst)
		if moved {
			t.Fatal("wrong result")
		}
	})
}

func TestState_MoveAbsResourceInstance(t *testing.T) {
	state := NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "foo",
		}.Instance(addrs.NoKey),
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
	// src resource from the state above
	src := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "foo"}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	t.Run("resource to resource instance", func(t *testing.T) {
		s := state.DeepCopy()
		// For a little extra fun, move a resource to a resource instance: test_thing.foo to test_thing.foo[1]
		dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "foo"}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance)

		s.MoveAbsResourceInstance(src, dst)

		if s.Empty() {
			t.Fatal("unexpected empty state")
		}

		if len(s.RootModule().Resources) != 1 {
			t.Fatalf("wrong number of resources in state; expected 1, found %d", len(state.RootModule().Resources))
		}

		got := s.ResourceInstance(dst)
		if got == nil {
			t.Fatalf("dst resource not in state")
		}
	})

	t.Run("move to new module", func(t *testing.T) {
		s := state.DeepCopy()
		// test_thing.foo to module.kinder.test_thing.foo["baz"]
		dstModule := addrs.RootModuleInstance.Child("kinder", addrs.NoKey)
		dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "foo"}.Instance(addrs.IntKey(1)).Absolute(dstModule)

		s.MoveAbsResourceInstance(src, dst)

		if s.Empty() {
			t.Fatal("unexpected empty state")
		}

		if s.Module(dstModule) == nil {
			t.Fatalf("child module %s not in state", dstModule.String())
		}

		if len(s.Module(dstModule).Resources) != 1 {
			t.Fatalf("wrong number of resources in state; expected 1, found %d", len(s.Module(dstModule).Resources))
		}

		got := s.ResourceInstance(dst)
		if got == nil {
			t.Fatalf("dst resource not in state")
		}
	})
}

func TestState_MaybeMoveAbsResourceInstance(t *testing.T) {
	state := NewState()
	rootModule := state.RootModule()
	rootModule.SetResourceInstanceCurrent(
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: "foo",
		}.Instance(addrs.NoKey),
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

	// For a little extra fun, let's go from a resource to a resource instance- test_thing.foo to test_thing.bar[1]
	src := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "foo"}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
	dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "foo"}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance)

	// First move, success
	t.Run("first move", func(t *testing.T) {
		moved := state.MaybeMoveAbsResourceInstance(src, dst)
		if !moved {
			t.Fatal("wrong result")
		}
		got := state.ResourceInstance(dst)
		if got == nil {
			t.Fatal("destination resource instance not in state")
		}
	})

	// Moving a resource instance that doesn't exist in state to a resource which does exist should be a noop.
	t.Run("noop", func(t *testing.T) {
		moved := state.MaybeMoveAbsResourceInstance(src, dst)
		if moved {
			t.Fatal("wrong result")
		}
	})
}
