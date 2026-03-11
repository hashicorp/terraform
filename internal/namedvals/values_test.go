// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package namedvals

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestValues(t *testing.T) {
	// The behavior of [values] is the same for all named value address types,
	// and so we'll just use local values here as a placeholder and assume
	// that input variables and output values would also work.

	// The following addresses are taking some liberties with which combinations
	// would actually be possible in practice with a real Terraform
	// configuration -- unknowns and knowns cannot typically mix at the same
	// known-expansion module prefix -- but the abstraction in this package
	// doesn't aim to enforce those rules, and so we can expect it to be a
	// little more flexible here than it really needs to be, as a way to
	// reduce the amount of test setup we need.
	childInst0 := addrs.ModuleInstance{{Name: "child", InstanceKey: addrs.IntKey(0)}}
	childInst1 := addrs.ModuleInstance{{Name: "child", InstanceKey: addrs.IntKey(1)}}
	childInst2 := addrs.ModuleInstance{{Name: "child", InstanceKey: addrs.IntKey(2)}}
	childInstUnk := addrs.RootModuleInstance.UnexpandedChild(addrs.ModuleCall{Name: "child"})
	grandchildInst0_0 := childInst0.Child("grandchild", addrs.IntKey(0))
	grandchildInst1_unk := childInst1.UnexpandedChild(addrs.ModuleCall{Name: "grandchild"})
	grandchildInst2_unk := childInst2.UnexpandedChild(addrs.ModuleCall{Name: "grandchild"})
	grandchildInstUnk_unk := childInstUnk.Child(addrs.ModuleCall{Name: "grandchild"})

	inRoot := addrs.LocalValue{Name: "in_root"}.Absolute(addrs.RootModuleInstance)
	inChild0 := addrs.LocalValue{Name: "in_child"}.Absolute(childInst0)
	inChildUnk := addrs.ObjectInPartialExpandedModule(childInstUnk, addrs.LocalValue{Name: "in_child"})
	inGrandchild0_0 := addrs.LocalValue{Name: "in_grandchild"}.Absolute(grandchildInst0_0)
	inGrandchild1_unk := addrs.ObjectInPartialExpandedModule(grandchildInst1_unk, addrs.LocalValue{Name: "in_grandchild"})
	inGrandchild2_unk := addrs.ObjectInPartialExpandedModule(grandchildInst2_unk, addrs.LocalValue{Name: "in_grandchild"})
	inGrandchildUnk_unk := addrs.ObjectInPartialExpandedModule(grandchildInstUnk_unk, addrs.LocalValue{Name: "in_grandchild"})

	vals := newValues[addrs.LocalValue, addrs.AbsLocalValue]()
	vals.SetExactResult(inRoot, cty.StringVal("in root"))
	vals.SetExactResult(inChild0, cty.StringVal("in child 0"))
	vals.SetExactResult(inGrandchild0_0, cty.StringVal("in grandchild 0, 0"))
	vals.SetPlaceholderResult(inChildUnk, cty.StringVal("placeholder for all unknown instances of child"))
	vals.SetPlaceholderResult(inGrandchild1_unk, cty.StringVal("placeholder for all unknown instances of child 1 grandchild"))
	vals.SetPlaceholderResult(inGrandchildUnk_unk, cty.StringVal("placeholder for all unknown instances of child unknown grandchild"))

	t.Run("exact values", func(t *testing.T) {
		// Exact values require the given address to exactly match something
		// that was registered.

		if got, want := vals.GetExactResult(inRoot), cty.StringVal("in root"); !want.RawEquals(got) {
			t.Errorf("wrong exact value for %s\ngot:  %#v\nwant: %#v", inRoot, got, want)
		}
		if got, want := vals.GetExactResult(inChild0), cty.StringVal("in child 0"); !want.RawEquals(got) {
			t.Errorf("wrong exact value for %s\ngot:  %#v\nwant: %#v", inChild0, got, want)
		}
		if got, want := vals.GetExactResult(inGrandchild0_0), cty.StringVal("in grandchild 0, 0"); !want.RawEquals(got) {
			t.Errorf("wrong exact value for %s\ngot:  %#v\nwant: %#v", inGrandchild0_0, got, want)
		}
	})
	t.Run("placeholder values", func(t *testing.T) {
		// Placeholder values are selected by longest-prefix pattern matching,
		// and so we can ask both for specific prefixes we registered above
		// and for more specific prefixes that we didn't register but yet
		// still match a less-specific address.

		if got, want := vals.GetPlaceholderResult(inChildUnk), cty.StringVal("placeholder for all unknown instances of child"); !want.RawEquals(got) {
			// This one exactly matches one of the address patterns we registered.
			t.Errorf("wrong exact value for %s\ngot:  %#v\nwant: %#v", inChildUnk, got, want)
		}
		if got, want := vals.GetPlaceholderResult(inGrandchild1_unk), cty.StringVal("placeholder for all unknown instances of child 1 grandchild"); !want.RawEquals(got) {
			// This one exactly matches one of the address patterns we registered.
			t.Errorf("wrong exact value for %s\ngot:  %#v\nwant: %#v", inGrandchild1_unk, got, want)
		}
		if got, want := vals.GetPlaceholderResult(inGrandchild2_unk), cty.StringVal("placeholder for all unknown instances of child unknown grandchild"); !want.RawEquals(got) {
			// This one falls back to the placeholder for when the child
			// instance key isn't known, because there isn't a more specific
			// placeholder available.
			t.Errorf("wrong exact value for %s\ngot:  %#v\nwant: %#v", inGrandchild2_unk, got, want)
		}
		if got, want := vals.GetPlaceholderResult(inGrandchildUnk_unk), cty.StringVal("placeholder for all unknown instances of child unknown grandchild"); !want.RawEquals(got) {
			// This one falls back to the placeholder for when the child
			// instance key isn't known, because we can't know which of
			// the more specific placeholders to select.
			t.Errorf("wrong exact value for %s\ngot:  %#v\nwant: %#v", inGrandchildUnk_unk, got, want)
		}
	})
}
