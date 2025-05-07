// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
)

// Hooks is an optional API for external callers to be notified about various
// progress events during plan and apply operations.
//
// This type is exposed to external callers through a type alias in package
// stackruntime, and so it is part of the public API of that package despite
// being defined in here.
type Hooks struct {
	// BeginPlan is called at the very start of a stack plan operation,
	// encompassing that entire operation to allow establishing a top-level
	// tracing context for that operation.
	//
	// BeginPlan does not provide any additional data, because no work has
	// happened yet.
	BeginPlan hooks.BeginFunc[struct{}]

	// EndPlan marks the end of the overall planning process started at
	// [Hooks.BeginPlan]. If [Hooks.BeginPlan] opened a tracing span then
	// this EndPlan should end it.
	//
	// EndPlan does not provide any additional data, because all relevant
	// information is provided by other means.
	EndPlan hooks.MoreFunc[struct{}]

	// BeginApply is called at the very start of a stack apply operation,
	// encompassing that entire operation to allow establishing a top-level
	// tracing context for that operation.
	//
	// BeginApply does not provide any additional data, because no work has
	// happened yet.
	BeginApply hooks.BeginFunc[struct{}]

	// EndApply marks the end of the overall apply process started at
	// [Hooks.BeginApply]. If [Hooks.BeginApply] opened a tracing span then
	// this EndApply should end it.
	//
	// EndApply does not provide any additional data, because all relevant
	// information is provided by other means.
	EndApply hooks.MoreFunc[struct{}]

	// ComponentExpanded is called when a plan operation evaluates the
	// expansion argument for a component, resulting in zero or more instances.
	ComponentExpanded hooks.SingleFunc[*hooks.ComponentInstances]

	// PendingComponentInstancePlan is called at the start of the plan
	// operation, before evaluating the component instance's inputs and
	// providers.
	PendingComponentInstancePlan hooks.SingleFunc[stackaddrs.AbsComponentInstance]

	// BeginComponentInstancePlan is called when the component instance's
	// inputs and providers are ready and planning begins, and can be used to
	// establish a nested tracing context wrapping the plan operation.
	BeginComponentInstancePlan hooks.BeginFunc[stackaddrs.AbsComponentInstance]

	// EndComponentInstancePlan is called when the component instance plan
	// started at [Hooks.BeginComponentInstancePlan] completes successfully. If
	// a context is established by [Hooks.BeginComponentInstancePlan] then this
	// hook should end it.
	EndComponentInstancePlan hooks.MoreFunc[stackaddrs.AbsComponentInstance]

	// ErrorComponentInstancePlan is similar to [Hooks.EndComponentInstancePlan], but
	// is called when the plan operation failed.
	ErrorComponentInstancePlan hooks.MoreFunc[stackaddrs.AbsComponentInstance]

	// DeferComponentInstancePlan is similar to [Hooks.EndComponentInstancePlan], but
	// is called when the plan operation succeeded but signaled a deferral.
	DeferComponentInstancePlan hooks.MoreFunc[stackaddrs.AbsComponentInstance]

	// PendingComponentInstanceApply is called at the start of the apply
	// operation.
	PendingComponentInstanceApply hooks.SingleFunc[stackaddrs.AbsComponentInstance]

	// BeginComponentInstanceApply is called when the component instance starts
	// applying the plan, and can be used to establish a nested tracing context
	// wrapping the apply operation.
	BeginComponentInstanceApply hooks.BeginFunc[stackaddrs.AbsComponentInstance]

	// EndComponentInstanceApply is called when the component instance plan
	// started at [Hooks.BeginComponentInstanceApply] completes successfully. If
	// a context is established by [Hooks.BeginComponentInstanceApply] then
	// this hook should end it.
	EndComponentInstanceApply hooks.MoreFunc[stackaddrs.AbsComponentInstance]

	// ErrorComponentInstanceApply is similar to [Hooks.EndComponentInstanceApply], but
	// is called when the apply operation failed.
	ErrorComponentInstanceApply hooks.MoreFunc[stackaddrs.AbsComponentInstance]

	// ReportResourceInstanceStatus is called when a resource instance's status
	// changes during a plan or apply operation. It should be called inside a
	// tracing context established by [Hooks.BeginComponentInstancePlan] or
	// [Hooks.BeginComponentInstanceApply].
	ReportResourceInstanceStatus hooks.MoreFunc[*hooks.ResourceInstanceStatusHookData]

	// ReportResourceInstanceProvisionerStatus is called when a provisioner for
	// a resource instance begins or ends. It should be called inside a tracing
	// context established by [Hooks.BeginComponentInstanceApply].
	ReportResourceInstanceProvisionerStatus hooks.MoreFunc[*hooks.ResourceInstanceProvisionerHookData]

	// ReportResourceInstanceDrift is called after a component instance's plan
	// determines that a resource instance has experienced changes outside of
	// Terraform. It should be called inside a tracing context established by
	// [Hooks.BeginComponentInstancePlan].
	ReportResourceInstanceDrift hooks.MoreFunc[*hooks.ResourceInstanceChange]

	// ReportResourceInstancePlanned is called after a component instance's
	// plan results in proposed changes for a resource instance. It should be
	// called inside a tracing context established by
	// [Hooks.BeginComponentInstancePlan].
	ReportResourceInstancePlanned hooks.MoreFunc[*hooks.ResourceInstanceChange]

	// ReportResourceInstanceDeferred is called after a component instance's
	// plan results in a resource instance being deferred. It should be called
	// inside a tracing context established by
	// [Hooks.BeginComponentInstancePlan].
	ReportResourceInstanceDeferred hooks.MoreFunc[*hooks.DeferredResourceInstanceChange]

	// ReportComponentInstancePlanned is called after a component instance
	// is planned. It should be called inside a tracing context established by
	// [Hooks.BeginComponentInstancePlan].
	ReportComponentInstancePlanned hooks.MoreFunc[*hooks.ComponentInstanceChange]

	// ReportComponentInstanceApplied is called after a component instance
	// plan is applied. It should be called inside a tracing context
	// established by [Hooks.BeginComponentInstanceApply].
	ReportComponentInstanceApplied hooks.MoreFunc[*hooks.ComponentInstanceChange]

	// ContextAttach is an optional callback for wrapping a non-nil value
	// returned by a [hooks.BeginFunc] into a [context.Context] to be passed
	// to other context-aware operations that descend from the operation that
	// was begun.
	//
	// See the docs for [hooks.ContextAttachFunc] for more information.
	ContextAttach hooks.ContextAttachFunc
}

