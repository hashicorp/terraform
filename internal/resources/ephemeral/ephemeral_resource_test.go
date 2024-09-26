// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestResources(t *testing.T) {
	resources := NewResources()

	ephemA0 := addrs.ResourceInstance{
		Resource: addrs.Resource{
			Mode: addrs.EphemeralResourceMode,
			Type: "test",
			Name: "a",
		},
		Key: addrs.IntKey(0),
	}.Absolute(addrs.RootModuleInstance)
	ephemA1 := addrs.ResourceInstance{
		Resource: addrs.Resource{
			Mode: addrs.EphemeralResourceMode,
			Type: "test",
			Name: "a",
		},
		Key: addrs.IntKey(1),
	}.Absolute(addrs.RootModuleInstance)

	ephemB := addrs.ResourceInstance{
		Resource: addrs.Resource{
			Mode: addrs.EphemeralResourceMode,
			Type: "test",
			Name: "b",
		},
		Key: addrs.NoKey,
	}.Absolute(addrs.RootModuleInstance)

	ctx := context.TODO()

	notifyRenew := make(chan string, 10)

	testA0 := &testResourceInstance{
		name:          ephemA0.String(),
		renewInterval: 10 * time.Millisecond,
		notifyRenew:   notifyRenew,
	}
	testA1 := &testResourceInstance{
		name:          ephemA1.String(),
		renewInterval: 10 * time.Millisecond,
		notifyRenew:   notifyRenew,
	}
	testB := &testResourceInstance{}

	resources.RegisterInstance(ctx, ephemA0, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[0]"),
		}),
		Impl:    testA0,
		RenewAt: time.Now().Add(10 * time.Millisecond),
	})

	resources.RegisterInstance(ctx, ephemA1, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[1]"),
		}),
		Impl:    testA1,
		RenewAt: time.Now().Add(10 * time.Millisecond),
	})

	resources.RegisterInstance(ctx, ephemB, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.b"),
		}),
		Impl: testB,
	})

	// Make sure these are renewed the first time as expected from registration,
	// and at least one additional time as requested by the instance.
	renewed := map[string]int{}
	for range 4 {
		a := <-notifyRenew
		renewed[a]++
	}

	if renewed[ephemA0.String()] != 2 {
		t.Fatalf("%s not renewed at least twice as expected", ephemA0)
	}
	if renewed[ephemA1.String()] != 2 {
		t.Fatalf("%s not renewed at least twice as expected", ephemA1)
	}

	for _, addr := range []addrs.AbsResourceInstance{ephemA0, ephemA1, ephemB} {
		val, live := resources.InstanceValue(addr)
		if !live {
			t.Fatalf("%s should be live", addr)
		}
		want := cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal(addr.String()),
		})
		if !want.RawEquals(val) {
			t.Fatalf("wanted: %#v\ngot: %#v\n", want, val)
		}
	}

	testB.Lock()
	if testB.renewed > 0 {
		t.Fatalf("%s should not be renewed", ephemB)
	}
	testB.Unlock()

	// close all instances, which should indicate the values are no longer "live"
	diags := resources.CloseInstances(ctx, ephemA0.ConfigResource())
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}
	diags = resources.CloseInstances(ctx, ephemB.ConfigResource())
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	if !testA0.closed {
		t.Fatalf("%s not closed", ephemA0)
	}
	if !testA1.closed {
		t.Fatalf("%s not closed", ephemA1)
	}
	if !testB.closed {
		t.Fatalf("%s not closed", ephemB)
	}

	for _, addr := range []addrs.AbsResourceInstance{ephemA0, ephemA1, ephemB} {
		val, live := resources.InstanceValue(addr)
		if live {
			t.Fatalf("%s should not be live", addr)
		}
		if !val.RawEquals(cty.DynamicVal) {
			t.Fatalf("unexpected value %#v\n", val)
		}
	}
}

