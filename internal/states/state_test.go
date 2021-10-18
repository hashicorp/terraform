package states

import (
	"fmt"
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

func TestStateHasResourceInstanceObjects(t *testing.T) {
	providerConfig := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.MustParseProviderSourceString("test/test"),
	}
	childModuleProviderConfig := addrs.AbsProviderConfig{
		Module:   addrs.RootModule.Child("child"),
		Provider: addrs.MustParseProviderSourceString("test/test"),
	}

	tests := map[string]struct {
		Setup func(ss *SyncState)
		Want  bool
	}{
		"empty": {
			func(ss *SyncState) {},
			false,
		},
		"one current, ready object in root module": {
			func(ss *SyncState) {
				ss.SetResourceInstanceCurrent(
					mustAbsResourceAddr("test.foo").Instance(addrs.NoKey),
					&ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    ObjectReady,
					},
					providerConfig,
				)
			},
			true,
		},
		"one current, ready object in child module": {
			func(ss *SyncState) {
				ss.SetResourceInstanceCurrent(
					mustAbsResourceAddr("module.child.test.foo").Instance(addrs.NoKey),
					&ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    ObjectReady,
					},
					childModuleProviderConfig,
				)
			},
			true,
		},
		"one current, tainted object in root module": {
			func(ss *SyncState) {
				ss.SetResourceInstanceCurrent(
					mustAbsResourceAddr("test.foo").Instance(addrs.NoKey),
					&ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    ObjectTainted,
					},
					providerConfig,
				)
			},
			true,
		},
		"one deposed, ready object in root module": {
			func(ss *SyncState) {
				ss.SetResourceInstanceDeposed(
					mustAbsResourceAddr("test.foo").Instance(addrs.NoKey),
					DeposedKey("uhoh"),
					&ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    ObjectTainted,
					},
					providerConfig,
				)
			},
			true,
		},
		"one empty resource husk in root module": {
			func(ss *SyncState) {
				// Current Terraform doesn't actually create resource husks
				// as part of its everyday work, so this is a "should never
				// happen" case but we'll test to make sure we're robust to
				// it anyway, because this was a historical bug blocking
				// "terraform workspace delete" and similar.
				ss.SetResourceInstanceCurrent(
					mustAbsResourceAddr("test.foo").Instance(addrs.NoKey),
					&ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    ObjectTainted,
					},
					providerConfig,
				)
				s := ss.Lock()
				delete(s.Modules[""].Resources["test.foo"].Instances, addrs.NoKey)
				ss.Unlock()
			},
			false,
		},
		"one current data resource object in root module": {
			func(ss *SyncState) {
				ss.SetResourceInstanceCurrent(
					mustAbsResourceAddr("data.test.foo").Instance(addrs.NoKey),
					&ResourceInstanceObjectSrc{
						AttrsJSON: []byte(`{}`),
						Status:    ObjectReady,
					},
					providerConfig,
				)
			},
			false, // data resources aren't managed resources, so they don't count
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			state := BuildState(test.Setup)
			got := state.HasManagedResourceInstanceObjects()
			if got != test.Want {
				t.Errorf("wrong result\nstate content: (using legacy state string format; might not be comprehensive)\n%s\n\ngot:  %t\nwant: %t", state, got, test.Want)
			}
		})
	}

}

