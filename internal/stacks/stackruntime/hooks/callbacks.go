// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hooks

import (
	"context"
)

// BeginFunc is the signature of a callback for a hook which begins a
// series of related events.
//
// The given context is guaranteed to preserve the values from whichever
// context was passed to the top-level [stackruntime.Plan] or
// [stackruntime.Apply] call.
//
// The hook callback may return any value, and that value will be passed
// verbatim to the corresponding [MoreFunc]. A typical use for that
// extra arbitrary value would be to begin a tracing span in "Begin" and then
// either adding events to or ending that span in "More".
//
// If a particular "begin" hook isn't implemented but one of its "more" hooks
// is implemented then the extra tracking value will always be nil when
// the first "more" hook runs.
type BeginFunc[Msg any] func(context.Context, Msg) any

// MoreFunc is the signature of a callback for a hook which reports
// ongoing process or completion of a multi-step process previously reported
// using a [HookFuncBegin] callback.
//
// The given context is guaranteed to preserve the values from whichever
// context was passed to the top-level [stackruntime.Plan] or
// [stackruntime.Apply] call.
//
// The hook callback recieves an additional argument which is guaranteed to be
// the same value returned from the corresponding [BeginFunc]. See
// [BeginFunc]'s documentation for more information.
//
// If the overall hooks also defines a [ContextAttachFunc] then a context
// descended from its result will be passed into the [MoreFunc] for any events
// related to the operation previously signalled by [BeginFunc].
//
// The hook callback may optionally return a new arbitrary tracking value. If
// the return value is non-nil then it replaces the original value for future
// hooks belonging to the same context. If it's nil then the previous value
// is retained.
//
// MoreFunc is also sometimes used in isolation for one-shot events,
// in which case the extra value will always be nil unless stated otherwise
// in a particular hook's documentation.
type MoreFunc[Msg any] func(context.Context, any, Msg) any

// ContextAttachFunc is the signature of an optional callback that knows
// how to bind an arbitrary tracking value previously returned by a [BeginFunc]
// to the values of a [context.Context] so that the tracking value can be
// made available to downstream operations outside the direct scope of the
// stack runtime, such as external HTTP requests.
//
// Use this if your [BeginFunc]s return something that should be visible to
// all context-aware operations within the scope of the operation that was
// begun.
//
// If you use this then your related [MoreFunc] callbacks for the same event
// should always return nil, because there is no way to mutate the context
// with a new tracking value after the fact.
type ContextAttachFunc func(parent context.Context, tracking any) context.Context

// SingleFunc is the signature of a callback for a hook which operates in
// isolation, and has no related or enclosed events.
//
// The given context is guaranteed to preserve the values from whichever
// context was passed to the top-level [stackruntime.Plan] or
// [stackruntime.Apply] call.
type SingleFunc[Msg any] func(context.Context, Msg)
