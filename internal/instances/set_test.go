package instances

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestSet(t *testing.T) {
	exp := NewExpander()

	// The following constructs the following imaginary module/resource tree:
	// - root module
	//   - test_thing.single: no repetition
	//   - test_thing.count: count = 1
	//   - test_thing.for_each: for_each = { c = "C" }
	//   - module.single: no repetition
	//     - test_thing.single: no repetition
	//     - module.nested_single: no repetition
	//       - module.zero_count: count = 0
	//   - module.count: count = 2
	//     - module.nested_for_each: [0] for_each = {}, [1] for_each = { e = "E" }
	//   - module.for_each: for_each = { a = "A", b = "B" }
	//     - test_thing.count: ["a"] count = 0, ["b"] count = 1
	exp.SetModuleSingle(addrs.RootModuleInstance, addrs.ModuleCall{Name: "single"})
	exp.SetModuleCount(addrs.RootModuleInstance, addrs.ModuleCall{Name: "count"}, 2)
	exp.SetModuleForEach(addrs.RootModuleInstance, addrs.ModuleCall{Name: "for_each"}, map[string]cty.Value{
		"a": cty.StringVal("A"),
		"b": cty.StringVal("B"),
	})
	exp.SetModuleSingle(addrs.RootModuleInstance.Child("single", addrs.NoKey), addrs.ModuleCall{Name: "nested_single"})
	exp.SetModuleForEach(addrs.RootModuleInstance.Child("count", addrs.IntKey(0)), addrs.ModuleCall{Name: "nested_for_each"}, nil)
	exp.SetModuleForEach(addrs.RootModuleInstance.Child("count", addrs.IntKey(1)), addrs.ModuleCall{Name: "nested_for_each"}, map[string]cty.Value{
		"e": cty.StringVal("E"),
	})
	exp.SetModuleCount(
		addrs.RootModuleInstance.Child("single", addrs.NoKey).Child("nested_single", addrs.NoKey),
		addrs.ModuleCall{Name: "zero_count"},
		0,
	)

	rAddr := func(name string) addrs.Resource {
		return addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_thing",
			Name: name,
		}
	}
	exp.SetResourceSingle(addrs.RootModuleInstance, rAddr("single"))
	exp.SetResourceCount(addrs.RootModuleInstance, rAddr("count"), 1)
	exp.SetResourceForEach(addrs.RootModuleInstance, rAddr("for_each"), map[string]cty.Value{
		"c": cty.StringVal("C"),
	})
	exp.SetResourceSingle(addrs.RootModuleInstance.Child("single", addrs.NoKey), rAddr("single"))
	exp.SetResourceCount(addrs.RootModuleInstance.Child("for_each", addrs.StringKey("a")), rAddr("count"), 0)
	exp.SetResourceCount(addrs.RootModuleInstance.Child("for_each", addrs.StringKey("b")), rAddr("count"), 1)

	set := exp.AllInstances()

	// HasModuleInstance tests
	if input := addrs.RootModuleInstance; !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey); !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey).Child("nested_single", addrs.NoKey); !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.IntKey(0)); !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.IntKey(1)); !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.IntKey(1)).Child("nested_for_each", addrs.StringKey("e")); !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("for_each", addrs.StringKey("a")); !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("for_each", addrs.StringKey("b")); !set.HasModuleInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.IntKey(0)); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.StringKey("a")); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey).Child("nonexist", addrs.NoKey); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.NoKey); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.IntKey(2)); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.StringKey("a")); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.IntKey(0)).Child("nested_for_each", addrs.StringKey("e")); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey).Child("nested_single", addrs.NoKey).Child("zero_count", addrs.NoKey); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey).Child("nested_single", addrs.NoKey).Child("zero_count", addrs.IntKey(0)); set.HasModuleInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}

	// HasModuleCall tests
	if input := addrs.RootModuleInstance.ChildCall("single"); !set.HasModuleCall(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey).ChildCall("nested_single"); !set.HasModuleCall(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.ChildCall("count"); !set.HasModuleCall(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.IntKey(0)).ChildCall("nested_for_each"); !set.HasModuleCall(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("count", addrs.IntKey(1)).ChildCall("nested_for_each"); !set.HasModuleCall(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.ChildCall("for_each"); !set.HasModuleCall(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey).Child("nested_single", addrs.NoKey).ChildCall("zero_count"); !set.HasModuleCall(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.ChildCall("nonexist"); set.HasModuleCall(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := addrs.RootModuleInstance.Child("single", addrs.NoKey).ChildCall("nonexist"); set.HasModuleCall(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}

	// HasResourceInstance tests
	if input := rAddr("single").Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance); !set.HasResourceInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("count").Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance); !set.HasResourceInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("for_each").Instance(addrs.StringKey("c")).Absolute(addrs.RootModuleInstance); !set.HasResourceInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("single").Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("single", addrs.NoKey)); !set.HasResourceInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("count").Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance.Child("for_each", addrs.StringKey("b"))); !set.HasResourceInstance(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("single").Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance); set.HasResourceInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := rAddr("single").Instance(addrs.StringKey("")).Absolute(addrs.RootModuleInstance); set.HasResourceInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := rAddr("count").Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance); set.HasResourceInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := rAddr("count").Instance(addrs.StringKey("")).Absolute(addrs.RootModuleInstance); set.HasResourceInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := rAddr("count").Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance); set.HasResourceInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := rAddr("single").Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("single", addrs.IntKey(0))); set.HasResourceInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := rAddr("count").Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance.Child("for_each", addrs.StringKey("a"))); set.HasResourceInstance(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}

	// HasResource tests
	if input := rAddr("single").Absolute(addrs.RootModuleInstance); !set.HasResource(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("count").Absolute(addrs.RootModuleInstance); !set.HasResource(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("for_each").Absolute(addrs.RootModuleInstance); !set.HasResource(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("single").Absolute(addrs.RootModuleInstance.Child("single", addrs.NoKey)); !set.HasResource(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("count").Absolute(addrs.RootModuleInstance.Child("for_each", addrs.StringKey("a"))); !set.HasResource(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("count").Absolute(addrs.RootModuleInstance.Child("for_each", addrs.StringKey("b"))); !set.HasResource(input) {
		t.Errorf("missing %T %s", input, input.String())
	}
	if input := rAddr("nonexist").Absolute(addrs.RootModuleInstance); set.HasResource(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}
	if input := rAddr("count").Absolute(addrs.RootModuleInstance.Child("for_each", addrs.StringKey("nonexist"))); set.HasResource(input) {
		t.Errorf("unexpected %T %s", input, input.String())
	}

}
