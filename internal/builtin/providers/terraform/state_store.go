package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

var stateStores = map[string]providers.Schema{
	"fs": fsStateStoreSchema(),
}

func fsStateStoreSchema() providers.Schema {
	return providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"path": {
					Type:        cty.String,
					Optional:    true,
					Description: "The path to the tfstate file. This defaults to 'terraform.tfstate' relative to the root module by default.",
				},
				"workspace_dir": {
					Type:        cty.String,
					Optional:    true,
					Description: "The path to non-default workspaces.",
				},
			},
		},
	}
}

func (p *Provider) ValidateStateStoreConfig(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	var resp providers.ValidateStateStoreConfigResponse
	_, ok := stateStores[req.TypeName]
	if !ok {
		// Should not get here if the caller is behaving correctly, because
		// we don't declare any state stores in our schema that we don't have
		// implementations for.
		resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
		return resp
	}

	// attr := ss.Body.AttributeByPath(cty.GetAttrPath("path"))

	// TODO: real validation logic here
	resp.Diagnostics.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "validation warning",
		Detail:   "yolo tada yada",
	})

	return resp
}

func (p *Provider) ConfigureStateStore(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	var resp providers.ConfigureStateStoreResponse
	_, ok := stateStores[req.TypeName]
	if !ok {
		// Should not get here if the caller is behaving correctly, because
		// we don't declare any state stores in our schema that we don't have
		// implementations for.
		resp.Diagnostics = tfdiags.Diagnostics{}
		resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
		return resp
	}

	// TODO: real configuration logic here
	resp.Diagnostics.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "configuration warning",
		Detail:   "yolo tada yada",
	})

	return resp
}

func (p *Provider) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	var resp providers.GetStatesResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}

func (p *Provider) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	var resp providers.DeleteStateResponse
	resp.Diagnostics.Append(fmt.Errorf("unsupported state store type %q", req.TypeName))
	return resp
}
