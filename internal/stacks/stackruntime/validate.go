package stackruntime

import (
	"context"

	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Validate performs static validation of a full stack configuration, returning
// diagnostics in case of any detected problems.
func Validate(ctx context.Context, req *ValidateRequest) tfdiags.Diagnostics {
	ctx, span := tracer.Start(ctx, "validate stack configuration")
	defer span.End()

	main := stackeval.NewForValidating(req.Config, stackeval.ValidateOpts{})
	return main.ValidateAll(ctx)
}

type ValidateRequest struct {
	Config *stackconfig.Config

	// TODO: Provider factories and other similar such things
}
