// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform/version"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// If this environment variable is set to "otlp" when running Terraform CLI
// then we'll enable an experimental OTLP trace exporter.
//
// BEWARE! This is not a committed external interface.
//
// Everything about this is experimental and subject to change in future
// releases. Do not depend on anything about the structure of this output.
// This mechanism might be removed altogether if a different strategy seems
// better based on experience with this experiment.
const openTelemetryExporterEnvVar = "OTEL_TRACES_EXPORTER"

// tracer is the OpenTelemetry tracer to use for traces in package main only.
var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/hashicorp/terraform")
}

// openTelemetryInit initializes the optional OpenTelemetry exporter.
//
// By default we don't export telemetry information at all, since Terraform is
// a CLI tool and so we don't assume we're running in an environment with
// a telemetry collector available.
//
// However, for those running Terraform in automation we allow setting
// the standard OpenTelemetry environment variable OTEL_TRACES_EXPORTER=otlp
// to enable an OTLP exporter, which is in turn configured by all of the
// standard OTLP exporter environment variables:
//
//	https://opentelemetry.io/docs/specs/otel/protocol/exporter/#configuration-options
//
// We don't currently support any other telemetry export protocols, because
// OTLP has emerged as a de-facto standard and each other exporter we support
// means another relatively-heavy external dependency. OTLP happens to use
// protocol buffers and gRPC, which Terraform would depend on for other reasons
// anyway.
func openTelemetryInit() error {
	// We'll check the environment variable ourselves first, because the
	// "autoexport" helper we're about to use is built under the assumption
	// that exporting should always be enabled and so will expect to find
	// an OTLP server on localhost if no environment variables are set at all.
	if os.Getenv(openTelemetryExporterEnvVar) != "otlp" {
		return nil // By default we just discard all telemetry calls
	}

	otelResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("Terraform CLI"),
		semconv.ServiceVersionKey.String(version.Version),
	)

	// If the environment variable was set to explicitly enable telemetry
	// then we'll enable it, using the "autoexport" library to automatically
	// handle the details based on the other OpenTelemetry standard environment
	// variables.
	exp, err := autoexport.NewSpanExporter(context.Background())
	if err != nil {
		return err
	}
	sp := sdktrace.NewSimpleSpanProcessor(exp)
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sp),
		sdktrace.WithResource(otelResource),
	)
	otel.SetTracerProvider(provider)

	pgtr := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(pgtr)

	return nil
}

// extractParentTraceContext returns a context that carries any trace context
// propagated to this CLI invocation by a parent process (for example,
// tfc-agent running "terraform plan" with policy enabled) via the standard
// W3C environment variables TRACEPARENT and TRACESTATE, plus OpenTelemetry
// BAGGAGE.
//
// This is what lets a "terraform <args>" root span -- and therefore every
// policy span emitted beneath it -- become a child of the caller's span,
// stitching the agent and core traces into a single tree.
//
// Extraction uses the globally-installed TextMapPropagator, which is only
// configured when telemetry export is enabled (see openTelemetryInit). When
// telemetry is disabled, or when no trace-context variables are present in
// the environment, the supplied context is returned unchanged so the command
// simply starts a brand-new root span as before.
func extractParentTraceContext(ctx context.Context) context.Context {
	// The W3C TraceContext/Baggage propagators use lower-case carrier keys
	// ("traceparent", "tracestate", "baggage"); the corresponding environment
	// variables are conventionally upper-case.
	carrier := propagation.MapCarrier{}
	for _, key := range []string{"traceparent", "tracestate", "baggage"} {
		if v := os.Getenv(strings.ToUpper(key)); v != "" {
			carrier[key] = v
		}
	}
	if len(carrier) == 0 {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
