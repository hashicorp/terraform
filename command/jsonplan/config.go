package jsonplan

import (
	"encoding/json"

	"github.com/hashicorp/terraform/configs/configload"

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
	Outputs     map[string]output `json:"outputs,omitempty"`
	Resources   []resource        `json:"resources,omitempty"`
	ModuleCalls []moduleCall      `json:"module_calls,omitempty"`
}

type configOutput struct {
	Sensitive  bool       `json:"sensitive,omitempty"`
	Expression expression `json:"expression,omitempty"`
}

func (p *plan) marshalConfig(snap *configload.Snapshot) error {
	configLoader := configload.NewLoaderFromSnapshot(snap)
	c, diags := configLoader.LoadConfig(snap.Modules[""].Dir)
	if diags.HasErrors() {
		return diags
	}

	var rs []resource
	for _, v := range c.Module.ManagedResources {
		r := resource{
			Address: v.Addr().String(),
			Mode:    v.Mode.String(),
			Type:    v.Type,
			Name:    v.Name,
			// Index: // Does not apply for config?
			ProviderName: v.ProviderConfigAddr().String(),
			// SchemaVersion:
			// Values:
		}
		rs = append(rs, r)
	}
	for _, v := range c.Module.DataResources {
		r := resource{
			Address: v.Addr().String(),
			Mode:    v.Mode.String(),
			Type:    v.Type,
			Name:    v.Name,
			// Index: // Does not apply for config?
			ProviderName: v.ProviderConfigRef.Name,
			// SchemaVersion:
			// Values:
		}
		rs = append(rs, r)
	}
	p.Config.RootModule.Resources = rs

	outputs := make(map[string]output)
	for _, v := range c.Module.Outputs {
		// Is there a context I should be using here?
		val, _ := v.Expr.Value(nil)
		valJSON, _ := ctyjson.Marshal(val, val.Type())
		outputs[v.Name] = output{
			Sensitive: v.Sensitive,
			Value:     json.RawMessage(valJSON),
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
