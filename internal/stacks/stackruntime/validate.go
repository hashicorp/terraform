// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"

	"go.opentelemetry.io/otel/codes"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Validate performs static validation of a full stack configuration, returning
// diagnostics in case of any detected problems.
func Validate(ctx context.Context, req *ValidateRequest) tfdiags.Diagnostics {
	ctx, span := tracer.Start(ctx, "validate stack configuration")
	defer span.End()

	main := stackeval.NewForValidating(req.Config, stackeval.ValidateOpts{
		ProviderFactories: req.ProviderFactories,
	})
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
	Config            *stackconfig.Config
	ProviderFactories map[addrs.Provider]providers.Factory

	ExperimentsAllowed bool
}
