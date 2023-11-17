// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"go.opentelemetry.io/otel/codes"
)

// ValidateAll checks the validation rules for declared in the configuration
// and returns any diagnostics returned by any of those checks.
//
// This function starts its own [promising.MainTask] and so is a good entry
// point for external callers that don't deal with promises directly themselves,
// encapsulating all of the promise-related implementation details.
//
// This function must be called with a context that belongs to a task started
// from the "promising" package, or else it will immediately panic.
func (m *Main) ValidateAll(ctx context.Context) tfdiags.Diagnostics {
	diags, err := promising.MainTask(ctx, func(ctx context.Context) (tfdiags.Diagnostics, error) {
		// The idea here is just to iterate over everything in the configuration,
		// find its corresponding evaluation object, and then ask it to validate
		// itself. We make all of these calls asynchronously so that everything
		// can get started and then downstream calls will block on promise
		// resolution to achieve the correct evaluation order.
		ws, complete := newWalkState()

		// Our generic static walker is built to support the more advanced
		// needs of the plan walk which produces streaming results through
		// an "output" object. We don't need that here so we'll just stub
		// it out as a zero-length type.
		walk := &walkWithOutput[struct{}]{
			out:   struct{}{},
			state: ws,
		}

		walkStaticObjects(
			ctx, walk, m,
			func(ctx context.Context, walk *walkWithOutput[struct{}], obj StaticEvaler) {
				m.walkValidateObject(ctx, walk.state, obj)
			},
		)

		return complete(), nil
	})
	diags = diags.Append(diagnosticsForPromisingTaskError(err, m))
	return finalDiagnosticsFromEval(diags)
}

// walkValidateObject arranges for any given [Validatable] object to be
// asynchronously validated, reporting any of its diagnostics to the
// [walkState].
//
// Just like the [Validatable] interface itself, this performs only shallow
// validation of the direct content of the given object. For object types
// that have child objects the caller must also discover each of those and
// arrange for them to be validated by a separate call to this method.
func (m *Main) walkValidateObject(ctx context.Context, ws *walkState, obj Validatable) {
	ws.AsyncTask(ctx, func(ctx context.Context) {
		ctx, span := tracer.Start(ctx, obj.tracingName()+" validation")
		defer span.End()
		diags := obj.Validate(ctx)
		ws.AddDiags(diags)
		if diags.HasErrors() {
			span.SetStatus(codes.Error, "validation returned errors")
		}
	})
}
