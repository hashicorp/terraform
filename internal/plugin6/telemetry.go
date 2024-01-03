// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin6

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// CombineTracingContext combines the tracing context with the context handling the cancellation.
// This is used to ensure that the tracing context is cancelled when the context handling the
// cancellation is cancelled.
func CombineTracingContext(ctx, tracingCtx context.Context) context.Context {
	if ctx == nil {
		return tracingCtx
	}

	if tracingCtx == nil {
		return ctx
	}

	return trace.ContextWithSpan(ctx, trace.SpanFromContext(tracingCtx))
}
