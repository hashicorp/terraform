// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
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
	tracer = otel.Tracer("github.com/hashicorp/terraform/internal/command")

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

func findSingleCommandRunSpan(t *testing.T, exp *tracetest.InMemoryExporter, commandName string) tracetest.SpanStub {
	t.Helper()

	deadline := time.Now().Add(250 * time.Millisecond)
	for {
		spans := findCommandTelemetrySpans(exp, func(span tracetest.SpanStub) bool {
			return span.Name == commandRunSpanName(commandName)
		})
		if len(spans) == 1 {
			return spans[0]
		}
		if len(spans) > 1 {
			t.Fatalf("wrong number of command run spans for %q\ngot: %d\nwant: 1\nall spans: %#v", commandName, len(spans), exp.GetSpans())
		}
		if time.Now().After(deadline) {
			t.Fatalf("wrong number of command run spans for %q\ngot: 0\nwant: 1\nall spans: %#v", commandName, exp.GetSpans())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertCommandRunSpanParent(t *testing.T, span tracetest.SpanStub, parent trace.SpanContext) {
	t.Helper()

	if got, want := span.Parent.SpanID(), parent.SpanID(); got != want {
		t.Fatalf("command span parent mismatch\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := span.SpanContext.TraceID(), parent.TraceID(); got != want {
		t.Fatalf("command span trace mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func assertCommandRunSpanAttrs(t *testing.T, span tracetest.SpanStub, commandName string) {
	t.Helper()

	attrs := commandTelemetryAttributesMap(span.Attributes)
	if got, want := attrs[commandRunSpanAttrCommandName], commandName; got != want {
		t.Fatalf("wrong command span attr %q\n got: %#v\nwant: %#v", commandRunSpanAttrCommandName, got, want)
	}
	if _, exists := attrs["terraform.command.args"]; exists {
		t.Fatalf("unexpected raw args attribute recorded: %#v", attrs["terraform.command.args"])
	}
	if _, exists := attrs["terraform.cli.args"]; exists {
		t.Fatalf("unexpected raw CLI args attribute recorded: %#v", attrs["terraform.cli.args"])
	}
}

func assertCommandRunSpanStatusNotError(t *testing.T, span tracetest.SpanStub) {
	t.Helper()

	if got := span.Status.Code; got == codes.Error {
		t.Fatalf("expected non-error span status, got error (%q)", span.Status.Description)
	}
}

func assertCommandRunSpanHasEvent(t *testing.T, span tracetest.SpanStub, eventName string) {
	t.Helper()

	for _, name := range commandTelemetryEventNames(span) {
		if name == eventName {
			return
		}
	}
	t.Fatalf("missing event %q on span; got events %#v", eventName, commandTelemetryEventNames(span))
}

func assertCommandRunSpanNoEvent(t *testing.T, span tracetest.SpanStub, eventName string) {
	t.Helper()

	for _, name := range commandTelemetryEventNames(span) {
		if name == eventName {
			t.Fatalf("unexpected event %q on span; got events %#v", eventName, commandTelemetryEventNames(span))
		}
	}
}
