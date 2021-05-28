package jsonconfig

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terraform"
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
	DependsOn         []string               `json:"depends_on,omitempty"`
}

// variables is the JSON representation of the variables provided to the current
// plan.
type variables map[string]*variable

type variable struct {
	Default     json.RawMessage `json:"default,omitempty"`
	Description string          `json:"description,omitempty"`
	Sensitive   bool            `json:"sensitive,omitempty"`
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
	//
	// NOTE: If a given resource is in a ModuleCall, and the provider was
	// configured outside of the module (in a higher level configuration file),
	// the ProviderConfigKey will not match a key in the ProviderConfigs map.
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
	CountExpression   *expression `json:"count_expression,omitempty"`
	ForEachExpression *expression `json:"for_each_expression,omitempty"`

	DependsOn []string `json:"depends_on,omitempty"`
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

	// We want to determine only the provider requirements from this module,
	// ignoring any descendants.  Disregard any diagnostics when determining
	// requirements because we want this marshalling to succeed even if there
	// are invalid constraints.
	reqs, _ := c.ProviderRequirementsShallow()

	// Add an entry for each provider configuration block in the module.
	for k, pc := range c.Module.ProviderConfigs {
		providerFqn := c.ProviderForConfigAddr(addrs.LocalProviderConfig{LocalName: pc.Name})
		schema := schemas.ProviderConfig(providerFqn)

		p := providerConfig{
			Name:          pc.Name,
			Alias:         pc.Alias,
			ModuleAddress: c.Path.String(),
			Expressions:   marshalExpressions(pc.Config, schema),
		}

		// Store the fully resolved provider version constraint, rather than
		// using the version argument in the configuration block. This is both
		// future proof (for when we finish the deprecation of the provider config
		// version argument) and more accurate (as it reflects the full set of
		// constraints, in case there are multiple).
		if vc, ok := reqs[providerFqn]; ok {
			p.VersionConstraint = getproviders.VersionConstraintsString(vc)
		}

		key := opaqueProviderKey(k, c.Path.String())

		m[key] = p
	}

	// Ensure that any required providers with no associated configuration
	// block are included in the set.
	for k, pr := range c.Module.ProviderRequirements.RequiredProviders {
		// If there exists a value for this provider, we have nothing to add
		// to it, so skip.
		key := opaqueProviderKey(k, c.Path.String())
		if _, exists := m[key]; exists {
			continue
		}

		// Given no provider configuration block exists, the only fields we can
		// fill here are the local name, module address, and version
		// constraints.
		p := providerConfig{
			Name:          pr.Name,
			ModuleAddress: c.Path.String(),
		}

		if vc, ok := reqs[pr.Type]; ok {
			p.VersionConstraint = getproviders.VersionConstraintsString(vc)
		}

		m[key] = p
	}

	// Must also visit our child modules, recursively.
	for _, cc := range c.Children {
		marshalProviderConfigs(cc, schemas, m)
	}
}

func marshalModule(c *configs.Config, schemas *terraform.Schemas, addr string) (module, error) {
	var module module
	var rs []resource

	managedResources, err := marshalResources(c.Module.ManagedResources, schemas, addr)
	if err != nil {
		return module, err
	}
	dataResources, err := marshalResources(c.Module.DataResources, schemas, addr)
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
				Sensitive:   v.Sensitive,
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
		// We're intentionally echoing back exactly what the user entered
		// here, rather than the normalized version in SourceAddr, because
		// historically we only _had_ the raw address and thus it would be
		// a (admittedly minor) breaking change to start normalizing them
		// now, in case consumers of this data are expecting a particular
		// non-normalized syntax.
		Source:            mc.SourceAddrRaw,
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

	if len(mc.DependsOn) > 0 {
		dependencies := make([]string, len(mc.DependsOn))
		for i, d := range mc.DependsOn {
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

	return ret
}

func marshalResources(resources map[string]*configs.Resource, schemas *terraform.Schemas, moduleAddr string) ([]resource, error) {
	var rs []resource
	for _, v := range resources {
		r := resource{
			Address:           v.Addr().String(),
			Type:              v.Type,
			Name:              v.Name,
			ProviderConfigKey: opaqueProviderKey(v.ProviderConfigAddr().StringCompact(), moduleAddr),
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
			r.CountExpression = &cExp
		} else {
			fExp := marshalExpression(v.ForEach)
			if !fExp.Empty() {
				r.ForEachExpression = &fExp
			}
		}

		schema, schemaVer := schemas.ResourceTypeConfig(
			v.Provider,
			v.Mode,
			v.Type,
		)
		if schema == nil {
			return nil, fmt.Errorf("no schema found for %s (in provider %s)", v.Addr().String(), v.Provider)
		}
		r.SchemaVersion = schemaVer

		r.Expressions = marshalExpressions(v.Config, schema)

		// Managed is populated only for Mode = addrs.ManagedResourceMode
		if v.Managed != nil && len(v.Managed.Provisioners) > 0 {
			var provisioners []provisioner
			for _, p := range v.Managed.Provisioners {
				schema := schemas.ProvisionerConfig(p.Type)
				prov := provisioner{
					Type:        p.Type,
					Expressions: marshalExpressions(p.Config, schema),
				}
				provisioners = append(provisioners, prov)
			}
			r.Provisioners = provisioners
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
			r.DependsOn = dependencies
		}

		rs = append(rs, r)
	}
	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Address < rs[j].Address
	})
	return rs, nil
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
