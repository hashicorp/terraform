// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSetSortedNatural(t *testing.T) {
	// We're using AbsResourceInstance here just because it happens to
	// implement the required Less method, but this is intended as a test
	// of SetSortedNatural itself, not of any particular type's Less
	// implementation.
	boop1 := AbsResourceInstance{
		Module: RootModuleInstance,
		Resource: ResourceInstance{
			Resource: Resource{
				Mode: ManagedResourceMode,
				Type: "test",
				Name: "boop1",
			},
			Key: NoKey,
		},
	}
	boop2 := AbsResourceInstance{
		Module: RootModuleInstance,
		Resource: ResourceInstance{
			Resource: Resource{
				Mode: ManagedResourceMode,
				Type: "test",
				Name: "boop2",
			},
			Key: NoKey,
		},
	}
	boop3 := AbsResourceInstance{
		Module: RootModuleInstance,
		Resource: ResourceInstance{
			Resource: Resource{
				Mode: ManagedResourceMode,
				Type: "test",
				Name: "boop3",
			},
			Key: NoKey,
		},
	}
	s := MakeSet(
		boop3,
		boop2,
		boop1,
	)

	got := SetSortedNatural(s)
	want := []AbsResourceInstance{
		boop1,
		boop2,
		boop3,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}
