// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

//lint:file-ignore U1000 Some utilities in here are intentionally unused in VCS but are for temporary use while debugging a test.

import (
	"context"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/setup"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// initTelemetryForTest configures OpenTelemetry to collect spans into a
// local in-memory buffer and returns an object that provides access to that
// buffer.
//
// The OpenTelemetry tracer provider is a global cross-cutting concern shared
// throughout the program, so it isn't valid to use this function in any test
// that calls t.Parallel, or in subtests of a parent test that has already
// used this function.
func initTelemetryForTest(t *testing.T, providerOptions ...sdktrace.TracerProviderOption) *tracetest.InMemoryExporter {
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

	// We'll automatically shut down the provider at the end of the test run,
	// because otherwise a subsequent test which runs something that generates
	// telemetry _without_ calling initTelemetryForTest (which is optional)
	// could end up appending irrelevant spans to an earlier test's exporter.
	t.Cleanup(func() {
		provider.Shutdown(context.Background())
		otel.SetTracerProvider(nil)
		otel.SetTextMapPropagator(nil)
	})

	t.Log("OpenTelemetry initialized")
	return exp
}

// findTestTelemetrySpan tests each of the spans that have been reported to the
// given [tracetest.InMemoryExporter] with the given predicate function and
// returns the first one for which the predicate matches.
//
// If the predicate returns false for all spans then this function will fail
// the test using the given [testing.T].
func findTestTelemetrySpan(t *testing.T, exp *tracetest.InMemoryExporter, predicate func(tracetest.SpanStub) bool) tracetest.SpanStub {
	for _, span := range exp.GetSpans() {
		if predicate(span) {
			return span
		}
	}
	t.Fatal("no spans matched the predicate")
	return tracetest.SpanStub{}
}

// findTestTelemetrySpans tests each of the spans that have been reported to the
// given [tracetest.InMemoryExporter] with the given predicate function and
// returns only those for which the predicate matches.
//
// If no spans match at all then the result is a zero-length slice. If you are
// expecting to find exactly one matching span then [findTestTelemetrySpan]
// (singular) might be more convenient.
func findTestTelemetrySpans(t *testing.T, exp *tracetest.InMemoryExporter, predicate func(tracetest.SpanStub) bool) tracetest.SpanStubs {
	var ret tracetest.SpanStubs
	for _, span := range exp.GetSpans() {
		if predicate(span) {
			ret = append(ret, span)
		}
	}
	return ret
}

// overwriteTestSpanTimestamps overwrites the timestamps in all of the given
// spans to be exactly the given fakeTime, as a way to avoid considering exact
// timestamps when comparing actual spans with desired spans.
//
// This function overwrites both the start and end times of the spans themselves
// and also the timestamps of any events associated with the spans.
func overwriteTestSpanTimestamps(spans tracetest.SpanStubs, fakeTime time.Time) {
	for i := range spans {
		spans[i].StartTime = fakeTime
		spans[i].EndTime = fakeTime
		for j := range spans[i].Events {
			spans[i].Events[j].Time = fakeTime
		}
	}
}

func fixedTraceID(n uint32) trace.TraceID {
	return trace.TraceID{
		0xfe, 0xed, 0xfa, 0xce,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), uint8(n >> 0),
	}
}

func fixedSpanID(n uint32) trace.SpanID {
	return trace.SpanID{
		0xfa, 0xce, 0xfe, 0xed,
		uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), uint8(n >> 0),
	}
}

