package atlas

import (
	"fmt"

	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/configschema"
)

// Backend is an implementation of Backend that exists only to return a
// somewhat-helpful error to anyone who still has a configuration using
// the obsolete "atlas" backend.
type Backend struct {
}

var _ backend.Backend = (*Backend)(nil)

// New returns a new initialized Atlas backend.
func New() *Backend {
	return &Backend{}
}

func (b *Backend) ConfigSchema() *configschema.Block {
	// NOTE: We have this here just so existing configurations can still
	// pass initial schema validation, and then get to PrepareConfig where
	// we'll return our specialized error about obsolescence.
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"name": {
				Type:        cty.String,
				Required:    true,
				Description: "Full name of the environment in Terraform Enterprise, such as 'myorg/myenv'",
			},
			"access_token": {
				Type:        cty.String,
				Optional:    true,
				Description: "Access token to use to access Terraform Enterprise; the ATLAS_TOKEN environment variable is used if this argument is not set",
			},
			"address": {
				Type:        cty.String,
				Optional:    true,
				Description: "Base URL for your Terraform Enterprise installation; the ATLAS_ADDRESS environment variable is used if this argument is not set, finally falling back to a default of 'https://atlas.hashicorp.com/' if neither are set.",
			},
		},
	}
}

func (b *Backend) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		`The "atlas" backend is obsolete`,
		`HashiCorp Atlas has reached end of life and so its corresponding Terraform backend is no longer available. If you have migrated to Terraform Cloud or Terraform Enterprise, use the "remote" backend instead.`,
		nil, // an empty path refers to the containing block itself
	))
	return obj, diags
}

func (b *Backend) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.AttributeValue(
		tfdiags.Error,
		`The "atlas" backend is obsolete`,
		`HashiCorp Atlas has reached end of life and so its corresponding Terraform backend is no longer available. If you have migrated to Terraform Cloud or Terraform Enterprise, use the "remote" backend instead.`,
		nil, // an empty path refers to the containing block itself
	))
	return diags
}

func (b *Backend) Workspaces() ([]string, error) {
	return nil, fmt.Errorf("the atlas backend is obsolete")
}

func (b *Backend) DeleteWorkspace(name string) error {
	return fmt.Errorf("the atlas backend is obsolete")
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	return nil, fmt.Errorf("the atlas backend is obsolete")
}
