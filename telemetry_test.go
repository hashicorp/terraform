// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestExtractParentTraceContext(t *testing.T) {
	// Install the same propagator that openTelemetryInit configures when
	// telemetry export is enabled, so extraction has something to work with.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	t.Cleanup(func() { otel.SetTextMapPropagator(nil) })

	const (
		traceID = "0af7651916cd43dd8448eb211c80319c"
		spanID  = "b7ad6b7169203331"
	)

	t.Run("adopts the parent from TRACEPARENT", func(t *testing.T) {
		t.Setenv("TRACEPARENT", "00-"+traceID+"-"+spanID+"-01")

		ctx := extractParentTraceContext(context.Background())

		sc := trace.SpanContextFromContext(ctx)
		if !sc.IsValid() {
			t.Fatal("expected a valid span context, got an invalid one")
		}
		if got := sc.TraceID().String(); got != traceID {
			t.Errorf("trace id = %s, want %s", got, traceID)
		}
		if got := sc.SpanID().String(); got != spanID {
			t.Errorf("span id = %s, want %s", got, spanID)
		}
		if !sc.IsRemote() {
			t.Error("expected the extracted span context to be marked remote")
		}
	})

	t.Run("returns the context unchanged when no trace context is present", func(t *testing.T) {
		// Treat empty values as unset.
		t.Setenv("TRACEPARENT", "")
		t.Setenv("TRACESTATE", "")
		t.Setenv("BAGGAGE", "")

		ctx := extractParentTraceContext(context.Background())

		if trace.SpanContextFromContext(ctx).IsValid() {
			t.Error("expected no span context when the environment is empty")
		}
	})
}