func TestResourcesCancellation(t *testing.T) {
	resources := NewResources()

	ephemA0 := addrs.ResourceInstance{
		Resource: addrs.Resource{
			Mode: addrs.EphemeralResourceMode,
			Type: "test",
			Name: "a",
		},
		Key: addrs.IntKey(0),
	}.Absolute(addrs.RootModuleInstance)
	ephemA1 := addrs.ResourceInstance{
		Resource: addrs.Resource{
			Mode: addrs.EphemeralResourceMode,
			Type: "test",
			Name: "a",
		},
		Key: addrs.IntKey(1),
	}.Absolute(addrs.RootModuleInstance)

	ephemB := addrs.ResourceInstance{
		Resource: addrs.Resource{
			Mode: addrs.EphemeralResourceMode,
			Type: "test",
			Name: "b",
		},
		Key: addrs.NoKey,
	}.Absolute(addrs.RootModuleInstance)

	ctx, cancel := context.WithCancel(context.Background())
	// cancelling now should cause the first renew op to report the cancellation
	cancel()

	testA0 := &testResourceInstance{
		renewInterval: time.Second,
	}
	testA1 := &testResourceInstance{
		renewInterval: time.Second,
	}
	testB := &testResourceInstance{}

	resources.RegisterInstance(ctx, ephemA0, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[0]"),
		}),
		Impl:    testA0,
		RenewAt: time.Now().Add(10 * time.Millisecond),
	})

	resources.RegisterInstance(ctx, ephemA1, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[1]"),
		}),
		Impl:    testA1,
		RenewAt: time.Now().Add(10 * time.Millisecond),
	})

	resources.RegisterInstance(ctx, ephemB, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.b"),
		}),
		Impl: testB,
	})

	// Use the internal WaitGroup to catch when the renew goroutines have exited from the cancellation.
	cancelled := make(chan int)
	go func() {
		resources.wg.Wait()
		close(cancelled)
	}()
	select {
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for cancellation")
	case <-cancelled:
		// this should be almost immediate, but we'll allow a second just in
		// case of crazy slow integration test hosts
	}

	// ephemB has no call for renew, so shouldn't know about the cancel yet
	_, live := resources.InstanceValue(ephemB)
	if !live {
		t.Fatalf("%s should still be live", ephemB)
	}

	for _, addr := range []addrs.AbsResourceInstance{ephemA0, ephemA1} {
		_, live := resources.InstanceValue(addr)
		if live {
			t.Fatalf("%s was canceled, should not be live", addr)
		}
	}

	testB.Lock()
	if testB.renewed > 0 {
		t.Fatalf("%s should not be renewed", ephemB)
	}
	testB.Unlock()

	// close all instances, which should indicate the values are no longer "live"
	diags := resources.CloseInstances(ctx, ephemA0.ConfigResource())
	if len(diags) != 2 {
		t.Fatalf("expected 2 error diagnostics, got:\n%s", diags.ErrWithWarnings())
	}
	diagStr := diags.Err().Error()
	if strings.Count(diagStr, "context canceled") != 2 {
		t.Fatal("expected 2 context canceled errors, got:\n", diagStr)
	}

	diags = resources.CloseInstances(ctx, ephemB.ConfigResource())
	if diags.HasErrors() {
		t.Fatal(diags.ErrWithWarnings())
	}

	if !testA0.closed {
		t.Fatalf("%s not closed", ephemA0)
	}
	if !testA1.closed {
		t.Fatalf("%s not closed", ephemA1)
	}
	if !testB.closed {
		t.Fatalf("%s not closed", ephemB)
	}
}

type testResourceInstance struct {
	sync.Mutex
	name          string
	renewInterval time.Duration
	renewed       int
	notifyRenew   chan string
	closed        bool
}

func (r *testResourceInstance) Renew(ctx context.Context, req providers.EphemeralRenew) (*providers.EphemeralRenew, tfdiags.Diagnostics) {
	nextRenew := &providers.EphemeralRenew{
		RenewAt: time.Now().Add(r.renewInterval),
	}
	r.Lock()
	defer r.Unlock()
	r.renewed++
	select {
	case r.notifyRenew <- r.name:
	case <-time.After(time.Second):
		// stop renewing if no-one is listening
		return nil, nil
	}
	return nextRenew, nil
}

func (r *testResourceInstance) Close(ctx context.Context) tfdiags.Diagnostics {
	r.Lock()
	defer r.Unlock()
	r.closed = true
	return nil
}
