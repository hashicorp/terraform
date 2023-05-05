// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package states

import (
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestResourceInstanceObject_encode(t *testing.T) {
	value := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.True,
	})
	// The in-memory order of resource dependencies is random, since they're an
	// unordered set.
	depsOne := []addrs.ConfigResource{
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "honk"),
		addrs.RootModule.Child("child").Resource(addrs.ManagedResourceMode, "test", "flub"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "boop"),
	}
	depsTwo := []addrs.ConfigResource{
		addrs.RootModule.Child("child").Resource(addrs.ManagedResourceMode, "test", "flub"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "boop"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "honk"),
	}

	// multiple instances may have been assigned the same deps slice
	objs := []*ResourceInstanceObject{
		&ResourceInstanceObject{
			Value:        value,
			Status:       ObjectPlanned,
			Dependencies: depsOne,
		},
		&ResourceInstanceObject{
			Value:        value,
			Status:       ObjectPlanned,
			Dependencies: depsTwo,
		},
		&ResourceInstanceObject{
			Value:        value,
			Status:       ObjectPlanned,
			Dependencies: depsOne,
		},
		&ResourceInstanceObject{
			Value:        value,
			Status:       ObjectPlanned,
			Dependencies: depsOne,
		},
	}

	var encoded []*ResourceInstanceObjectSrc

	// Encoding can happen concurrently, so we need to make sure the shared
	// Dependencies are safely handled
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, obj := range objs {
		obj := obj
		wg.Add(1)
		go func() {
			defer wg.Done()
			rios, err := obj.Encode(value.Type(), 0)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			mu.Lock()
			encoded = append(encoded, rios)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// However, identical sets of dependencies should always be written to state
	// in an identical order, so we don't do meaningless state updates on refresh.
	for i := 0; i < len(encoded)-1; i++ {
		if diff := cmp.Diff(encoded[i].Dependencies, encoded[i+1].Dependencies); diff != "" {
			t.Errorf("identical dependencies got encoded in different orders:\n%s", diff)
		}
	}
}
