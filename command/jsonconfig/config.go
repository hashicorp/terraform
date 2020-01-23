package jsonconfig

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
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
	Name              string                 `json:"name,omitempty"`
	Alias             string                 `json:"alias,omitempty"`
	VersionConstraint string                 `json:"version_constraint,omitempty"`
	ModuleAddress     string                 `json:"module_address,omitempty"`
	Expressions       map[string]interface{} `json:"expressions,omitempty"`
}

type module struct {
	Outputs map[string]output `json:"outputs,omitempty"`
	// Resources are sorted in a user-friendly order that is undefined at this
	// time, but consistent.
	Resources   []resource            `json:"resources,omitempty"`
	ModuleCalls map[string]moduleCall `json:"module_calls,omitempty"`
	Variables   variables             `json:"variables,omitempty"`
}

type moduleCall struct {
	Source            string                 `json:"source,omitempty"`
	Expressions       map[string]interface{} `json:"expressions,omitempty"`
	CountExpression   *expression            `json:"count_expression,omitempty"`
	ForEachExpression *expression            `json:"for_each_expression,omitempty"`
	Module            module                 `json:"module,omitempty"`
	VersionConstraint string                 `json:"version_constraint,omitempty"`
}

// variables is the JSON representation of the variables provided to the current
// plan.
type variables map[string]*variable

type variable struct {
	Default     json.RawMessage `json:"default,omitempty"`
	Description string          `json:"description,omitempty"`
}

type output struct {
	Sensitive   bool       `json:"sensitive,omitempty"`
	Expression  expression `json:"expression,omitempty"`
	DependsOn   []string   `json:"depends_on,omitempty"`
	Description string     `json:"description,omitempty"`
}

type provisioner struct {
	Type        string                 `json:"type,omitempty"`
	Expressions map[string]interface{} `json:"expressions,omitempty"`
}

// Marshal returns the json encoding of terraform configuration.
func Marshal(c *configs.Config, schemas *terraform.Schemas) ([]byte, error) {
	var output config

	pcs := make(map[string]providerConfig)
	marshalProviderConfigs(c, schemas, pcs)
	output.ProviderConfigs = pcs

	rootModule, err := marshalModule(c, schemas, "")
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

	for k, pc := range c.Module.ProviderConfigs {

		var fqn addrs.Provider
		if provider, exists := c.Module.ProviderRequirements[pc.Name]; exists {
			fqn = provider.Type
		} else {
			fqn = addrs.NewLegacyProvider(pc.Name)
		}
		schema := schemas.ProviderConfig(fqn.String())
		p := providerConfig{
			Name:              pc.Name,
			Alias:             pc.Alias,
			ModuleAddress:     c.Path.String(),
			Expressions:       marshalExpressions(pc.Config, schema),
			VersionConstraint: pc.Version.Required.String(),
		}
		absPC := opaqueProviderKey(k, c.Path.String())

		m[absPC] = p
	}

	// Must also visit our child modules, recursively.
	for _, cc := range c.Children {
		marshalProviderConfigs(cc, schemas, m)
	}
}

func marshalModule(c *configs.Config, schemas *terraform.Schemas, addr string) (module, error) {
	var module module
	var rs []resource

	managedResources, dataResources, err := marshalResources(c, schemas, addr)
	if err != nil {
		return module, err
	}

	rs = append(managedResources, dataResources...)
	module.Resources = rs

	outputs := make(map[string]output)
	for _, v := range c.Module.Outputs {
		o := output{
			Sensitive:  v.Sensitive,
			Expression: marshalExpression(v.Expr),
		}
		if v.Description != "" {
			o.Description = v.Description
		}
		if len(v.DependsOn) > 0 {
			dependencies := make([]string, len(v.DependsOn))
			for i, d := range v.DependsOn {
				ref, diags := addrs.ParseRef(d)
				// we should not get an error here, because `terraform validate`
				// would have complained well before this point, but if we do we'll
				// silenty skip it.
				if !diags.HasErrors() {
					dependencies[i] = ref.Subject.String()
				}
			}
			o.DependsOn = dependencies
		}

		outputs[v.Name] = o
	}
	module.Outputs = outputs

	module.ModuleCalls = marshalModuleCalls(c, schemas)

	if len(c.Module.Variables) > 0 {
		vars := make(variables, len(c.Module.Variables))
		for k, v := range c.Module.Variables {
			var defaultValJSON []byte
			if v.Default == cty.NilVal {
				defaultValJSON = nil
			} else {
				defaultValJSON, err = ctyjson.Marshal(v.Default, v.Default.Type())
				if err != nil {
					return module, err
				}
			}
			vars[k] = &variable{
				Default:     defaultValJSON,
				Description: v.Description,
			}
		}
		module.Variables = vars
	}

	return module, nil
}