func TestState_MoveAbsResource(t *testing.T) {
	// Set up a starter state for the embedded tests, which should start from a copy of this state.
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

	t.Run("basic move", func(t *testing.T) {
		s := state.DeepCopy()
		dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "bar"}.Absolute(addrs.RootModuleInstance)

		s.MoveAbsResource(src, dst)

		if s.Empty() {
			t.Fatal("unexpected empty state")
		}

		if len(s.RootModule().Resources) != 1 {
			t.Fatalf("wrong number of resources in state; expected 1, found %d", len(state.RootModule().Resources))
		}

		got := s.Resource(dst)
		if got.Addr.Resource != dst.Resource {
			t.Fatalf("dst resource not in state")
		}
	})

	t.Run("move to new module", func(t *testing.T) {
		s := state.DeepCopy()
		dstModule := addrs.RootModuleInstance.Child("kinder", addrs.StringKey("one"))
		dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "bar"}.Absolute(dstModule)

		s.MoveAbsResource(src, dst)

		if s.Empty() {
			t.Fatal("unexpected empty state")
		}

		if s.Module(dstModule) == nil {
			t.Fatalf("child module %s not in state", dstModule.String())
		}

		if len(s.Module(dstModule).Resources) != 1 {
			t.Fatalf("wrong number of resources in state; expected 1, found %d", len(s.Module(dstModule).Resources))
		}

		got := s.Resource(dst)
		if got.Addr.Resource != dst.Resource {
			t.Fatalf("dst resource not in state")
		}
	})

	t.Run("from a child module to root", func(t *testing.T) {
		s := state.DeepCopy()
		srcModule := addrs.RootModuleInstance.Child("kinder", addrs.NoKey)
		cm := s.EnsureModule(srcModule)
		cm.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "child",
			}.Instance(addrs.IntKey(0)), // Moving the AbsResouce moves all instances
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
		cm.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "child",
			}.Instance(addrs.IntKey(1)), // Moving the AbsResouce moves all instances
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

		src := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "child"}.Absolute(srcModule)
		dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "child"}.Absolute(addrs.RootModuleInstance)
		s.MoveAbsResource(src, dst)

		if s.Empty() {
			t.Fatal("unexpected empty state")
		}

		// The child module should have been removed after removing its only resource
		if s.Module(srcModule) != nil {
			t.Fatalf("child module %s was not removed from state after mv", srcModule.String())
		}

		if len(s.RootModule().Resources) != 2 {
			t.Fatalf("wrong number of resources in state; expected 2, found %d", len(s.RootModule().Resources))
		}

		if len(s.Resource(dst).Instances) != 2 {
			t.Fatalf("wrong number of resource instances for dst, got %d expected 2", len(s.Resource(dst).Instances))
		}

		got := s.Resource(dst)
		if got.Addr.Resource != dst.Resource {
			t.Fatalf("dst resource not in state")
		}
	})

	t.Run("module to new module", func(t *testing.T) {
		s := NewState()
		srcModule := addrs.RootModuleInstance.Child("kinder", addrs.StringKey("exists"))
		dstModule := addrs.RootModuleInstance.Child("kinder", addrs.StringKey("new"))
		cm := s.EnsureModule(srcModule)
		cm.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "child",
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

		src := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "child"}.Absolute(srcModule)
		dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "child"}.Absolute(dstModule)
		s.MoveAbsResource(src, dst)

		if s.Empty() {
			t.Fatal("unexpected empty state")
		}

		// The child module should have been removed after removing its only resource
		if s.Module(srcModule) != nil {
			t.Fatalf("child module %s was not removed from state after mv", srcModule.String())
		}

		gotMod := s.Module(dstModule)
		if len(gotMod.Resources) != 1 {
			t.Fatalf("wrong number of resources in state; expected 1, found %d", len(gotMod.Resources))
		}

		got := s.Resource(dst)
		if got.Addr.Resource != dst.Resource {
			t.Fatalf("dst resource not in state")
		}
	})

	t.Run("module to new module", func(t *testing.T) {
		s := NewState()
		srcModule := addrs.RootModuleInstance.Child("kinder", addrs.StringKey("exists"))
		dstModule := addrs.RootModuleInstance.Child("kinder", addrs.StringKey("new"))
		cm := s.EnsureModule(srcModule)
		cm.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_thing",
				Name: "child",
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

		src := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "child"}.Absolute(srcModule)
		dst := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_thing", Name: "child"}.Absolute(dstModule)
		s.MoveAbsResource(src, dst)

		if s.Empty() {
			t.Fatal("unexpected empty state")
		}

		// The child module should have been removed after removing its only resource
		if s.Module(srcModule) != nil {
			t.Fatalf("child module %s was not removed from state after mv", srcModule.String())
		}

		gotMod := s.Module(dstModule)
		if len(gotMod.Resources) != 1 {
			t.Fatalf("wrong number of resources in state; expected 1, found %d", len(gotMod.Resources))
		}

		got := s.Resource(dst)
		if got.Addr.Resource != dst.Resource {
			t.Fatalf("dst resource not in state")
		}
	})
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

	// For a little extra fun, let's go from a resource to a resource instance: test_thing.foo to test_thing.bar[1]
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

