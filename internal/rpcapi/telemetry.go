// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// tracer is the OpenTelemetry tracer to use for tracing for code in this
// package.
//
// When creating tracing spans in gRPC service functions, always use the
// a [context.Context] descended from the one passed in to the service
// function so that the spans can attach to the automatically-generated
// server request span and, if the client is also using OpenTelemetry,
// to the client's request span.
var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("github.com/hashicorp/terraform/internal/rpcapi")
}
