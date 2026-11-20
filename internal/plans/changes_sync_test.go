// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"slices"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
)

func testOutputChange(addr addrs.AbsOutputValue, val string) *OutputChange {
	return &OutputChange{
		Addr: addr,
		Change: Change{
			Action: Create,
			Before: cty.NullVal(cty.String),
			After:  cty.StringVal(val),
		},
	}
}

func testResourceInstanceAddr(mod addrs.ModuleInstance, key addrs.InstanceKey) addrs.AbsResourceInstance {
	return addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test",
		Name: "r",
	}.Instance(key).Absolute(mod)
}

func testResourceInstanceChange(addr addrs.AbsResourceInstance, dk addrs.DeposedKey, val string) *ResourceInstanceChange {
	return &ResourceInstanceChange{
		Addr:        addr,
		PrevRunAddr: addr,
		DeposedKey:  dk,
		ProviderAddr: addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("test"),
		},
		Change: Change{
			Action: Update,
			Before: cty.NullVal(cty.Object(map[string]cty.Type{"value": cty.String})),
			After:  cty.ObjectVal(map[string]cty.Value{"value": cty.StringVal(val)}),
		},
	}
}

func TestChangesSyncOutputChanges(t *testing.T) {
	root := addrs.RootModuleInstance
	child := root.Child("child", addrs.IntKey(0))
	a := root.OutputValue("a")
	b := child.OutputValue("b")

	cs := NewChanges().SyncWrapper()
	cs.AppendOutputChange(testOutputChange(a, "a"))
	cs.AppendOutputChange(testOutputChange(b, "b"))

	if got := cs.GetOutputChange(b); got == nil || got.After.AsString() != "b" {
		t.Fatalf("wrong change for %s: %#v", b, got)
	}
	if got := cs.GetOutputChange(child.OutputValue("missing")); got != nil {
		t.Fatalf("unexpected change for missing output: %#v", got)
	}

	// appending a change for the same output replaces the existing change
	cs.AppendOutputChange(testOutputChange(a, "new"))
	if got := cs.GetOutputChange(a); got == nil || got.After.AsString() != "new" {
		t.Fatalf("wrong change for %s: %#v", a, got)
	}

	cs.RemoveOutputChange(a)
	if got := cs.GetOutputChange(a); got != nil {
		t.Fatalf("unexpected change for removed output: %#v", got)
	}
	// removing a missing change does nothing
	cs.RemoveOutputChange(a)

	if got := cs.GetOutputChange(b); got == nil || got.After.AsString() != "b" {
		t.Fatalf("wrong change for %s: %#v", b, got)
	}
}

