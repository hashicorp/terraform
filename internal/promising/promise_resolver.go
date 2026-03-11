// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package promising

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PromiseResolver is an object representing responsibility for a promise, which
// can be passed between tasks to delegate responsibility and then eventually
// be used to provide the promise's final results.
type PromiseResolver[T any] struct {
	p *promise
}

// Resolve provides the final results for a promise.
//
// This may be called only from the task that is currently responsible for
// the promise
func (pr PromiseResolver[T]) Resolve(ctx context.Context, v T, err error) {
	callerT := mustTaskFromContext(ctx)
	if pr.p.responsible.Load() != callerT {
		panic("promise resolved by incorrect task")
	}
	resolvePromise(pr.p, v, err)

	resolvingTaskSpan := trace.SpanFromContext(ctx)
	resolvingTaskSpanContext := resolvingTaskSpan.SpanContext()
	promiseSpanContext := pr.p.traceSpan.SpanContext()
	pr.p.traceSpan.AddEvent(
		"resolved",
		trace.WithAttributes(
			attribute.String("promising.resolved_by", resolvingTaskSpanContext.SpanID().String()),
		),
	)
	resolvingTaskSpan.AddEvent(
		"resolved a promise",
		trace.WithAttributes(
			attribute.String("promising.resolved_id", promiseSpanContext.SpanID().String()),
		),
	)
	pr.p.traceSpan.End()
}

func (pr PromiseResolver[T]) PromiseID() PromiseID {
	return PromiseID{pr.p}
}

// promise implements AnyPromiseResolver.
func (pr PromiseResolver[T]) promise() *promise {
	return pr.p
}

// AnnounceContainedPromises implements PromiseContainer for a single naked
// promise resolver.
func (pr PromiseResolver[T]) AnnounceContainedPromises(cb func(AnyPromiseResolver)) {
	cb(pr)
}

// AnyPromiseResolver is an interface implemented by all [PromiseResolver]
// instantiations, regardless of result type.
//
// Callers should typically not type-assert an AnyPromiseResolver into an
// instance of [PromiseResolver] unless the caller is the task that is
// currently responsible for resolving the promise.
type AnyPromiseResolver interface {
	promise() *promise
}
