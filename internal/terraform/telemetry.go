// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// tracer returns the OpenTelemetry tracer used by Terraform Core.
//
// Resolved lazily on every call so the global TracerProvider installed by
// the CLI's openTelemetryInit (which runs after this package's init) is
// reflected in the tracer used at runtime.
func tracer() trace.Tracer {
	return otel.Tracer("github.com/hashicorp/terraform/internal/terraform")
}
