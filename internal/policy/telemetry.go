// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// tracer returns the OpenTelemetry tracer used by the policy client.
//
// Resolved lazily on every call so the global TracerProvider installed by
// `openTelemetryInit` (in package main, which runs after this package's
// `init`) is reflected in the tracer used at runtime.
func tracer() trace.Tracer {
	return otel.Tracer("github.com/hashicorp/terraform/internal/policy")
}
