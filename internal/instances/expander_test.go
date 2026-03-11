// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package instances

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
)

func TestExpanderWithOverrides(t *testing.T) {

	mustModuleInstance := func(t *testing.T, s string) addrs.ModuleInstance {
		if len(s) == 0 {
			return addrs.RootModuleInstance
		}

		addr, diags := addrs.ParseModuleInstanceStr(s)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}
		return addr
	}

	tcs := map[string]struct {
		// Hook to install chosen overrides.
		overrides mocking.InitLocalOverrides

		// Hook to initialise the expander with the desired state.
		expander func(*Expander)

		// The target module instance to inspect.
		target string

		// Set to true to include overrides in the result.
		includeOverrides bool

		// The expected result.
		wantModules []addrs.ModuleInstance

		// The expected result for partial modules.
		wantPartials map[string]bool
	}{
		"root module": {
			wantModules:  singletonRootModule,
			wantPartials: make(map[string]bool),
		},
		"instanced child module not overridden": {
			expander: func(expander *Expander) {
				expander.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "double"}, 2)
			},
			target: "module.double",
			wantModules: []addrs.ModuleInstance{
				mustModuleInstance(t, "module.double[0]"),
				mustModuleInstance(t, "module.double[1]"),
			},
			wantPartials: make(map[string]bool),
		},
		"instanced child module single instance overridden": {
			overrides: func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance(t, "module.double[0]"), &configs.Override{})
			},
			expander: func(expander *Expander) {
				expander.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "double"}, 2)
			},
			target: "module.double",
			wantModules: []addrs.ModuleInstance{
				mustModuleInstance(t, "module.double[1]"),
			},
			wantPartials: make(map[string]bool),
		},
		"instanced child module single instance overridden includes overrides": {
			overrides: func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance(t, "module.double[0]"), &configs.Override{})
			},
			expander: func(expander *Expander) {
				expander.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "double"}, 2)
			},
			target:           "module.double",
			includeOverrides: true,
			wantModules: []addrs.ModuleInstance{
				mustModuleInstance(t, "module.double[0]"),
				mustModuleInstance(t, "module.double[1]"),
			},
			wantPartials: make(map[string]bool),
		},
		"deeply nested child module with parent overridden": {
			overrides: func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance(t, "module.double[0]"), &configs.Override{})
			},
			expander: func(expander *Expander) {
				expander.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "double"}, 2)
				expander.SetModuleSingle(mustModuleInstance(t, "module.double[1]"), addrs.ModuleCall{Name: "single"})
			},
			target:       "module.double.module.single",
			wantModules:  []addrs.ModuleInstance{mustModuleInstance(t, "module.double[1].module.single")},
			wantPartials: make(map[string]bool),
		},
		"unknown child module overridden by instanced module": {
			overrides: func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance(t, "module.unknown[0]"), &configs.Override{})
			},
			expander: func(expander *Expander) {
				expander.SetModuleCountUnknown(addrs.RootModuleInstance, addrs.ModuleCall{Name: "unknown"})
			},
			target: "module.unknown",
			wantPartials: map[string]bool{
				"module.unknown[*]": true,
			},
		},
		"unknown child module overridden by instanced module includes overrides": {
			overrides: func(overrides addrs.Map[addrs.Targetable, *configs.Override]) {
				overrides.Put(mustModuleInstance(t, "module.unknown"), &configs.Override{})
			},
			expander: func(expander *Expander) {
				expander.SetModuleCountUnknown(addrs.RootModuleInstance, addrs.ModuleCall{Name: "unknown"})
			},
			target:       "module.unknown",
			wantPartials: make(map[string]bool), // This time it's empty, as we overrode all instances.
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			overrides := mocking.OverridesForTesting(nil, tc.overrides)

			expander := NewExpander(overrides)
			if tc.expander != nil {
				tc.expander(expander)
			}

			target := mustModuleInstance(t, tc.target).Module()

			gotModules := expander.ExpandModule(target, tc.includeOverrides)
			gotPartials := expander.UnknownModuleInstances(target, tc.includeOverrides)

			if diff := cmp.Diff(tc.wantModules, gotModules); len(diff) > 0 {
				t.Errorf("wrong result\n%s", diff)
			}

			// Convert the gotPartials into strings to make cmp.Diff work.
			gotPartialsStr := make(map[string]bool, len(gotPartials))
			for _, partial := range gotPartials {
				gotPartialsStr[partial.String()] = true
			}

			if diff := cmp.Diff(tc.wantPartials, gotPartialsStr, ctydebug.CmpOptions); len(diff) > 0 {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}

}

