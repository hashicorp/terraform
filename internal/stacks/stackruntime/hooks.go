// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"

	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
)

// This file exposes a small part of the API surface of "stackeval" to external
// callers. We need to orient it this way because package stackeval cannot
// itself depend on stackruntime.

// Hooks is an optional mechanism for callers to get streaming notifications
// of various kinds of events that can occur during plan and apply operations.
//
// To use this, construct a Hooks object and then wrap it in a [context.Context]
// using the [ContextWithHooks] function, and then use that context (or another
// context derived from it that inherits the values) when calling into
// [Plan] or [Apply].
//
// All of the callback fields in Hooks are optional and may be left as nil
// if the caller has no interest in a particular event.
//
// The events exposed by Hooks are intended for ancillary use-cases like
// realtime UI updates, and so a caller that is only concerned with the primary
// results of an operation can safely ignore this and just consume the direct
// results from the [Plan] and [Apply] functions as described in their
// own documentation.
//
// Hook functions should typically run to completion quickly to avoid noticable
// delays to the progress of the operation being monitored. In particular,
// if a hook implementation is sending data to a network service then the
// actual transmission of the events should be decoupled from the notifications,
// such as by using a buffered channel as a FIFO queue and ideally transmitting
// the events in batches where possible.
type Hooks = stackeval.Hooks

// ContextWithHooks returns a context that carries the given [Hooks] as
// one of its values.
//
// Pass the resulting context -- or a descendant that preserves the values --
// to [Plan] or [Apply] to be notified when the different hookable events
// occur during that plan or apply process.
func ContextWithHooks(parent context.Context, hooks *Hooks) context.Context {
	return stackeval.ContextWithHooks(parent, hooks)
}