func TestTelemetryInTests(t *testing.T) {
	ctx := context.Background()

	testResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("telemetry test"),
		semconv.ServiceVersionKey.String("1.2.3"),
	)

	telemetry := initTelemetryForTest(t,
		sdktrace.WithResource(testResource),
	)

	var parentSpanContext, childSpanContext trace.SpanContext

	tracer := otel.Tracer("test thingy")
	{
		ctx, parentSpan := tracer.Start(ctx, "parent span")
		parentSpanContext = parentSpan.SpanContext()
		{
			_, childSpan := tracer.Start(ctx, "child span")
			childSpanContext = childSpan.SpanContext()
			childSpan.AddEvent("did something totally hilarious")
			childSpan.SetStatus(codes.Error, "it went wrong")
			childSpan.End()
		}
		parentSpan.End()
	}

	gotSpans := telemetry.GetSpans()

	// The spans contain real timestamps that make them annoying to compare,
	// so we'll just replace those with fixed timestamps so we can easily
	// compare everything else.
	fakeTime := time.Now()
	overwriteTestSpanTimestamps(gotSpans, fakeTime)

	wantSpans := tracetest.SpanStubs{
		// These are ordered by the calls to Span.End above, so child should
		// always appear first. (That's a detail of this in-memory-only
		// exporter, not a general guarantee about OpenTracing.)
		{
			Name:        "child span",
			SpanContext: childSpanContext,
			Parent:      parentSpanContext,
			SpanKind:    trace.SpanKindInternal,
			StartTime:   fakeTime,
			EndTime:     fakeTime,
			Events: []sdktrace.Event{
				{
					Name: "did something totally hilarious",
					Time: fakeTime,
				},
			},
			Status: sdktrace.Status{
				Code:        codes.Error,
				Description: "it went wrong",
			},
			Resource: testResource,
			InstrumentationLibrary: instrumentation.Scope{
				Name: "test thingy",
			},
		},
		{
			Name:           "parent span",
			SpanContext:    parentSpanContext,
			SpanKind:       trace.SpanKindInternal,
			StartTime:      fakeTime,
			EndTime:        fakeTime,
			ChildSpanCount: 1,
			Resource:       testResource,
			InstrumentationLibrary: instrumentation.Scope{
				Name: "test thingy",
			},
		},
	}

	if diff := cmp.Diff(wantSpans, gotSpans); diff != "" {
		t.Errorf("wrong spans\n%s", diff)
	}
}

func TestTelemetryInTestsGRPC(t *testing.T) {
	ctx := context.Background()

	testResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("TestTelemetryInTestsGRPC"),
	)
	telemetry := initTelemetryForTest(t,
		sdktrace.WithResource(testResource),
	)

	client, close := grpcClientForTesting(ctx, t, func(srv *grpc.Server) {
		server := &setupServer{
			initOthers: func(ctx context.Context, cc *setup.Handshake_Request, stopper *stopper) (*setup.ServerCapabilities, error) {
				return &setup.ServerCapabilities{}, nil
			},
		}
		setup.RegisterSetupServer(srv, server)
	})
	defer close()
	setupClient := setup.NewSetupClient(client)

	{
		ctx, span := otel.Tracer("TestTelemetryInTestsGRPC").Start(ctx, "root")
		_, err := setupClient.Handshake(ctx, &setup.Handshake_Request{
			Capabilities: &setup.ClientCapabilities{},
		})
		if err != nil {
			t.Fatal(err)
		}
		span.End()
	}

	clientSpan := findTestTelemetrySpan(t, telemetry, func(ss tracetest.SpanStub) bool {
		return ss.SpanKind == trace.SpanKindClient
	})
	serverSpan := findTestTelemetrySpan(t, telemetry, func(ss tracetest.SpanStub) bool {
		return ss.SpanKind == trace.SpanKindServer
	})
	t.Run("client span", func(t *testing.T) {
		span := clientSpan
		t.Logf("client span: %s", spew.Sdump(span))
		if got, want := span.Name, "terraform1.setup.Setup/Handshake"; got != want {
			t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
		}
		attrs := otelAttributesMap(span.Attributes)
		if got, want := attrs["rpc.system"], "grpc"; got != want {
			t.Errorf("wrong rpc.system\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.service"], "terraform1.setup.Setup"; got != want {
			t.Errorf("wrong rpc.service\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.method"], "Handshake"; got != want {
			t.Errorf("wrong rpc.method\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("server span", func(t *testing.T) {
		span := serverSpan
		t.Logf("server span: %s", spew.Sdump(span))
		if got, want := span.Name, "terraform1.setup.Setup/Handshake"; got != want {
			t.Errorf("wrong name\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := span.Parent.SpanID(), clientSpan.SpanContext.SpanID(); got != want {
			t.Errorf("server span is not a child of the client span\nclient span ID:        %s\nserver span parent ID: %s", want, got)
		}
		if got, want := serverSpan.SpanContext.TraceID(), clientSpan.SpanContext.TraceID(); got != want {
			t.Errorf("server span belongs to different trace than client span\nclient trace ID: %s\nserver trace ID: %s", want, got)
		}
		attrs := otelAttributesMap(span.Attributes)
		if got, want := attrs["rpc.system"], "grpc"; got != want {
			t.Errorf("wrong rpc.system\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.service"], "terraform1.setup.Setup"; got != want {
			t.Errorf("wrong rpc.service\ngot:  %s\nwant: %s", got, want)
		}
		if got, want := attrs["rpc.method"], "Handshake"; got != want {
			t.Errorf("wrong rpc.method\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func otelAttributesMap(kvs []attribute.KeyValue) map[string]any {
	ret := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		ret[string(kv.Key)] = kv.Value.AsInterface()
	}
	return ret
}
