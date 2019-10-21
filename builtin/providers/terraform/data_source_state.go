package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"

	backendInit "github.com/hashicorp/terraform/backend/init"
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

func dataSourceRemoteStateValidate(cfg cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Getting the backend implicitly validates the configuration for it,
	// but we can only do that if it's all known already.
	if cfg.GetAttr("config").IsWhollyKnown() && cfg.GetAttr("backend").IsKnown() {
		_, moreDiags := getBackend(cfg)
		diags = diags.Append(moreDiags)
	} else {
		// Otherwise we'll just type-check the config object itself.
		configTy := cfg.GetAttr("config").Type()
		if configTy != cty.DynamicPseudoType && !(configTy.IsObjectType() || configTy.IsMapType()) {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid backend configuration",
				"The configuration must be an object value.",
				cty.GetAttrPath("config"),
			))
		}
	}

	{
		defaultsTy := cfg.GetAttr("defaults").Type()
		if defaultsTy != cty.DynamicPseudoType && !(defaultsTy.IsObjectType() || defaultsTy.IsMapType()) {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid default values",
				"Defaults must be given in an object value.",
				cty.GetAttrPath("defaults"),
			))
		}
	}

	return diags
}

func dataSourceRemoteStateRead(d cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	b, moreDiags := getBackend(d)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return cty.NilVal, diags
	}

	newState := make(map[string]cty.Value)
	newState["backend"] = d.GetAttr("backend")
	newState["config"] = d.GetAttr("config")

	workspaceName := backend.DefaultStateName

	if workspaceVal := d.GetAttr("workspace"); !workspaceVal.IsNull() {
		newState["workspace"] = workspaceVal
		workspaceName = workspaceVal.AsString()
	}

	newState["workspace"] = cty.StringVal(workspaceName)

	state, err := b.StateMgr(workspaceName)
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
	} else {
		newState["defaults"] = cty.NullVal(cty.DynamicPseudoType)
	}

	remoteState := state.State()
	if remoteState == nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Unable to find remote state",
			"No stored state was found for the given workspace in the given backend.",
			cty.Path(nil).GetAttr("workspace"),
		))
		newState["outputs"] = cty.EmptyObjectVal
		return cty.ObjectVal(newState), diags
	}
	mod := remoteState.RootModule()
	if mod != nil { // should always have a root module in any valid state
		for k, os := range mod.OutputValues {
			outputs[k] = os.Value
		}
	}

	newState["outputs"] = cty.ObjectVal(outputs)

	return cty.ObjectVal(newState), diags
}

func getBackend(cfg cty.Value) (backend.Backend, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	backendType := cfg.GetAttr("backend").AsString()

	// Don't break people using the old _local syntax - but note warning above
	if backendType == "_local" {
		log.Println(`[INFO] Switching old (unsupported) backend "_local" to "local"`)
		backendType = "local"
	}

	// Create the client to access our remote state
	log.Printf("[DEBUG] Initializing remote state backend: %s", backendType)
	f := backendInit.Backend(backendType)
	if f == nil {
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Invalid backend configuration",
			fmt.Sprintf("There is no backend type named %q.", backendType),
			cty.Path(nil).GetAttr("backend"),
		))
		return nil, diags
	}
	b := f()

	config := cfg.GetAttr("config")
	if config.IsNull() {
		// We'll treat this as an empty configuration and see if the backend's
		// schema and validation code will accept it.
		config = cty.EmptyObjectVal
	}

	if config.Type().IsMapType() { // The code below expects an object type, so we'll convert
		config = cty.ObjectVal(config.AsValueMap())
	}

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
		return nil, diags
	}

	newVal, validateDiags := b.PrepareConfig(configVal)
	diags = diags.Append(validateDiags)
	if validateDiags.HasErrors() {
		return nil, diags
	}
	configVal = newVal

	configureDiags := b.Configure(configVal)
	if configureDiags.HasErrors() {
		diags = diags.Append(configureDiags.Err())
		return nil, diags
	}

	return b, diags
}