func marshalModuleCalls(c *configs.Config, schemas *terraform.Schemas) map[string]moduleCall {
	ret := make(map[string]moduleCall)

	for name, mc := range c.Module.ModuleCalls {
		mcConfig := c.Children[name]
		ret[name] = marshalModuleCall(mcConfig, mc, schemas)
	}

	return ret
}

func marshalModuleCall(c *configs.Config, mc *configs.ModuleCall, schemas *terraform.Schemas) moduleCall {
	// It is possible to have a module call with a nil config.
	if c == nil {
		return moduleCall{}
	}

	ret := moduleCall{
		Source:            mc.SourceAddr,
		VersionConstraint: mc.Version.Required.String(),
	}
	cExp := marshalExpression(mc.Count)
	if !cExp.Empty() {
		ret.CountExpression = &cExp
	} else {
		fExp := marshalExpression(mc.ForEach)
		if !fExp.Empty() {
			ret.ForEachExpression = &fExp
		}
	}

	schema := &configschema.Block{}
	schema.Attributes = make(map[string]*configschema.Attribute)
	for _, variable := range c.Module.Variables {
		schema.Attributes[variable.Name] = &configschema.Attribute{
			Required: variable.Default == cty.NilVal,
		}
	}

	ret.Expressions = marshalExpressions(mc.Config, schema)
	module, _ := marshalModule(c, schemas, mc.Name)
	ret.Module = module

	return ret
}

func marshalResources(c *configs.Config, schemas *terraform.Schemas, moduleAddr string) (managed resources, data resources, err error) {
	for _, v := range c.Module.ManagedResources {
		fqn := c.ProviderForLocalConfigAddr(v.ProviderConfigAddr())
		r, err := marshalResource(v, moduleAddr, fqn, schemas)
		if err != nil {
			return nil, nil, err
		}
		managed = append(managed, r)
	}
	for _, v := range c.Module.DataResources {
		fqn := c.ProviderForLocalConfigAddr(v.ProviderConfigAddr())
		r, err := marshalResource(v, moduleAddr, fqn, schemas)
		if err != nil {
			return nil, nil, err
		}
		data = append(data, r)
	}
	sort.Sort(managed)
	sort.Sort(data)

	return managed, data, nil
}

func marshalResource(r *configs.Resource, moduleAddr string, provider addrs.Provider, schemas *terraform.Schemas) (resource, error) {
	ret := resource{
		Address:           r.Addr().String(),
		Type:              r.Type,
		Name:              r.Name,
		ProviderConfigKey: opaqueProviderKey(r.ProviderConfigAddr().StringCompact(), moduleAddr),
	}

	switch r.Mode {
	case addrs.ManagedResourceMode:
		ret.Mode = "managed"
	case addrs.DataResourceMode:
		ret.Mode = "data"
	default:
		return ret, fmt.Errorf("resource %s has an unsupported mode %s", ret.Address, r.Mode.String())
	}

	cExp := marshalExpression(r.Count)
	if !cExp.Empty() {
		ret.CountExpression = &cExp
	} else {
		fExp := marshalExpression(r.ForEach)
		if !fExp.Empty() {
			ret.ForEachExpression = &fExp
		}
	}

	schema, schemaVer := schemas.ResourceTypeConfig(
		provider.String(),
		r.Mode,
		r.Type,
	)
	if schema == nil {
		return ret, fmt.Errorf("no schema found for %s", r.Addr().String())
	}
	ret.SchemaVersion = schemaVer

	ret.Expressions = marshalExpressions(r.Config, schema)

	// Managed is populated only for Mode = addrs.ManagedResourceMode
	if r.Managed != nil && len(r.Managed.Provisioners) > 0 {
		var provisioners []provisioner
		for _, p := range r.Managed.Provisioners {
			schema := schemas.ProvisionerConfig(p.Type)
			prov := provisioner{
				Type:        p.Type,
				Expressions: marshalExpressions(p.Config, schema),
			}
			provisioners = append(provisioners, prov)
		}
		ret.Provisioners = provisioners
	}

	if len(r.DependsOn) > 0 {
		dependencies := make([]string, len(r.DependsOn))
		for i, d := range r.DependsOn {
			ref, diags := addrs.ParseRef(d)
			// we should not get an error here, because `terraform validate`
			// would have complained well before this point, but if we do we'll
			// silenty skip it.
			if !diags.HasErrors() {
				dependencies[i] = ref.Subject.String()
			}
		}
		ret.DependsOn = dependencies
	}

	return ret, nil
}

// opaqueProviderKey generates a unique absProviderConfig-like string from the module
// address and provider
func opaqueProviderKey(provider string, addr string) (key string) {
	key = provider
	if addr != "" {
		key = fmt.Sprintf("%s:%s", addr, provider)
	}
	return key
}