// A do-nothing default Hooks that we use when the caller doesn't provide one.
var noHooks = &Hooks{}

// ContextWithHooks returns a context that carries the given [Hooks] as
// one of its values.
func ContextWithHooks(parent context.Context, hooks *Hooks) context.Context {
	return context.WithValue(parent, hooksContextKey{}, hooks)
}

func hooksFromContext(ctx context.Context) *Hooks {
	hooks, ok := ctx.Value(hooksContextKey{}).(*Hooks)
	if !ok {
		return noHooks
	}
	return hooks
}

type hooksContextKey struct{}

// hookSeq is a small helper for keeping track of a sequence of hooks related
// to the same multi-step action.
//
// It retains the hook implementer's arbitrary tracking values between calls
// so as to reduce the visual noise and complexity of our main evaluation code.
// Once a hook sequence has begun using a "begin" callback, it's safe to run
// subsequent hooks concurrently from multiple goroutines, although from
// the caller's perspective that will make the propagation of changes to their
// tracking values appear unpredictable.
type hookSeq struct {
	tracking any
	mu       sync.Mutex
}

// hookBegin begins a hook sequence by calling a [hooks.BeginFunc] callback.
//
// The result can be used with [hookMore] to report ongoing progress or
// completion of whatever multi-step process has begun.
//
// This function also deals with the optional [hook.ContextAttachFunc] that
// hook implementers may provide. If it's non-nil then the returned context
// is the result of that function. Otherwise it is the same context provided
// by the caller.

// Callers should use the returned context for all subsequent context-aware
// calls that are related to whatever multi-step operation this hook sequence
// represents, so that the hook subscriber can use this mechanism to propagate
// distributed tracing spans to downstream operations. Callers MUST also use
// descendants of the resulting context for any subsequent calls to
// [runHookBegin] using the returned [hookSeq].
func hookBegin[Msg any](ctx context.Context, cb hooks.BeginFunc[Msg], ctxCb hooks.ContextAttachFunc, msg Msg) (*hookSeq, context.Context) {
	tracking := runHookBegin(ctx, cb, msg)
	if ctxCb != nil {
		ctx = ctxCb(ctx, tracking)
	}
	return &hookSeq{
		tracking: tracking,
	}, ctx
}

// hookMore continues a hook sequence by calling a [hooks.MoreFunc] callback
// using the tracking state retained by the given [hookSeq].
//
// It's safe to use [hookMore] with the same [hookSeq] from multiple goroutines
// concurrently, and it's guaranteed that no two hooks will run concurrently
// within the same sequence, but it'll be unpredictable from the caller's
// standpoint which order the hooks will occur.
func hookMore[Msg any](ctx context.Context, seq *hookSeq, cb hooks.MoreFunc[Msg], msg Msg) {
	// We hold the lock throughout the hook call so that callers don't need
	// to worry about concurrent calls to their hooks and so that the
	// propagation of the arbitrary "tracking" values from one hook to the
	// next will always exact follow the sequence of the calls.
	seq.mu.Lock()
	seq.tracking = runHookMore(ctx, cb, seq.tracking, msg)
	seq.mu.Unlock()
}

// hookSingle calls an isolated [hooks.SingleFunc] callback, if it is non-nil.
func hookSingle[Msg any](ctx context.Context, cb hooks.SingleFunc[Msg], msg Msg) {
	if cb != nil {
		cb(ctx, msg)
	}
}

// runHookBegin is a lower-level helper that just directly runs a given
// callback if it isn't nil and returns its result. If the given callback is
// nil then runHookBegin immediately returns nil.
func runHookBegin[Msg any](ctx context.Context, cb hooks.BeginFunc[Msg], msg Msg) any {
	if cb == nil {
		return nil
	}
	return cb(ctx, msg)
}

// runHookMore is a lower-level helper that just directly runs a given
// callback if it isn't nil and returns the effective new tracking value,
// which may or may not be the same value passed as "tracking".
// If the given callback is nil then runHookMore immediately returns the given
// tracking value.
func runHookMore[Msg any](ctx context.Context, cb hooks.MoreFunc[Msg], tracking any, msg Msg) any {
	if cb == nil {
		// We'll retain any existing tracking value, then.
		return tracking
	}
	newTracking := cb(ctx, tracking, msg)
	if newTracking != nil {
		return newTracking
	}
	return tracking
}
