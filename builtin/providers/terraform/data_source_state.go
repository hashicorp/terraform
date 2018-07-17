package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/backend"
	backendinit "github.com/hashicorp/terraform/backend/init"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func dataSourceRemoteStateGetSchema() providers.Schema {
	return providers.Schema{
		Block: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"backend": {
					Type:     cty.String,
					Required: true,
				},
				"config": {
					Type:     cty.DynamicPseudoType,
					Optional: true,
				},
				"defaults": {
					Type:     cty.DynamicPseudoType,
					Optional: true,
				},
				"outputs": {
					Type:     cty.DynamicPseudoType,
					Computed: true,
				},
				"workspace": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
	}
}

func dataSourceRemoteStateRead(d *cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	newState := make(map[string]cty.Value)
	newState["backend"] = d.GetAttr("backend")

	backendType := d.GetAttr("backend").AsString()

	// Don't break people using the old _local syntax - but note warning above
	if backendType == "_local" {
		log.Println(`[INFO] Switching old (unsupported) backend "_local" to "local"`)
		backendType = "local"
	}

	// Create the client to access our remote state
	log.Printf("[DEBUG] Initializing remote state backend: %s", backendType)
	f := backendinit.Backend(backendType)
	if f == nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid backend configuration",
			fmt.Sprintf("Unknown backend type: %s", backendType),
			cty.Path(nil).GetAttr("backend"),
		))
		return cty.NilVal, diags
	}
	b := f()

	config := d.GetAttr("config")
	newState["config"] = config

	schema := b.ConfigSchema()
	// Try to coerce the provided value into the desired configuration type.
	configVal, err := schema.CoerceValue(config)
	if err != nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid backend configuration",
			fmt.Sprintf("The given configuration is not valid for backend %q: %s.", backendType,
				tfdiags.FormatError(err)),
			cty.Path(nil).GetAttr("config"),
		))
		return cty.NilVal, diags
	}

	validateDiags := b.ValidateConfig(configVal)
	diags = diags.Append(validateDiags)
	if validateDiags.HasErrors() {
		return cty.NilVal, diags
	}

	configureDiags := b.Configure(configVal)
	if configureDiags.HasErrors() {
		diags = diags.Append(configureDiags.Err())
		return cty.NilVal, diags
	}

	var name string

	if workspaceVal := d.GetAttr("workspace"); !workspaceVal.IsNull() {
		newState["workspace"] = workspaceVal
		ws := workspaceVal.AsString()
		if ws != backend.DefaultStateName {
			name = ws
		}
	}

	state, err := b.State(name)
	if err != nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Error loading state error",
			fmt.Sprintf("error loading the remote state: %s", err),
			cty.Path(nil).GetAttr("backend"),
		))
		return cty.NilVal, diags
	}

	if err := state.RefreshState(); err != nil {
		diags = diags.Append(err)
		return cty.NilVal, diags
	}

	outputs := make(map[string]cty.Value)

	if defaultsVal := d.GetAttr("defaults"); !defaultsVal.IsNull() {
		newState["defaults"] = defaultsVal
		it := defaultsVal.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			outputs[k.AsString()] = v
		}
	}

	remoteState := state.State()
	if remoteState.Empty() {
		log.Println("[DEBUG] empty remote state")
	} else {
		for k, os := range remoteState.RootModule().Outputs {
			outputs[k] = hcl2shim.HCL2ValueFromConfigValue(os.Value)
		}
	}

	newState["outputs"] = cty.ObjectVal(outputs)

	return cty.ObjectVal(newState), diags
}