func TestChangesSyncResourceChanges(t *testing.T) {
	a := testResourceInstanceAddr(addrs.RootModuleInstance, addrs.StringKey("a"))
	b := testResourceInstanceAddr(addrs.RootModuleInstance, addrs.StringKey("b"))
	const dk = addrs.DeposedKey("00000001")

	cs := NewChanges().SyncWrapper()
	cs.AppendResourceInstanceChange(testResourceInstanceChange(a, addrs.NotDeposed, "a"))
	cs.AppendResourceInstanceChange(testResourceInstanceChange(a, dk, "a-deposed"))
	cs.AppendResourceInstanceChange(testResourceInstanceChange(b, addrs.NotDeposed, "b"))

	value := func(rc *ResourceInstanceChange) string {
		if rc == nil {
			return "<nil>"
		}
		return rc.After.GetAttr("value").AsString()
	}

	if got := value(cs.GetResourceInstanceChange(a, addrs.NotDeposed)); got != "a" {
		t.Fatalf("wrong change for %s: %s", a, got)
	}
	if got := value(cs.GetResourceInstanceChange(a, dk)); got != "a-deposed" {
		t.Fatalf("wrong change for %s deposed object %s: %s", a, dk, got)
	}
	if got := cs.GetResourceInstanceChange(b, dk); got != nil {
		t.Fatalf("unexpected change for missing deposed object: %#v", got)
	}

	// appending a change for the same object replaces the existing change
	cs.AppendResourceInstanceChange(testResourceInstanceChange(b, addrs.NotDeposed, "new"))
	if got := value(cs.GetResourceInstanceChange(b, addrs.NotDeposed)); got != "new" {
		t.Fatalf("wrong change for %s: %s", b, got)
	}

	// removing the current object's change leaves the deposed object's change
	cs.RemoveResourceInstanceChange(a, addrs.NotDeposed)
	if got := cs.GetResourceInstanceChange(a, addrs.NotDeposed); got != nil {
		t.Fatalf("unexpected change for removed instance: %#v", got)
	}
	if got := value(cs.GetResourceInstanceChange(a, dk)); got != "a-deposed" {
		t.Fatalf("wrong change for %s deposed object %s: %s", a, dk, got)
	}
	// removing a missing change does nothing
	cs.RemoveResourceInstanceChange(a, addrs.NotDeposed)

	// the recorded change is a copy of the given change, and callers get a
	// copy of the recorded change
	change := testResourceInstanceChange(a, addrs.NotDeposed, "a")
	cs.AppendResourceInstanceChange(change)
	change.Addr = b
	got := cs.GetResourceInstanceChange(a, addrs.NotDeposed)
	if got == nil || !got.Addr.Equal(a) {
		t.Fatalf("wrong change for %s: %#v", a, got)
	}
	got.Addr = b
	if got := cs.GetResourceInstanceChange(a, addrs.NotDeposed); got == nil || !got.Addr.Equal(a) {
		t.Fatalf("wrong change for %s: %#v", a, got)
	}
}

func TestChangesEncodeOrder(t *testing.T) {
	root := addrs.RootModuleInstance
	child := root.Child("child", addrs.IntKey(0))
	const dk = addrs.DeposedKey("00000001")

	changes := NewChanges()
	cs := changes.SyncWrapper()
	for _, addr := range []addrs.AbsOutputValue{
		child.OutputValue("a"),
		root.OutputValue("b"),
		root.OutputValue("a"),
	} {
		cs.AppendOutputChange(testOutputChange(addr, addr.String()))
	}
	for _, obj := range []addrs.AbsResourceInstanceObject{
		testResourceInstanceAddr(child, addrs.IntKey(0)).CurrentObject(),
		testResourceInstanceAddr(root, addrs.IntKey(10)).CurrentObject(),
		testResourceInstanceAddr(root, addrs.IntKey(2)).DeposedObject(dk),
		testResourceInstanceAddr(root, addrs.IntKey(2)).CurrentObject(),
	} {
		cs.AppendResourceInstanceChange(testResourceInstanceChange(obj.ResourceInstance, obj.DeposedKey, obj.String()))
	}

	schemas := &schemarepo.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): {
				ResourceTypes: map[string]providers.Schema{
					"test": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"value": {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
		},
	}
	src, err := changes.Encode(schemas)
	if err != nil {
		t.Fatal(err)
	}

	var gotOutputs []string
	for _, oc := range src.Outputs {
		gotOutputs = append(gotOutputs, oc.Addr.String())
	}
	wantOutputs := []string{
		"output.a",
		"output.b",
		"module.child[0].output.a",
	}
	if !slices.Equal(gotOutputs, wantOutputs) {
		t.Errorf("wrong output order\ngot:  %q\nwant: %q", gotOutputs, wantOutputs)
	}

	var gotResources []string
	for _, rc := range src.Resources {
		gotResources = append(gotResources, rc.ObjectAddr().String())
	}
	wantResources := []string{
		"test.r[2]",
		"test.r[2] deposed object 00000001",
		"test.r[10]",
		"module.child[0].test.r[0]",
	}
	if !slices.Equal(gotResources, wantResources) {
		t.Errorf("wrong resource order\ngot:  %q\nwant: %q", gotResources, wantResources)
	}
}