func TestExpander(t *testing.T) {
	// Some module and resource addresses and values we'll use repeatedly below.
	singleModuleAddr := addrs.ModuleCall{Name: "single"}
	count2ModuleAddr := addrs.ModuleCall{Name: "count2"}
	count0ModuleAddr := addrs.ModuleCall{Name: "count0"}
	forEachModuleAddr := addrs.ModuleCall{Name: "for_each"}
	singleResourceAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "single",
	}
	count2ResourceAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "count2",
	}
	count0ResourceAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "count0",
	}
	forEachResourceAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "for_each",
	}
	eachMap := map[string]cty.Value{
		"a": cty.NumberIntVal(1),
		"b": cty.NumberIntVal(2),
	}

	// In normal use, Expander would be called in the context of a graph
	// traversal to ensure that information is registered/requested in the
	// correct sequence, but to keep this test self-contained we'll just
	// manually write out the steps here.
	//
	// The steps below are assuming a configuration tree like the following:
	// - root module
	//   - resource test.single with no count or for_each
	//   - resource test.count2 with count = 2
	//   - resource test.count0 with count = 0
	//   - resource test.for_each with for_each = { a = 1, b = 2 }
	//   - child module "single" with no count or for_each
	//     - resource test.single with no count or for_each
	//     - resource test.count2 with count = 2
	//   - child module "count2" with count = 2
	//     - resource test.single with no count or for_each
	//     - resource test.count2 with count = 2
	//     - child module "count2" with count = 2
	//       - resource test.count2 with count = 2
	//   - child module "count0" with count = 0
	//     - resource test.single with no count or for_each
	//   - child module for_each with for_each = { a = 1, b = 2 }
	//     - resource test.single with no count or for_each
	//     - resource test.count2 with count = 2

	ex := NewExpander(nil)

	// We don't register the root module, because it's always implied to exist.
	//
	// Below we're going to use braces and indentation just to help visually
	// reflect the tree structure from the tree in the above comment, in the
	// hope that the following is easier to follow.
	//
	// The Expander API requires that we register containing modules before
	// registering anything inside them, so we'll work through the above
	// in a depth-first order in the registration steps that follow.
	{
		ex.SetResourceSingle(addrs.RootModuleInstance, singleResourceAddr)
		ex.SetResourceCount(addrs.RootModuleInstance, count2ResourceAddr, 2)
		ex.SetResourceCount(addrs.RootModuleInstance, count0ResourceAddr, 0)
		ex.SetResourceForEach(addrs.RootModuleInstance, forEachResourceAddr, eachMap)

		ex.SetModuleSingle(addrs.RootModuleInstance, singleModuleAddr)
		{
			// The single instance of the module
			moduleInstanceAddr := addrs.RootModuleInstance.Child("single", addrs.NoKey)
			ex.SetResourceSingle(moduleInstanceAddr, singleResourceAddr)
			ex.SetResourceCount(moduleInstanceAddr, count2ResourceAddr, 2)
		}

		ex.SetModuleCount(addrs.RootModuleInstance, count2ModuleAddr, 2)
		for i1 := 0; i1 < 2; i1++ {
			moduleInstanceAddr := addrs.RootModuleInstance.Child("count2", addrs.IntKey(i1))
			ex.SetResourceSingle(moduleInstanceAddr, singleResourceAddr)
			ex.SetResourceCount(moduleInstanceAddr, count2ResourceAddr, 2)
			ex.SetModuleCount(moduleInstanceAddr, count2ModuleAddr, 2)
			for i2 := 0; i2 < 2; i2++ {
				moduleInstanceAddr := moduleInstanceAddr.Child("count2", addrs.IntKey(i2))
				ex.SetResourceCount(moduleInstanceAddr, count2ResourceAddr, 2)
			}
		}

		ex.SetModuleCount(addrs.RootModuleInstance, count0ModuleAddr, 0)
		{
			// There are no instances of module "count0", so our nested module
			// would never actually get registered here: the expansion node
			// for the resource would see that its containing module has no
			// instances and so do nothing.
		}

		ex.SetModuleForEach(addrs.RootModuleInstance, forEachModuleAddr, eachMap)
		for k := range eachMap {
			moduleInstanceAddr := addrs.RootModuleInstance.Child("for_each", addrs.StringKey(k))
			ex.SetResourceSingle(moduleInstanceAddr, singleResourceAddr)
			ex.SetResourceCount(moduleInstanceAddr, count2ResourceAddr, 2)
		}
	}

	t.Run("root module", func(t *testing.T) {
		// Requesting expansion of the root module doesn't really mean anything
		// since it's always a singleton, but for consistency it should work.
		got := ex.ExpandModule(addrs.RootModule, false)
		want := []addrs.ModuleInstance{addrs.RootModuleInstance}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("resource single", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			addrs.RootModule,
			singleResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`test.single`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("resource count2", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			addrs.RootModule,
			count2ResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`test.count2[0]`),
			mustAbsResourceInstanceAddr(`test.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("resource count0", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			addrs.RootModule,
			count0ResourceAddr,
		)
		want := []addrs.AbsResourceInstance(nil)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("resource for_each", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			addrs.RootModule,
			forEachResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`test.for_each["a"]`),
			mustAbsResourceInstanceAddr(`test.for_each["b"]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module single", func(t *testing.T) {
		got := ex.ExpandModule(addrs.RootModule.Child("single"), false)
		want := []addrs.ModuleInstance{
			mustModuleInstanceAddr(`module.single`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module single resource single", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			mustModuleAddr("single"),
			singleResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr("module.single.test.single"),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module single resource count2", func(t *testing.T) {
		// Two different ways of asking the same question, which should
		// both produce the same result.
		// First: nested expansion of all instances of the resource across
		// all instances of the module, but it's a single-instance module
		// so the first level is a singleton.
		got1 := ex.ExpandModuleResource(
			mustModuleAddr(`single`),
			count2ResourceAddr,
		)
		// Second: expansion of only instances belonging to a specific
		// instance of the module, but again it's a single-instance module
		// so there's only one to ask about.
		got2 := ex.ExpandResource(
			count2ResourceAddr.Absolute(
				addrs.RootModuleInstance.Child("single", addrs.NoKey),
			),
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.single.test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.single.test.count2[1]`),
		}
		if diff := cmp.Diff(want, got1); diff != "" {
			t.Errorf("wrong ExpandModuleResource result\n%s", diff)
		}
		if diff := cmp.Diff(want, got2); diff != "" {
			t.Errorf("wrong ExpandResource result\n%s", diff)
		}
	})
	t.Run("module single resource count2 with non-existing module instance", func(t *testing.T) {
		got := ex.ExpandResource(
			count2ResourceAddr.Absolute(
				// Note: This is intentionally an invalid instance key,
				// so we're asking about module.single[1].test.count2
				// even though module.single doesn't have count set and
				// therefore there is no module.single[1].
				addrs.RootModuleInstance.Child("single", addrs.IntKey(1)),
			),
		)
		// If the containing module instance doesn't exist then it can't
		// possibly have any resource instances inside it.
		want := ([]addrs.AbsResourceInstance)(nil)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2", func(t *testing.T) {
		got := ex.ExpandModule(mustModuleAddr(`count2`), false)
		want := []addrs.ModuleInstance{
			mustModuleInstanceAddr(`module.count2[0]`),
			mustModuleInstanceAddr(`module.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2 resource single", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			mustModuleAddr(`count2`),
			singleResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.count2[0].test.single`),
			mustAbsResourceInstanceAddr(`module.count2[1].test.single`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2 resource count2", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			mustModuleAddr(`count2`),
			count2ResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.count2[0].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.count2[0].test.count2[1]`),
			mustAbsResourceInstanceAddr(`module.count2[1].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.count2[1].test.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2 module count2", func(t *testing.T) {
		got := ex.ExpandModule(mustModuleAddr(`count2.count2`), false)
		want := []addrs.ModuleInstance{
			mustModuleInstanceAddr(`module.count2[0].module.count2[0]`),
			mustModuleInstanceAddr(`module.count2[0].module.count2[1]`),
			mustModuleInstanceAddr(`module.count2[1].module.count2[0]`),
			mustModuleInstanceAddr(`module.count2[1].module.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2[0] module count2 instances", func(t *testing.T) {
		instAddr := mustModuleInstanceAddr(`module.count2[0].module.count2[0]`)
		callAddr := instAddr.AbsCall() // discards the final [0] instance key from the above
		keyType, got, known := ex.ExpandAbsModuleCall(callAddr)
		if !known {
			t.Fatal("expansion unknown; want known")
		}
		if keyType != addrs.IntKeyType {
			t.Fatalf("wrong key type %#v; want %#v", keyType, addrs.IntKeyType)
		}
		want := []addrs.InstanceKey{
			addrs.IntKey(0),
			addrs.IntKey(1),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2 module count2 GetDeepestExistingModuleInstance", func(t *testing.T) {
		t.Run("first step invalid", func(t *testing.T) {
			got := ex.GetDeepestExistingModuleInstance(mustModuleInstanceAddr(`module.count2["nope"].module.count2[0]`))
			want := addrs.RootModuleInstance
			if !want.Equal(got) {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
			}
		})
		t.Run("second step invalid", func(t *testing.T) {
			got := ex.GetDeepestExistingModuleInstance(mustModuleInstanceAddr(`module.count2[1].module.count2`))
			want := mustModuleInstanceAddr(`module.count2[1]`)
			if !want.Equal(got) {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
			}
		})
		t.Run("neither step valid", func(t *testing.T) {
			got := ex.GetDeepestExistingModuleInstance(mustModuleInstanceAddr(`module.count2.module.count2["nope"]`))
			want := addrs.RootModuleInstance
			if !want.Equal(got) {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
			}
		})
		t.Run("both steps valid", func(t *testing.T) {
			got := ex.GetDeepestExistingModuleInstance(mustModuleInstanceAddr(`module.count2[1].module.count2[0]`))
			want := mustModuleInstanceAddr(`module.count2[1].module.count2[0]`)
			if !want.Equal(got) {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
			}
		})
	})
	t.Run("module count2 resource count2 resource count2", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			mustModuleAddr(`count2.count2`),
			count2ResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.count2[0].module.count2[0].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.count2[0].module.count2[0].test.count2[1]`),
			mustAbsResourceInstanceAddr(`module.count2[0].module.count2[1].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.count2[0].module.count2[1].test.count2[1]`),
			mustAbsResourceInstanceAddr(`module.count2[1].module.count2[0].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.count2[1].module.count2[0].test.count2[1]`),
			mustAbsResourceInstanceAddr(`module.count2[1].module.count2[1].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.count2[1].module.count2[1].test.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2 resource count2 resource count2", func(t *testing.T) {
		got := ex.ExpandResource(
			count2ResourceAddr.Absolute(mustModuleInstanceAddr(`module.count2[0].module.count2[1]`)),
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.count2[0].module.count2[1].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.count2[0].module.count2[1].test.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count0", func(t *testing.T) {
		got := ex.ExpandModule(mustModuleAddr(`count0`), false)
		want := []addrs.ModuleInstance(nil)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count0 resource single", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			mustModuleAddr(`count0`),
			singleResourceAddr,
		)
		// The containing module has zero instances, so therefore there
		// are zero instances of this resource even though it doesn't have
		// count = 0 set itself.
		want := []addrs.AbsResourceInstance(nil)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module for_each", func(t *testing.T) {
		got := ex.ExpandModule(mustModuleAddr(`for_each`), false)
		want := []addrs.ModuleInstance{
			mustModuleInstanceAddr(`module.for_each["a"]`),
			mustModuleInstanceAddr(`module.for_each["b"]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module for_each resource single", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			mustModuleAddr(`for_each`),
			singleResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.for_each["a"].test.single`),
			mustAbsResourceInstanceAddr(`module.for_each["b"].test.single`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module for_each resource count2", func(t *testing.T) {
		got := ex.ExpandModuleResource(
			mustModuleAddr(`for_each`),
			count2ResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.for_each["a"].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.for_each["a"].test.count2[1]`),
			mustAbsResourceInstanceAddr(`module.for_each["b"].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.for_each["b"].test.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module for_each resource count2", func(t *testing.T) {
		got := ex.ExpandResource(
			count2ResourceAddr.Absolute(mustModuleInstanceAddr(`module.for_each["a"]`)),
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.for_each["a"].test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.for_each["a"].test.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})

	t.Run(`module.for_each["b"] repetitiondata`, func(t *testing.T) {
		got := ex.GetModuleInstanceRepetitionData(
			mustModuleInstanceAddr(`module.for_each["b"]`),
		)
		want := RepetitionData{
			EachKey:   cty.StringVal("b"),
			EachValue: cty.NumberIntVal(2),
		}
		if diff := cmp.Diff(want, got, cmp.Comparer(valueEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run(`module.count2[0].module.count2[1] repetitiondata`, func(t *testing.T) {
		got := ex.GetModuleInstanceRepetitionData(
			mustModuleInstanceAddr(`module.count2[0].module.count2[1]`),
		)
		want := RepetitionData{
			CountIndex: cty.NumberIntVal(1),
		}
		if diff := cmp.Diff(want, got, cmp.Comparer(valueEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run(`module.for_each["a"] repetitiondata`, func(t *testing.T) {
		got := ex.GetModuleInstanceRepetitionData(
			mustModuleInstanceAddr(`module.for_each["a"]`),
		)
		want := RepetitionData{
			EachKey:   cty.StringVal("a"),
			EachValue: cty.NumberIntVal(1),
		}
		if diff := cmp.Diff(want, got, cmp.Comparer(valueEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})

	t.Run(`test.for_each["a"] repetitiondata`, func(t *testing.T) {
		got := ex.GetResourceInstanceRepetitionData(
			mustAbsResourceInstanceAddr(`test.for_each["a"]`),
		)
		want := RepetitionData{
			EachKey:   cty.StringVal("a"),
			EachValue: cty.NumberIntVal(1),
		}
		if diff := cmp.Diff(want, got, cmp.Comparer(valueEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run(`module.for_each["a"].test.single repetitiondata`, func(t *testing.T) {
		got := ex.GetResourceInstanceRepetitionData(
			mustAbsResourceInstanceAddr(`module.for_each["a"].test.single`),
		)
		want := RepetitionData{}
		if diff := cmp.Diff(want, got, cmp.Comparer(valueEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run(`module.for_each["a"].test.count2[1] repetitiondata`, func(t *testing.T) {
		got := ex.GetResourceInstanceRepetitionData(
			mustAbsResourceInstanceAddr(`module.for_each["a"].test.count2[1]`),
		)
		want := RepetitionData{
			CountIndex: cty.NumberIntVal(1),
		}
		if diff := cmp.Diff(want, got, cmp.Comparer(valueEquals)); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}

func TestExpanderWithUnknowns(t *testing.T) {
	t.Run("resource in root module with unknown for_each", func(t *testing.T) {
		resourceAddr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test",
			Name: "foo",
		}
		ex := NewExpander(nil)
		ex.SetResourceForEachUnknown(addrs.RootModuleInstance, resourceAddr)

		got := ex.ExpandModuleResource(addrs.RootModule, resourceAddr)
		if len(got) != 0 {
			t.Errorf("unexpected known addresses: %#v", got)
		}
	})
	t.Run("resource in root module with unknown count", func(t *testing.T) {
		resourceAddr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test",
			Name: "foo",
		}
		ex := NewExpander(nil)
		ex.SetResourceCountUnknown(addrs.RootModuleInstance, resourceAddr)

		got := ex.ExpandModuleResource(addrs.RootModule, resourceAddr)
		if len(got) != 0 {
			t.Errorf("unexpected known addresses: %#v", got)
		}
	})
	t.Run("module with unknown for_each", func(t *testing.T) {
		moduleCallAddr := addrs.ModuleCall{Name: "foo"}
		ex := NewExpander(nil)
		ex.SetModuleForEachUnknown(addrs.RootModuleInstance, moduleCallAddr)

		got := ex.ExpandModule(addrs.Module{moduleCallAddr.Name}, false)
		if len(got) != 0 {
			t.Errorf("unexpected known addresses: %#v", got)
		}

		gotUnknown := ex.UnknownModuleInstances(addrs.Module{moduleCallAddr.Name}, false)
		if len(gotUnknown) != 1 {
			t.Errorf("unexpected unknown addresses: %#v", gotUnknown)
		}
		wantUnknownCall := addrs.RootModuleInstance.UnexpandedChild(moduleCallAddr)
		if !gotUnknown.Has(wantUnknownCall) {
			t.Errorf("unknown should have %s, but it doesn't", wantUnknownCall)
		}
	})
	t.Run("module with unknown count", func(t *testing.T) {
		moduleCallAddr := addrs.ModuleCall{Name: "foo"}
		ex := NewExpander(nil)
		ex.SetModuleCountUnknown(addrs.RootModuleInstance, moduleCallAddr)

		gotKnown := ex.ExpandModule(addrs.Module{moduleCallAddr.Name}, false)
		if len(gotKnown) != 0 {
			t.Errorf("unexpected known addresses: %#v", gotKnown)
		}

		gotUnknown := ex.UnknownModuleInstances(addrs.Module{moduleCallAddr.Name}, false)
		if len(gotUnknown) != 1 {
			t.Errorf("unexpected unknown addresses: %#v", gotUnknown)
		}
		wantUnknownCall := addrs.RootModuleInstance.UnexpandedChild(moduleCallAddr)
		if !gotUnknown.Has(wantUnknownCall) {
			t.Errorf("unknown should have %s, but it doesn't", wantUnknownCall)
		}
	})
	t.Run("nested module with unknown count", func(t *testing.T) {
		moduleCallAddr1 := addrs.ModuleCall{Name: "foo"}
		moduleCallAddr2 := addrs.ModuleCall{Name: "bar"}
		module1 := addrs.RootModule.Child(moduleCallAddr1.Name)
		module2 := module1.Child(moduleCallAddr2.Name)
		module1Inst0 := addrs.RootModuleInstance.Child("foo", addrs.IntKey(0))
		module1Inst1 := addrs.RootModuleInstance.Child("foo", addrs.IntKey(1))
		module1Inst2 := addrs.RootModuleInstance.Child("foo", addrs.IntKey(2))
		ex := NewExpander(nil)
		ex.SetModuleCount(addrs.RootModuleInstance, moduleCallAddr1, 3)
		ex.SetModuleCountUnknown(module1Inst0, moduleCallAddr2)
		ex.SetModuleCount(module1Inst1, moduleCallAddr2, 1)
		ex.SetModuleCountUnknown(module1Inst2, moduleCallAddr2)

		// We'll also put some resources inside module.foo[1].module.bar[0]
		// so that we can test requesting unknown resource instance sets.
		resourceAddrKnownExp := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test",
			Name: "known_expansion",
		}
		resourceAddrUnknownExp := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test",
			Name: "unknown_expansion",
		}
		module1Inst1Module2Inst0 := module1Inst1.Child("bar", addrs.IntKey(0))
		ex.SetResourceCount(module1Inst1Module2Inst0, resourceAddrKnownExp, 2)
		ex.SetResourceCountUnknown(module1Inst1Module2Inst0, resourceAddrUnknownExp)

		module2Call := addrs.AbsModuleCall{
			Module: module1Inst0,
			Call:   moduleCallAddr2,
		}
		_, _, instsKnown := ex.ExpandAbsModuleCall(module2Call)
		if instsKnown {
			t.Fatalf("instances of %s are known; should be unknown", module2Call.String())
		}

		gotKnown := ex.ExpandModule(module2, false)
		wantKnown := []addrs.ModuleInstance{
			module1Inst1.Child("bar", addrs.IntKey(0)),
		}
		if diff := cmp.Diff(wantKnown, gotKnown); diff != "" {
			t.Errorf("unexpected known addresses\n%s", diff)
		}

		gotUnknown := ex.UnknownModuleInstances(module2, false)
		if len(gotUnknown) != 2 {
			t.Errorf("unexpected unknown addresses: %#v", gotUnknown)
		}
		if wantUnknownCall := module1Inst0.UnexpandedChild(moduleCallAddr2); !gotUnknown.Has(wantUnknownCall) {
			t.Errorf("unknown should have %s, but it doesn't", wantUnknownCall)
		}
		if unwantUnknownCall := module1Inst1.UnexpandedChild(moduleCallAddr2); gotUnknown.Has(unwantUnknownCall) {
			t.Errorf("unknown should not have %s, but does", unwantUnknownCall)
		}
		if wantUnknownCall := module1Inst2.UnexpandedChild(moduleCallAddr2); !gotUnknown.Has(wantUnknownCall) {
			t.Errorf("unknown should have %s, but it doesn't", wantUnknownCall)
		}

		gotKnownResource := ex.ExpandResource(module1Inst1Module2Inst0.Resource(
			resourceAddrKnownExp.Mode, resourceAddrKnownExp.Type, resourceAddrKnownExp.Name,
		))
		wantKnownResource := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr("module.foo[1].module.bar[0].test.known_expansion[0]"),
			mustAbsResourceInstanceAddr("module.foo[1].module.bar[0].test.known_expansion[1]"),
		}
		if diff := cmp.Diff(wantKnownResource, gotKnownResource); diff != "" {
			t.Errorf("unexpected known addresses\n%s", diff)
		}

		gotUnknownResource := ex.UnknownResourceInstances(module2.Resource(
			resourceAddrUnknownExp.Mode, resourceAddrUnknownExp.Type, resourceAddrUnknownExp.Name,
		))
		if len(gotUnknownResource) != 3 {
			t.Errorf("unexpected unknown addresses: %#v", gotUnknownResource)
		}
		if wantResInst := module1Inst0.UnexpandedChild(moduleCallAddr2).Resource(resourceAddrUnknownExp); !gotUnknownResource.Has(wantResInst) {
			t.Errorf("unknown should have %s, but it doesn't", wantResInst)
		}
		if wantResInst := module1Inst1Module2Inst0.UnexpandedResource(resourceAddrUnknownExp); !gotUnknownResource.Has(wantResInst) {
			t.Errorf("unknown should have %s, but it doesn't", wantResInst)
		}
		if wantResInst := module1Inst2.UnexpandedChild(moduleCallAddr2).Resource(resourceAddrUnknownExp); !gotUnknownResource.Has(wantResInst) {
			t.Errorf("unknown should have %s, but it doesn't", wantResInst)
		}
	})
}

func mustAbsResourceInstanceAddr(str string) addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(str)
	if diags.HasErrors() {
		panic(fmt.Sprintf("invalid absolute resource instance address: %s", diags.Err()))
	}
	return addr
}

func mustModuleAddr(str string) addrs.Module {
	if len(str) == 0 {
		return addrs.RootModule
	}
	// We don't have a real parser for these because they don't appear in the
	// language anywhere, but this interpretation mimics the format we
	// produce from the String method on addrs.Module.
	parts := strings.Split(str, ".")
	return addrs.Module(parts)
}

func mustModuleInstanceAddr(str string) addrs.ModuleInstance {
	if len(str) == 0 {
		return addrs.RootModuleInstance
	}
	addr, diags := addrs.ParseModuleInstanceStr(str)
	if diags.HasErrors() {
		panic(fmt.Sprintf("invalid module instance address: %s", diags.Err()))
	}
	return addr
}

func valueEquals(a, b cty.Value) bool {
	if a == cty.NilVal || b == cty.NilVal {
		return a == b
	}
	return a.RawEquals(b)
}
