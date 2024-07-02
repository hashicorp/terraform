// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"

	"github.com/zclconf/go-cty/cty"
)

// Resources is a tracking structure for active instances of ephemeral
// resources.
//
// The lifecycle of an ephemeral resource instance is quite different than
// other resource modes because it's live for at most the duration of a single
// graph walk, and because it might need periodic "renewing" in order to
// remain live for the necessary duration.
type Resources struct {
	active addrs.Map[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *resourceInstanceInternal]]
	mu     sync.Mutex
}

func NewResources() *Resources {
	return &Resources{
		active: addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *resourceInstanceInternal]](),
	}
}

type ResourceInstanceRegistration struct {
	Value        cty.Value
	ConfigBody   hcl.Body
	Impl         ResourceInstance
	FirstRenewal *providers.EphemeralRenew
}

func (r *Resources) RegisterInstance(ctx context.Context, addr addrs.AbsResourceInstance, reg ResourceInstanceRegistration) {
	if addr.Resource.Resource.Mode != addrs.EphemeralResourceMode {
		panic(fmt.Sprintf("can't register %s as an ephemeral resource instance", addr))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	configAddr := addr.ConfigResource()
	if !r.active.Has(configAddr) {
		r.active.Put(configAddr, addrs.MakeMap[addrs.AbsResourceInstance, *resourceInstanceInternal]())
	}
	ri := &resourceInstanceInternal{
		value:       reg.Value,
		configBody:  reg.ConfigBody,
		impl:        reg.Impl,
		renewCancel: noopCancel,
	}
	if reg.FirstRenewal != nil {
		ctx, cancel := context.WithCancel(ctx)
		ri.renewCancel = cancel
		go ri.handleRenewal(ctx, reg.FirstRenewal)
	}
	r.active.Get(configAddr).Put(addr, ri)
}

func (r *Resources) InstanceValue(addr addrs.AbsResourceInstance) (val cty.Value, live bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	configAddr := addr.ConfigResource()
	insts, ok := r.active.GetOk(configAddr)
	if !ok {
		return cty.DynamicVal, false
	}
	inst, ok := insts.GetOk(addr)
	if !ok {
		return cty.DynamicVal, false
	}
	// If renewal has failed then we can't assume that the object is still
	// live, but we can still return the original value regardless.
	return inst.value, !inst.renewDiags.HasErrors()
}

// CloseInstances shuts down any live ephemeral resource instances that are
// associated with the given resource address.
//
// This is the "happy path" way to shut down ephemeral resource instances,
// intended to be called during the visit to a graph node that depends on
// all other nodes that might make use of the instances of this ephemeral
// resource.
//
// The runtime should also eventually call [Resources.Close] once the graph
// walk is complete, to catch any stragglers that we didn't reach for
// piecemeal shutdown, e.g. due to errors during the graph walk.
func (r *Resources) CloseInstances(ctx context.Context, configAddr addrs.ConfigResource) tfdiags.Diagnostics {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: Can we somehow avoid holding the lock for the entire duration?
	// Closing an instance is likely to perform a network request, so this
	// could potentially take a while and block other work from starting.

	var diags tfdiags.Diagnostics
	for _, elem := range r.active.Get(configAddr).Elems {
		moreDiags := elem.Value.close(ctx)
		diags = diags.Append(moreDiags.InConfigBody(elem.Value.configBody, elem.Key.String()))
	}

	// Stop tracking the objects we've just closed, so that we know we don't
	// still need to close them.
	r.active.Remove(configAddr)

	return diags
}

// Close shuts down any ephemeral resource instances that are still running
// at the time of the call.
//
// This is intended to catch any "stragglers" that we weren't able to clean
// up during the graph walk, such as if an error prevents us from reaching
// the cleanup node.
func (r *Resources) Close(ctx context.Context) tfdiags.Diagnostics {
	// FIXME: The following really ought to take into account dependency
	// relationships between what's still running, because it's possible
	// that one ephemeral resource depends on another ephemeral resource
	// to operate correctly, such as if the HashiCorp Vault provider is
	// accessing a secret lease through an SSH tunnel: closing the SSH tunnel
	// before closing the Vault secret lease will make the Vault API
	// unreachable.
	//
	// We'll just ignore that for now since this is just a prototype anyway.

	r.mu.Lock()
	defer r.mu.Unlock()

	// We might be closing due to a context cancellation, but we still need
	// to be able to make non-canceled Close requests.
	ctx = context.WithoutCancel(ctx)

	var diags tfdiags.Diagnostics
	for _, elem := range r.active.Elems {
		for _, elem := range elem.Value.Elems {
			moreDiags := elem.Value.close(ctx)
			diags = diags.Append(moreDiags.InConfigBody(elem.Value.configBody, elem.Key.String()))
		}
	}
	r.active = addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *resourceInstanceInternal]]()
	return diags
}

type resourceInstanceInternal struct {
	value      cty.Value
	configBody hcl.Body
	impl       ResourceInstance

	renewCancel func()
	renewDiags  tfdiags.Diagnostics
	renewMu     sync.Mutex // hold when accessing renewCancel/renewDiags, and while actually renewing
}

// close halts this instance's asynchronous renewal loop, if any, and then
// calls Close on the resource instance's implementation object.
//
// The returned diagnostics are contextual diagnostics that should have
// [tfdiags.Diagnostics.WithConfigBody] called on them before returning to
// a context-unaware caller.
func (r *resourceInstanceInternal) close(ctx context.Context) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Stop renewing, if indeed we are. If we previously saw any errors during
	// renewing then they finally get returned here, to be reported along with
	// any errors during close.
	r.renewMu.Lock()
	r.renewCancel()
	diags = diags.Append(r.renewDiags)
	r.renewDiags = nil // just to avoid any risk of double-reporting
	r.renewMu.Unlock()

	// FIXME: If renewal failed earlier then it's pretty likely that closing
	// would fail too. For now this is assuming that it's the provider's
	// own responsibility to remember that it previously failed a renewal
	// and to avoid returning redundant errors from close, but perhaps we'll
	// revisit that in later work.
	diags = diags.Append(r.impl.Close(context.WithoutCancel(ctx)))

	return diags
}

func (r *resourceInstanceInternal) handleRenewal(ctx context.Context, firstRenewal *providers.EphemeralRenew) {
	t := time.NewTimer(time.Until(firstRenewal.ExpireTime.Add(-60 * time.Second)))
	nextRenew := firstRenewal
	for {
		select {
		case <-t.C:
			// It's time to renew
			r.renewMu.Lock()
			anotherRenew, diags := r.impl.Renew(ctx, *nextRenew)
			r.renewDiags.Append(diags)
			if diags.HasErrors() {
				// If renewal fails then we'll stop trying to renew.
				r.renewCancel = noopCancel
				r.renewMu.Unlock()
				return
			}
			if anotherRenew == nil {
				// If we don't have another round of renew to do then we'll stop.
				r.renewCancel = noopCancel
				r.renewMu.Unlock()
				return
			}
			nextRenew = anotherRenew
			t.Reset(time.Until(anotherRenew.ExpireTime.Add(-60 * time.Second)))
			r.renewMu.Unlock()
		case <-ctx.Done():
			// If we're cancelled then we'll halt renewing immediately.
			r.renewMu.Lock()
			t.Stop()
			r.renewCancel = noopCancel
			r.renewMu.Unlock()
			return // we don't need to run this loop anymore
		}
	}
}

func noopCancel() {}