func TestState_MoveModuleInstance(t *testing.T) {
	state := NewState()
	srcModule := addrs.RootModuleInstance.Child("kinder", addrs.NoKey)
	m := state.EnsureModule(srcModule)
	m.SetResourceInstanceCurrent(
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

	dstModule := addrs.RootModuleInstance.Child("child", addrs.IntKey(3))
	state.MoveModuleInstance(srcModule, dstModule)

	// srcModule should have been removed, dstModule should exist and have one resource
	if len(state.Modules) != 2 { // kinder[3] and root
		t.Fatalf("wrong number of modules in state. Expected 2, got %d", len(state.Modules))
	}

	got := state.Module(dstModule)
	if got == nil {
		t.Fatal("dstModule not found")
	}

	gone := state.Module(srcModule)
	if gone != nil {
		t.Fatal("srcModule not removed from state")
	}

	r := got.Resource(mustAbsResourceAddr("test_thing.foo").Resource)
	if r.Addr.Module.String() != dstModule.String() {
		fmt.Println(r.Addr.Module.String())
		t.Fatal("resource address was not updated")
	}

}

func TestState_MaybeMoveModuleInstance(t *testing.T) {
	state := NewState()
	src := addrs.RootModuleInstance.Child("child", addrs.StringKey("a"))
	cm := state.EnsureModule(src)
	cm.SetResourceInstanceCurrent(
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

	dst := addrs.RootModuleInstance.Child("kinder", addrs.StringKey("b"))

	// First move, success
	t.Run("first move", func(t *testing.T) {
		moved := state.MaybeMoveModuleInstance(src, dst)
		if !moved {
			t.Fatal("wrong result")
		}
	})

	// Second move, should be a noop
	t.Run("noop", func(t *testing.T) {
		moved := state.MaybeMoveModuleInstance(src, dst)
		if moved {
			t.Fatal("wrong result")
		}
	})
}

func TestState_MoveModule(t *testing.T) {
	// For this test, add two module instances (kinder and kinder["a"]).
	// MoveModule(kinder) should move both instances.
	state := NewState() // starter state, should be copied by the subtests.
	srcModule := addrs.RootModule.Child("kinder")
	m := state.EnsureModule(srcModule.UnkeyedInstanceShim())
	m.SetResourceInstanceCurrent(
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

	moduleInstance := addrs.RootModuleInstance.Child("kinder", addrs.StringKey("a"))
	mi := state.EnsureModule(moduleInstance)
	mi.SetResourceInstanceCurrent(
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

	_, mc := srcModule.Call()
	src := mc.Absolute(addrs.RootModuleInstance.Child("kinder", addrs.NoKey))

	t.Run("basic", func(t *testing.T) {
		s := state.DeepCopy()
		_, dstMC := addrs.RootModule.Child("child").Call()
		dst := dstMC.Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		s.MoveModule(src, dst)

		// srcModule should have been removed, dstModule should exist and have one resource
		if len(s.Modules) != 3 { // child, child["a"] and root
			t.Fatalf("wrong number of modules in state. Expected 3, got %d", len(s.Modules))
		}

		got := s.Module(dst.Module)
		if got == nil {
			t.Fatal("dstModule not found")
		}

		got = s.Module(addrs.RootModuleInstance.Child("child", addrs.StringKey("a")))
		if got == nil {
			t.Fatal("dstModule instance \"a\" not found")
		}

		gone := s.Module(srcModule.UnkeyedInstanceShim())
		if gone != nil {
			t.Fatal("srcModule not removed from state")
		}
	})

	t.Run("nested modules", func(t *testing.T) {
		s := state.DeepCopy()

		// add a child module to module.kinder
		mi := mustParseModuleInstanceStr(`module.kinder.module.grand[1]`)
		m := s.EnsureModule(mi)
		m.SetResourceInstanceCurrent(
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

		_, dstMC := addrs.RootModule.Child("child").Call()
		dst := dstMC.Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey))
		s.MoveModule(src, dst)

		moved := s.Module(addrs.RootModuleInstance.Child("child", addrs.StringKey("a")))
		if moved == nil {
			t.Fatal("dstModule not found")
		}

		// The nested module's relative address should also have been updated
		nested := s.Module(mustParseModuleInstanceStr(`module.child.module.grand[1]`))
		if nested == nil {
			t.Fatal("nested child module of src wasn't moved")
		}
	})
}

func mustParseModuleInstanceStr(str string) addrs.ModuleInstance {
	addr, diags := addrs.ParseModuleInstanceStr(str)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}

func mustAbsResourceAddr(s string) addrs.AbsResource {
	addr, diags := addrs.ParseAbsResourceStr(s)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return addr
}
