// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"context"
	"errors"
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
	mu     sync.RWMutex

	// WaitGroup to track renew goroutines
	wg sync.WaitGroup
}

func NewResources() *Resources {
	return &Resources{
		active: addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *resourceInstanceInternal]](),
	}
}

type ResourceInstanceRegistration struct {
	Value      cty.Value
	ConfigBody hcl.Body
	Impl       ResourceInstance
	RenewAt    time.Time
	Private    []byte
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
	if !reg.RenewAt.IsZero() {
		ctx, cancel := context.WithCancel(ctx)
		ri.renewCancel = cancel

		renewal := &providers.EphemeralRenew{
			RenewAt: reg.RenewAt,
			Private: reg.Private,
		}

		r.wg.Add(1)
		go ri.handleRenewal(ctx, &r.wg, renewal)
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
		// Here we can assume that if the entire resource exists, the instance
		// is valid because Close removes resources as a whole. Individual
		// instances may not actually be present when checks are evaluated,
		// because they are evaluated from instance nodes that are using "self".
		// The way an instance gets "self" is to call GetResource which needs to
		// compile all instances into a suitable value, so we may be missing
		// instances which have not yet been opened.
		return cty.DynamicVal, true
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
	var diags tfdiags.Diagnostics

	// Use a read-lock here so we can run multiple close calls concurrently for
	// different resources. This needs to call CloseEphemeralResource which is sent to
	// the provider and can take an unknown amount of time.
	r.mu.RLock()
	for _, elem := range r.active.Get(configAddr).Elems {
		moreDiags := elem.Value.close(ctx)
		diags = diags.Append(moreDiags.InConfigBody(elem.Value.configBody, elem.Key.String()))
	}
	r.mu.RUnlock()

	// Stop tracking the objects we've just closed, so that we know we don't
	// still need to close them.
	r.mu.Lock()
	defer r.mu.Unlock()
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
	// TODO: Investigate making sure individual close calls are always called
	// even after runtime errors. If individual resources should always be
	// closed before we get here, then we may not need this at all. If we only
	// get here during exceptional circumstances, then we're probably exiting
	// anyway so there's no cleanup needed.
	r.mu.Lock()
	defer r.mu.Unlock()

	// We might be closing due to a context cancellation, but we still need to
	// be able to make non-canceled Close requests.
	//
	// TODO: if we're going to ignore the cancellation to ensure that Close is
	// always called, should we add some sort of timeout?
	ctx = context.WithoutCancel(ctx)

	var diags tfdiags.Diagnostics
	for _, elem := range r.active.Elems {
		for _, elem := range elem.Value.Elems {
			moreDiags := elem.Value.close(ctx)
			diags = diags.Append(moreDiags.InConfigBody(elem.Value.configBody, elem.Key.String()))
		}
	}
	r.active = addrs.MakeMap[addrs.ConfigResource, addrs.Map[addrs.AbsResourceInstance, *resourceInstanceInternal]]()

	// All renew loops should have returned, or else we're going to leak
	// resources which could be continually renewing, or even interfering with
	// the same resources during the next operation.
	//
	// Use an asynchronous check so we can timeout and report the problem.
	done := make(chan int)
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// OK!
	case <-time.After(10 * time.Second):
		// This is probably harmless a lot of time, but is also indicative of an
		// ephemeral resource which would be misbehaving. The message isn't
		// very helpful with no context, so we'll have to rely on correlating
		// the problem via other log messages.
		diags = diags.Append(errors.New("Ephemeral resources failed to Close during renew operations"))
	}

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

	// if the resource could not be opened, there will not be anything to close either
	if r.impl == nil {
		return diags
	}

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
	diags = diags.Append(r.impl.Close(ctx))

	return diags
}

func (r *resourceInstanceInternal) handleRenewal(ctx context.Context, wg *sync.WaitGroup, firstRenewal *providers.EphemeralRenew) {
	defer wg.Done()
	t := time.NewTimer(time.Until(firstRenewal.RenewAt))
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
			t.Reset(time.Until(anotherRenew.RenewAt))
			r.renewMu.Unlock()
		case <-ctx.Done():
			// If we're cancelled then we'll halt renewing immediately.
			r.renewMu.Lock()
			t.Stop()
			r.renewCancel = noopCancel
			r.renewDiags = r.renewDiags.Append(ctx.Err())
			r.renewMu.Unlock()
			return // we don't need to run this loop anymore
		}
	}
}

func noopCancel() {}
