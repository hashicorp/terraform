// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// initCommandTelemetryForTest configures an in-memory exporter for tests in the
// command package. This mutates global OpenTelemetry state and therefore must
// not be used from parallel tests.
func initCommandTelemetryForTest(t *testing.T, providerOptions ...sdktrace.TracerProviderOption) *tracetest.InMemoryExporter {
	t.Helper()

	exp := tracetest.NewInMemoryExporter()
	sp := sdktrace.NewSimpleSpanProcessor(exp)
	providerOptions = append(
		[]sdktrace.TracerProviderOption{
			sdktrace.WithSpanProcessor(sp),
		},
		providerOptions...,
	)
	provider := sdktrace.NewTracerProvider(providerOptions...)
	otel.SetTracerProvider(provider)

	pgtr := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(pgtr)

	t.Cleanup(func() {
		provider.Shutdown(context.Background())
		otel.SetTracerProvider(nil)
		otel.SetTextMapPropagator(nil)
	})

	return exp
}

func findCommandTelemetrySpan(t *testing.T, exp *tracetest.InMemoryExporter, predicate func(tracetest.SpanStub) bool) tracetest.SpanStub {
	t.Helper()

	for _, span := range exp.GetSpans() {
		if predicate(span) {
			return span
		}
	}
	t.Fatal("no spans matched the predicate")
	return tracetest.SpanStub{}
}

func findCommandTelemetrySpans(exp *tracetest.InMemoryExporter, predicate func(tracetest.SpanStub) bool) tracetest.SpanStubs {
	var spans tracetest.SpanStubs
	for _, span := range exp.GetSpans() {
		if predicate(span) {
			spans = append(spans, span)
		}
	}
	return spans
}

func commandTelemetryAttributesMap(kvs []attribute.KeyValue) map[string]any {
	ret := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		ret[string(kv.Key)] = kv.Value.AsInterface()
	}
	return ret
}

func commandTelemetryEventNames(span tracetest.SpanStub) []string {
	ret := make([]string, 0, len(span.Events))
	for _, event := range span.Events {
		ret = append(ret, event.Name)
	}
	return ret
}
