package jsonplan

import (
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/lang"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// Config represents the complete configuration source
type config struct {
	ProviderConfigs []providerConfig `json:"provider_config,omitempty"`
	RootModule      configRootModule `json:"root_module,omitempty"`
}

// ProviderConfig describes all of the provider configurations throughout the
// configuration tree, flattened into a single map for convenience since
// provider configurations are the one concept in Terraform that can span across
// module boundaries.
type providerConfig struct {
	Name          string      `json:"name,omitempty"`
	Alias         string      `json:"alias,omitempty"`
	ModuleAddress string      `json:"module_address,omitempty"`
	Expressions   expressions `json:"expressions,omitempty"`
}

type configRootModule struct {
	Outputs     map[string]configOutput `json:"outputs,omitempty"`
	Resources   []configResource        `json:"resources,omitempty"`
	ModuleCalls []moduleCall            `json:"module_calls,omitempty"`
}

// Resource is the representation of a resource in the config
type configResource struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`

	// ProviderConfigKey is the key into "provider_configs" (shown above) for
	// the provider configuration that this resource is associated with.
	ProviderConfigKey string `json:"provider_config_key,omitempty"`

	// Provisioners is an optional field which describes any provisioners.
	// Connection info will not be included here.
	Provisioners []provisioner `json:"provisioners,omitempty"`

	// Expressions" describes the resource-type-specific  content of the
	// configuration block.
	Expressions expressions `json:"expressions,omitempty"`
	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion int `json:"schema_version,omitempty"`

	// CountExpression and ForEachExpression describe the expressions given for
	// the corresponding meta-arguments in the resource configuration block.
	// These are omitted if the corresponding argument isn't set.
	CountExpression   expression `json:"count_expression"`
	ForEachExpression expression `json:"for_each_expression,omitempty"`
}

type configOutput struct {
	Sensitive  bool       `json:"sensitive,omitempty"`
	Expression expression `json:"expression,omitempty"`
}

type provisioner struct {
	Name        string      `json:"name,omitempty"`
	Expressions expressions `json:"expressions,omitempty"`
}

func (p *plan) marshalConfig(snap *configload.Snapshot) error {
	configLoader := configload.NewLoaderFromSnapshot(snap)
	c, diags := configLoader.LoadConfig(snap.Modules[""].Dir)
	if diags.HasErrors() {
		return diags
	}

	var rs []configResource
	for _, v := range c.Module.ManagedResources {
		r := configResource{
			Address:           v.Addr().String(),
			Mode:              v.Mode.String(),
			Type:              v.Type,
			Name:              v.Name,
			ProviderConfigKey: v.ProviderConfigAddr().String(),
			// SchemaVersion:
			// Expressions:
		}
		rs = append(rs, r)
	}
	for _, v := range c.Module.DataResources {
		r := configResource{
			Address:           v.Addr().String(),
			Mode:              v.Mode.String(),
			Type:              v.Type,
			Name:              v.Name,
			ProviderConfigKey: v.ProviderConfigRef.Name,
			// SchemaVersion:
			// Expressions:
		}
		rs = append(rs, r)
	}
	p.Config.RootModule.Resources = rs

	outputs := make(map[string]configOutput)
	for _, v := range c.Module.Outputs {
		// Is there a context I should be using here?
		val, _ := v.Expr.Value(nil)
		var ex expression
		if val != cty.NilVal {
			valJSON, _ := ctyjson.Marshal(val, val.Type())
			ex.ConstantValue = valJSON
		}

		vars, _ := lang.ReferencesInExpr(v.Expr)
		var varString []string
		for _, v := range vars {
			varString = append(varString, v.Subject.String())
		}
		ex.References = varString

		outputs[v.Name] = configOutput{
			Sensitive:  v.Sensitive,
			Expression: ex,
		}
	}
	p.Config.RootModule.Outputs = outputs

	// this is not accurate provider marshalling, just a placeholder
	var pcs []providerConfig
	providers := c.ProviderTypes()
	for p := range providers {
		pc := providerConfig{
			Name: providers[p],
		}
		pcs = append(pcs, pc)
	}

	p.Config.ProviderConfigs = pcs

	return nil
}
