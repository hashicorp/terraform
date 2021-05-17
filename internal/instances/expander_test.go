package instances

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

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

	ex := NewExpander()

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
		got := ex.ExpandModule(addrs.RootModule)
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
		got := ex.ExpandModule(addrs.RootModule.Child("single"))
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
		got := ex.ExpandModuleResource(
			mustModuleAddr(`single`),
			count2ResourceAddr,
		)
		want := []addrs.AbsResourceInstance{
			mustAbsResourceInstanceAddr(`module.single.test.count2[0]`),
			mustAbsResourceInstanceAddr(`module.single.test.count2[1]`),
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("module count2", func(t *testing.T) {
		got := ex.ExpandModule(mustModuleAddr(`count2`))
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
		got := ex.ExpandModule(mustModuleAddr(`count2.count2`))
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
		got := ex.ExpandModule(mustModuleAddr(`count0`))
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
		got := ex.ExpandModule(mustModuleAddr(`for_each`))
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
