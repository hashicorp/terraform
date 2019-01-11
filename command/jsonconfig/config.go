package jsonconfig

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"
)

// Config represents the complete configuration source
type config struct {
	ProviderConfigs map[string]providerConfig `json:"provider_config,omitempty"`
	RootModule      module                    `json:"root_module,omitempty"`
}

// ProviderConfig describes all of the provider configurations throughout the
// configuration tree, flattened into a single map for convenience since
// provider configurations are the one concept in Terraform that can span across
// module boundaries.
type providerConfig struct {
	Name          string                 `json:"name,omitempty"`
	Alias         string                 `json:"alias,omitempty"`
	ModuleAddress string                 `json:"module_address,omitempty"`
	Expressions   map[string]interface{} `json:"expressions,omitempty"`
}

type module struct {
	Outputs     map[string]configOutput `json:"outputs,omitempty"`
	Resources   []resource              `json:"resources,omitempty"`
	ModuleCalls []moduleCall            `json:"module_calls,omitempty"`
}

type moduleCall struct {
	ResolvedSource    string                 `json:"resolved_source,omitempty"`
	Expressions       map[string]interface{} `json:"expressions,omitempty"`
	CountExpression   expression             `json:"count_expression,omitempty"`
	ForEachExpression expression             `json:"for_each_expression,omitempty"`
	Module            module                 `json:"module,omitempty"`
}

// Resource is the representation of a resource in the config
type resource struct {
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
	Expressions map[string]interface{} `json:"expressions,omitempty"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion uint64 `json:"schema_version"`

	// CountExpression and ForEachExpression describe the expressions given for
	// the corresponding meta-arguments in the resource configuration block.
	// These are omitted if the corresponding argument isn't set.
	CountExpression   expression `json:"count_expression,omitempty"`
	ForEachExpression expression `json:"for_each_expression,omitempty"`
}

type configOutput struct {
	Sensitive  bool       `json:"sensitive,omitempty"`
	Expression expression `json:"expression,omitempty"`
}

type provisioner struct {
	Name        string                 `json:"name,omitempty"`
	Expressions map[string]interface{} `json:"expressions,omitempty"`
}

// Marshal returns the json encoding of terraform configuration.
func Marshal(c *configs.Config, schemas *terraform.Schemas) ([]byte, error) {
	var output config

	pcs := make(map[string]providerConfig)
	marshalProviderConfigs(c, schemas, pcs)
	output.ProviderConfigs = pcs

	rootModule, err := marshalModule(c, schemas)
	if err != nil {
		return nil, err
	}
	output.RootModule = rootModule

	ret, err := json.Marshal(output)
	return ret, err
}

func marshalProviderConfigs(
	c *configs.Config,
	schemas *terraform.Schemas,
	m map[string]providerConfig,
) {
	if c == nil {
		return
	}

	for _, pc := range c.Module.ProviderConfigs {
		schema := schemas.ProviderConfig(pc.Name)
		m[pc.Name] = providerConfig{
			Name:          pc.Name,
			Alias:         pc.Alias,
			ModuleAddress: c.Path.String(),
			Expressions:   marshalExpressions(pc.Config, schema),
		}
	}

	// Must also visit our child modules, recursively.
	for _, cc := range c.Children {
		marshalProviderConfigs(cc, schemas, m)
	}
}

func marshalModule(c *configs.Config, schemas *terraform.Schemas) (module, error) {
	var module module
	var rs []resource

	managedResources, err := marshalResources(c.Module.ManagedResources, schemas)
	if err != nil {
		return module, err
	}
	dataResources, err := marshalResources(c.Module.DataResources, schemas)
	if err != nil {
		return module, err
	}

	rs = append(managedResources, dataResources...)
	module.Resources = rs

	outputs := make(map[string]configOutput)
	for _, v := range c.Module.Outputs {
		outputs[v.Name] = configOutput{
			Sensitive:  v.Sensitive,
			Expression: marshalExpression(v.Expr),
		}
	}
	module.Outputs = outputs
	module.ModuleCalls = marshalModuleCalls(c, schemas)
	return module, nil
}

func marshalModuleCalls(c *configs.Config, schemas *terraform.Schemas) []moduleCall {
	var ret []moduleCall
	for _, v := range c.Module.ModuleCalls {
		mc := moduleCall{
			ResolvedSource: v.SourceAddr,
		}
		cExp := marshalExpression(v.Count)
		if !cExp.Empty() {
			mc.CountExpression = cExp
		} else {
			fExp := marshalExpression(v.ForEach)
			if !fExp.Empty() {
				mc.ForEachExpression = fExp
			}
		}

		schema := &configschema.Block{}
		schema.Attributes = make(map[string]*configschema.Attribute)
		for _, variable := range c.Module.Variables {
			schema.Attributes[variable.Name] = &configschema.Attribute{
				Required: variable.Default == cty.NilVal,
			}
		}
		mc.Expressions = marshalExpressions(v.Config, schema)

		for _, cc := range c.Children {
			childModule, _ := marshalModule(cc, schemas)
			mc.Module = childModule
		}
		ret = append(ret, mc)

	}

	return ret

}

func marshalResources(resources map[string]*configs.Resource, schemas *terraform.Schemas) ([]resource, error) {
	var rs []resource
	for _, v := range resources {
		r := resource{
			Address:           v.Addr().String(),
			Type:              v.Type,
			Name:              v.Name,
			ProviderConfigKey: v.ProviderConfigAddr().String(),
		}

		switch v.Mode {
		case addrs.ManagedResourceMode:
			r.Mode = "managed"
		case addrs.DataResourceMode:
			r.Mode = "data"
		default:
			return rs, fmt.Errorf("resource %s has an unsupported mode %s", r.Address, v.Mode.String())
		}

		cExp := marshalExpression(v.Count)
		if !cExp.Empty() {
			r.CountExpression = cExp
		} else {
			fExp := marshalExpression(v.ForEach)
			if !fExp.Empty() {
				r.ForEachExpression = fExp
			}
		}

		schema, schemaVersion := schemas.ResourceTypeConfig(v.ProviderConfigAddr().String(), v.Mode, v.Type)
		r.SchemaVersion = schemaVersion

		r.Expressions = marshalExpressions(v.Config, schema)

		// Managed is populated only for Mode = addrs.ManagedResourceMode
		if v.Managed != nil && len(v.Managed.Provisioners) > 0 {
			var provisioners []provisioner
			for _, p := range v.Managed.Provisioners {
				schema := schemas.ProvisionerConfig(p.Type)
				prov := provisioner{
					Name:        p.Type,
					Expressions: marshalExpressions(p.Config, schema),
				}
				provisioners = append(provisioners, prov)
			}
			r.Provisioners = provisioners
		}

		rs = append(rs, r)
	}
	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Address < rs[j].Address
	})
	return rs, nil
}
