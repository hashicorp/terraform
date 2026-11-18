// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"fmt"
	"slices"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestChangesSyncOutputChanges(t *testing.T) {
	root := addrs.RootModuleInstance
	child := root.Child("child", addrs.IntKey(0))

	outputChange := func(addr addrs.AbsOutputValue, val string) *OutputChange {
		return &OutputChange{
			Addr: addr,
			Change: Change{
				Action: Create,
				Before: cty.NullVal(cty.String),
				After:  cty.StringVal(val),
			},
		}
	}

	// gotAddrs returns the addresses of the live output changes, in order.
	gotAddrs := func(changes *Changes) []string {
		var ret []string
		for _, oc := range changes.Outputs {
			if oc != nil {
				ret = append(ret, oc.Addr.String())
			}
		}
		return ret
	}

	t.Run("get and remove", func(t *testing.T) {
		changes := NewChanges()
		cs := changes.SyncWrapper()

		a := root.OutputValue("a")
		b := child.OutputValue("b")
		c := root.OutputValue("c")
		cs.AppendOutputChange(outputChange(a, "a"))
		cs.AppendOutputChange(outputChange(b, "b"))
		cs.AppendOutputChange(outputChange(c, "c"))

		if got := cs.GetOutputChange(b); got == nil || got.After.AsString() != "b" {
			t.Fatalf("wrong change for %s: %#v", b, got)
		}
		if got := cs.GetOutputChange(child.OutputValue("missing")); got != nil {
			t.Fatalf("unexpected change for missing output: %#v", got)
		}

		cs.RemoveOutputChange(b)
		if got := cs.GetOutputChange(b); got != nil {
			t.Fatalf("unexpected change for removed output: %#v", got)
		}
		// removing a missing change does nothing
		cs.RemoveOutputChange(b)

		if got := cs.GetOutputChange(c); got == nil || got.After.AsString() != "c" {
			t.Fatalf("wrong change for %s: %#v", c, got)
		}
		if got, want := gotAddrs(changes), []string{"output.a", "output.c"}; !slices.Equal(got, want) {
			t.Fatalf("wrong output changes\ngot:  %s\nwant: %s", got, want)
		}

		// The removed change must not be visible to readers of Changes.
		if got := changes.OutputValue(b); got != nil {
			t.Fatalf("unexpected change for removed output: %#v", got)
		}
		if got := len(changes.RootOutputValues()); got != 2 {
			t.Fatalf("wrong number of root output changes: %d", got)
		}
		if got := len(changes.OutputValues(root, addrs.ModuleCall{Name: "child"})); got != 0 {
			t.Fatalf("wrong number of child output changes: %d", got)
		}
		src, err := changes.Encode(nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(src.Outputs) != 2 || src.Outputs[0].Addr.String() != "output.a" || src.Outputs[1].Addr.String() != "output.c" {
			t.Fatalf("wrong encoded output changes: %#v", src.Outputs)
		}

		// replacing a change moves it to the end, as before
		cs.RemoveOutputChange(a)
		cs.AppendOutputChange(outputChange(a, "new"))
		if got := cs.GetOutputChange(a); got == nil || got.After.AsString() != "new" {
			t.Fatalf("wrong change for %s: %#v", a, got)
		}
		if got, want := gotAddrs(changes), []string{"output.c", "output.a"}; !slices.Equal(got, want) {
			t.Fatalf("wrong output changes\ngot:  %s\nwant: %s", got, want)
		}

		cs.Close()
		if slices.Contains(changes.Outputs, nil) {
			t.Fatalf("output changes not compacted on close: %#v", changes.Outputs)
		}
		if got := cs.GetOutputChange(c); got == nil || got.After.AsString() != "c" {
			t.Fatalf("wrong change for %s after close: %#v", c, got)
		}
	})

	t.Run("compaction preserves order", func(t *testing.T) {
		changes := NewChanges()
		cs := changes.SyncWrapper()

		var want []string
		for i := range 20 {
			addr := root.OutputValue(fmt.Sprintf("o%02d", i))
			cs.AppendOutputChange(outputChange(addr, addr.String()))
			if i%3 != 0 {
				want = append(want, addr.String())
			}
		}
		for i := 0; i < 20; i += 3 {
			cs.RemoveOutputChange(root.OutputValue(fmt.Sprintf("o%02d", i)))
		}
		// remove most of the rest to force compaction
		for i := 1; i < 16; i++ {
			if i%3 == 0 {
				continue
			}
			addr := root.OutputValue(fmt.Sprintf("o%02d", i))
			cs.RemoveOutputChange(addr)
			want = slices.DeleteFunc(want, func(s string) bool { return s == addr.String() })
		}

		if got := gotAddrs(changes); !slices.Equal(got, want) {
			t.Fatalf("wrong output changes\ngot:  %s\nwant: %s", got, want)
		}
		if holes := len(changes.Outputs) - len(want); holes > len(changes.Outputs)/2 {
			t.Fatalf("output changes were not compacted: %d holes in %d", holes, len(changes.Outputs))
		}
		for _, s := range want {
			addr, diags := addrs.ParseAbsOutputValueStr(s)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			if got := cs.GetOutputChange(addr); got == nil || got.After.AsString() != s {
				t.Fatalf("wrong change for %s: %#v", s, got)
			}
		}
	})

	t.Run("existing changes", func(t *testing.T) {
		a := root.OutputValue("a")
		b := root.OutputValue("b")
		changes := &Changes{
			Outputs: []*OutputChange{
				outputChange(a, "a"),
				outputChange(b, "b"),
			},
		}
		cs := changes.SyncWrapper()

		if got := cs.GetOutputChange(b); got == nil || got.After.AsString() != "b" {
			t.Fatalf("wrong change for %s: %#v", b, got)
		}
		cs.RemoveOutputChange(a)
		if got := cs.GetOutputChange(a); got != nil {
			t.Fatalf("unexpected change for removed output: %#v", got)
		}
		if got, want := gotAddrs(changes), []string{"output.b"}; !slices.Equal(got, want) {
			t.Fatalf("wrong output changes\ngot:  %s\nwant: %s", got, want)
		}
	})

	t.Run("duplicate changes", func(t *testing.T) {
		changes := NewChanges()
		cs := changes.SyncWrapper()

		a := root.OutputValue("a")
		cs.AppendOutputChange(outputChange(a, "first"))
		cs.AppendOutputChange(outputChange(root.OutputValue("b"), "b"))
		cs.AppendOutputChange(outputChange(a, "second"))

		if got := cs.GetOutputChange(a); got == nil || got.After.AsString() != "first" {
			t.Fatalf("wrong change for %s: %#v", a, got)
		}
		cs.RemoveOutputChange(a)
		if got := cs.GetOutputChange(a); got == nil || got.After.AsString() != "second" {
			t.Fatalf("wrong change for %s: %#v", a, got)
		}
		cs.RemoveOutputChange(a)
		if got := cs.GetOutputChange(a); got != nil {
			t.Fatalf("unexpected change for removed output: %#v", got)
		}
		if got, want := gotAddrs(changes), []string{"output.b"}; !slices.Equal(got, want) {
			t.Fatalf("wrong output changes\ngot:  %s\nwant: %s", got, want)
		}
	})
}
