// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package plans

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestChangesSyncResourceInstanceChanges(t *testing.T) {
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider("test"),
	}
	mustAddr := func(s string) addrs.AbsResourceInstance {
		addr, diags := addrs.ParseAbsResourceInstanceStr(s)
		if diags.HasErrors() {
			t.Fatal(diags.Err())
		}
		return addr
	}

	changes := NewChanges()
	cs := changes.SyncWrapper()
	cs.AppendResourceInstanceChange(&ResourceInstanceChange{
		Addr:         mustAddr("test_thing.a"),
		PrevRunAddr:  mustAddr("test_thing.a"),
		ProviderAddr: providerAddr,
		Change: Change{
			Action: Create,
			Before: cty.NullVal(cty.Object(map[string]cty.Type{"id": cty.String})),
			After:  cty.ObjectVal(map[string]cty.Value{"id": cty.UnknownVal(cty.String)}),
		},
	})
	cs.AppendResourceInstanceChange(&ResourceInstanceChange{
		Addr:         mustAddr("test_thing.b"),
		PrevRunAddr:  mustAddr("test_thing.b"),
		DeposedKey:   addrs.DeposedKey("00000001"),
		ProviderAddr: providerAddr,
		Change: Change{
			Action: Delete,
			Before: cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("old")}),
			After:  cty.NullVal(cty.Object(map[string]cty.Type{"id": cty.String})),
		},
	})
	cs.AppendResourceInstanceChange(&ResourceInstanceChange{
		Addr:         mustAddr("data.test_thing.c"),
		PrevRunAddr:  mustAddr("data.test_thing.c"),
		ProviderAddr: providerAddr,
		Change: Change{
			Action: Read,
			Before: cty.NullVal(cty.Object(map[string]cty.Type{"id": cty.String})),
			After:  cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("c")}),
		},
	})

	// Reading is still allowed after the changes are closed.
	cs.Close()

	got := cs.ResourceInstanceChanges()
	if len(got) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(got))
	}
	wantAddrs := []string{"test_thing.a", "test_thing.b", "data.test_thing.c"}
	for i, want := range wantAddrs {
		if got[i].Addr.String() != want {
			t.Errorf("change %d: got address %s, want %s", i, got[i].Addr, want)
		}
	}
	if got[1].DeposedKey != addrs.DeposedKey("00000001") {
		t.Errorf("expected the deposed change to keep its deposed key, got %q", got[1].DeposedKey)
	}

	// The results are copies, so modifying them doesn't affect the changes.
	got[0].Action = Delete
	got[0].Addr = mustAddr("test_thing.z")
	again := cs.ResourceInstanceChanges()
	if again[0].Action != Create || again[0].Addr.String() != "test_thing.a" {
		t.Errorf("modifying a result changed the recorded change: %s %s", again[0].Addr, again[0].Action)
	}
	if changes.Resources[0].Action != Create {
		t.Errorf("modifying a result changed the underlying changes: %s", changes.Resources[0].Action)
	}
}

func TestChangesSyncResourceInstanceChanges_empty(t *testing.T) {
	cs := NewChanges().SyncWrapper()
	if got := cs.ResourceInstanceChanges(); len(got) != 0 {
		t.Fatalf("expected no changes, got %d", len(got))
	}
}
