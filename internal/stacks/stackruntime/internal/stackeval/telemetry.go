package stackeval

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var telemetry trace.Tracer

func init() {
	telemetry = otel.Tracer("github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval")
}
