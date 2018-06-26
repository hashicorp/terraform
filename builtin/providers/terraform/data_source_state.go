package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/backend"
	backendinit "github.com/hashicorp/terraform/backend/init"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

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
		return cty.NilVal, diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			fmt.Sprintf("Unknown backend type: %s", backendType),
			"",
			cty.Path(nil).GetAttr("backend"),
		))
	}
	b := f()

	config := d.GetAttr("config")
	newState["config"] = config

	schema := b.ConfigSchema()
	// Try to coerce the provided value into the desired configuration type.
	configVal, err := schema.CoerceValue(config)
	if err != nil {
		return cty.NilVal, diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			fmt.Sprintf("invalid %s backend configuration: %s", backendType, tfdiags.FormatError(err)),
			"",
			cty.Path(nil).GetAttr("backend"),
		))
	}

	validateDiags := b.ValidateConfig(configVal)
	if validateDiags.HasErrors() {
		return cty.NilVal, diags.Append(validateDiags.Err())
	}
	configureDiags := b.Configure(configVal)
	if configureDiags.HasErrors() {
		return cty.NilVal, diags.Append(configureDiags.Err())
	}

	var name string
	if d.Type().HasAttribute("workspace") {
		newState["workspace"] = d.GetAttr("workspace")
		ws := d.GetAttr("workspace").AsString()
		if ws != backend.DefaultStateName {
			name = ws
		}
	}

	state, err := b.State(name)
	if err != nil {
		return cty.NilVal, diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			fmt.Sprintf("error loading the remote state: %s", err),
			"",
			cty.Path(nil).GetAttr("backend"),
		))
	}
	if err := state.RefreshState(); err != nil {
		return cty.NilVal, diags.Append(err)
	}

	outputs := make(map[string]cty.Value)

	if d.Type().HasAttribute("defaults") {
		defaults := d.GetAttr("defaults")
		newState["defaults"] = d.GetAttr("defaults")
		it := defaults.ElementIterator()
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
