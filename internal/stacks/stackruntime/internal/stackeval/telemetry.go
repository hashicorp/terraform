// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval")
}

// tracingNamer is implemented by types that can return a suitable name for
// themselves to use in the names or attributes of tracing spans.
type tracingNamer interface {
	tracingName() string
}
