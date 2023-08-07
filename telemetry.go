// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"os"

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
