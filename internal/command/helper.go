package command

import (
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func getSchemas(c *Meta, state *states.State, config *configs.Config) (*terraform.Schemas, tfdiags.Diagnostics) {
	var schemas *terraform.Schemas
	var diags tfdiags.Diagnostics
	opts, err := c.contextOpts()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}
	tfCtx, ctxDiags := terraform.NewContext(opts)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		return nil, diags
	}
	var schemaDiags tfdiags.Diagnostics
	schemas, schemaDiags = tfCtx.Schemas(config, state)
	diags = diags.Append(schemaDiags)
	if schemaDiags.HasErrors() {
		return nil, diags
	}
	return schemas, diags
}
