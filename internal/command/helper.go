package command

import (
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const failedToLoadSchemasMessage = `
Terraform failed to load schemas, which will in turn affect its ability to generate the
external JSON state file. This will not have any adverse effects on Terraforms ability
to maintain state information, but may have adverse effects on any external integrations
relying on this format. The file should be created on the next successful "terraform apply"
however, historic state information may be missing if the affected integration relies on that

%s
`

func isCloudMode(b backend.Enhanced) bool {
	_, ok := b.(*cloud.Cloud)

	return ok
}

func getSchemas(c *Meta, state *states.State, config *configs.Config) (*terraform.Schemas, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if config != nil || state != nil {
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
		schemas, schemaDiags := tfCtx.Schemas(config, state)
		diags = diags.Append(schemaDiags)
		if schemaDiags.HasErrors() {
			return nil, diags
		}
		return schemas, diags

	}
	return nil, diags
}
