// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"

	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"go.opentelemetry.io/otel/codes"
)

// Validate performs static validation of a full stack configuration, returning
// diagnostics in case of any detected problems.
func Validate(ctx context.Context, req *ValidateRequest) tfdiags.Diagnostics {
	ctx, span := tracer.Start(ctx, "validate stack configuration")
	defer span.End()

	main := stackeval.NewForValidating(req.Config, stackeval.ValidateOpts{})
	main.AllowLanguageExperiments(req.ExperimentsAllowed)
	diags := main.ValidateAll(ctx)
	diags = diags.Append(
		main.DoCleanup(ctx),
	)
	if diags.HasErrors() {
		span.SetStatus(codes.Error, "validation returned errors")
	}
	return diags
}

type ValidateRequest struct {
	Config *stackconfig.Config

	ExperimentsAllowed bool

	// TODO: Provider factories and other similar such things
}
