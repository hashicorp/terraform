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

	testA0 := &testResourceInstance{
		// FIXME: renewals are done one minute early, but this is not validated anywhere
		renewInterval: time.Minute + time.Millisecond,
		// allow some extra space to make sure no unexpected renew calls were made
		notifyRenew: make(chan int, 10),
	}
	testA1 := &testResourceInstance{
		renewInterval: time.Minute + time.Millisecond,
		notifyRenew:   make(chan int, 10),
	}
	testB := &testResourceInstance{}

	resources.RegisterInstance(ctx, ephemA0, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[0]"),
		}),
		Impl:         testA0,
		FirstRenewal: &providers.EphemeralRenew{ExpireTime: time.Now().Add(time.Millisecond)},
	})

	resources.RegisterInstance(ctx, ephemA1, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[1]"),
		}),
		Impl:         testA1,
		FirstRenewal: &providers.EphemeralRenew{ExpireTime: time.Now().Add(time.Millisecond)},
	})

	resources.RegisterInstance(ctx, ephemB, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.b"),
		}),
		Impl: testB,
	})

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

	select {
	case <-testA0.notifyRenew:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for renew")
	}
	select {
	case <-testA1.notifyRenew:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for renew")
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
		renewInterval: 2 * time.Minute,
		// allow some extra space to make sure no unexpected renew calls were made
		notifyRenew: make(chan int, 10),
	}
	testA1 := &testResourceInstance{
		renewInterval: 2 * time.Minute,
		notifyRenew:   make(chan int, 10),
	}
	testB := &testResourceInstance{}

	resources.RegisterInstance(ctx, ephemA0, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[0]"),
		}),
		Impl:         testA0,
		FirstRenewal: &providers.EphemeralRenew{ExpireTime: time.Now().Add(time.Millisecond)},
	})

	resources.RegisterInstance(ctx, ephemA1, ResourceInstanceRegistration{
		Value: cty.ObjectVal(map[string]cty.Value{
			"test": cty.StringVal("ephemeral.test.a[1]"),
		}),
		Impl:         testA1,
		FirstRenewal: &providers.EphemeralRenew{ExpireTime: time.Now().Add(time.Millisecond)},
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
		// this should be almost immediate, but we'll wait for a bit just in
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

	renewInterval time.Duration
	renewed       int
	notifyRenew   chan int
	closed        bool
}

func (r *testResourceInstance) Renew(ctx context.Context, req providers.EphemeralRenew) (*providers.EphemeralRenew, tfdiags.Diagnostics) {
	nextRenew := &providers.EphemeralRenew{
		ExpireTime: time.Now().Add(r.renewInterval),
	}
	r.Lock()
	defer r.Unlock()
	r.renewed++
	select {
	case r.notifyRenew <- r.renewed:
	default:
		panic("blocked on unexpected renew call")
	}

	return nextRenew, nil
}

func (r *testResourceInstance) Close(ctx context.Context) tfdiags.Diagnostics {
	r.Lock()
	defer r.Unlock()
	r.closed = true
	return nil
}
