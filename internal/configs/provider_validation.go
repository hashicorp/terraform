package configs

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// validateProviderConfigs walks the full configuration tree from the root
// module outward, static validation rules to the various combinations of
// provider configuration, required_providers values, and module call providers
// mappings.
//
// To retain compatibility with previous terraform versions, empty "proxy
// provider blocks" are still allowed within modules, though they will
// generate warnings when the configuration is loaded. The new validation
// however will generate an error if a suitable provider configuration is not
// passed in through the module call.
//
// The call argument is the ModuleCall for the provided Config cfg. The
// noProviderConfig argument is passed down the call stack, indicating that the
// module call, or a parent module call, has used a feature that precludes
// providers from being configured at all within the module.
func validateProviderConfigs(parentCall *ModuleCall, cfg *Config, noProviderConfig bool) (diags hcl.Diagnostics) {
	mod := cfg.Module

	for name, child := range cfg.Children {
		mc := mod.ModuleCalls[name]

		// if the module call has any of count, for_each or depends_on,
		// providers are prohibited from being configured in this module, or
		// any module beneath this module.
		nope := noProviderConfig || mc.Count != nil || mc.ForEach != nil || mc.DependsOn != nil
		diags = append(diags, validateProviderConfigs(mc, child, nope)...)
	}

	// the set of provider configuration names passed into the module, with the
	// source range of the provider assignment in the module call.
	passedIn := map[string]PassedProviderConfig{}

	// the set of empty configurations that could be proxy configurations, with
	// the source range of the empty configuration block.
	emptyConfigs := map[string]*hcl.Range{}

	// the set of provider with a defined configuration, with the source range
	// of the configuration block declaration.
	configured := map[string]*hcl.Range{}

	// the set of configuration_aliases defined in the required_providers
	// block, with the fully qualified provider type.
	configAliases := map[string]addrs.AbsProviderConfig{}

	// the set of provider names defined in the required_providers block, and
	// their provider types.
	localNames := map[string]addrs.AbsProviderConfig{}

	for _, pc := range mod.ProviderConfigs {
		name := providerName(pc.Name, pc.Alias)
		// Validate the config against an empty schema to see if it's empty.
		_, pcConfigDiags := pc.Config.Content(&hcl.BodySchema{})
		if pcConfigDiags.HasErrors() || pc.Version.Required != nil {
			configured[name] = &pc.DeclRange
		} else {
			emptyConfigs[name] = &pc.DeclRange
		}
	}

	if mod.ProviderRequirements != nil {
		for _, req := range mod.ProviderRequirements.RequiredProviders {
			addr := addrs.AbsProviderConfig{
				Module:   cfg.Path,
				Provider: req.Type,
			}
			localNames[req.Name] = addr
			for _, alias := range req.Aliases {
				addr := addrs.AbsProviderConfig{
					Module:   cfg.Path,
					Provider: req.Type,
					Alias:    alias.Alias,
				}
				configAliases[providerName(alias.LocalName, alias.Alias)] = addr
			}
		}
	}

	// collect providers passed from the parent
	if parentCall != nil {
		for _, passed := range parentCall.Providers {
			name := providerName(passed.InChild.Name, passed.InChild.Alias)
			passedIn[name] = passed
		}
	}

	parentModuleText := "the root module"
	moduleText := "the root module"
	if !cfg.Path.IsRoot() {
		moduleText = cfg.Path.String()
		if parent := cfg.Path.Parent(); !parent.IsRoot() {
			// module address are prefixed with `module.`
			parentModuleText = parent.String()
		}
	}

	// Verify that any module calls only refer to named providers, and that
	// those providers will have a configuration at runtime. This way we can
	// direct users where to add the missing configuration, because the runtime
	// error is only "missing provider X".
	for _, modCall := range mod.ModuleCalls {
		for _, passed := range modCall.Providers {
			// aliased providers are handled more strictly, and are never
			// inherited, so they are validated within modules further down.
			// Skip these checks to prevent redundant diagnostics.
			if passed.InParent.Alias != "" {
				continue
			}

			name := passed.InParent.String()
			_, confOK := configured[name]
			_, localOK := localNames[name]
			_, passedOK := passedIn[name]

			// This name was not declared somewhere within in the
			// configuration. We ignore empty configs, because they will
			// already produce a warning.
			if !(confOK || localOK) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  fmt.Sprintf("Provider %s is undefined", name),
					Detail: fmt.Sprintf("No provider named %s has been declared in %s.\n", name, moduleText) +
						fmt.Sprintf("If you wish to refer to the %s provider within the module, add a provider configuration, or an entry in the required_providers block.", name),
					Subject: &passed.InParent.NameRange,
				})
				continue
			}

			// Now we may have named this provider within the module, but
			// there won't be a configuration available at runtime if the
			// parent module did not pass one in.
			if !cfg.Path.IsRoot() && !(confOK || passedOK) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  fmt.Sprintf("No configuration passed in for provider %s in %s", name, cfg.Path),
					Detail: fmt.Sprintf("Provider %s is referenced within %s, but no configuration has been supplied.\n", name, moduleText) +
						fmt.Sprintf("Add a provider named %s to the providers map for %s in %s.", name, cfg.Path, parentModuleText),
					Subject: &passed.InParent.NameRange,
				})
			}
		}
	}

	if cfg.Path.IsRoot() {
		// nothing else to do in the root module
		return diags
	}

	// there cannot be any configurations if no provider config is allowed
	if len(configured) > 0 && noProviderConfig {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("Module %s contains provider configuration", cfg.Path),
			Detail:   "Providers cannot be configured within modules using count, for_each or depends_on.",
		})
	}

	// now check that the user is not attempting to override a config
	for name := range configured {
		if passed, ok := passedIn[name]; ok {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot override provider configuration",
				Detail:   fmt.Sprintf("Provider %s is configured within the module %s and cannot be overridden.", name, cfg.Path),
				Subject:  &passed.InChild.NameRange,
			})
		}
	}

	// A declared alias requires either a matching configuration within the
	// module, or one must be passed in.
	for name, providerAddr := range configAliases {
		_, confOk := configured[name]
		_, passedOk := passedIn[name]

		if confOk || passedOk {
			continue
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("No configuration for provider %s", name),
			Detail: fmt.Sprintf("Configuration required for %s.\n", providerAddr) +
				fmt.Sprintf("Add a provider named %s to the providers map for %s in %s.", name, cfg.Path, parentModuleText),
			Subject: &parentCall.DeclRange,
		})
	}

	// You cannot pass in a provider that cannot be used
	for name, passed := range passedIn {
		childTy := passed.InChild.providerType
		// get a default type if there was none set
		if childTy.IsZero() {
			// This means the child module is only using an inferred
			// provider type. We allow this but will generate a warning to
			// declare provider_requirements below.
			childTy = addrs.NewDefaultProvider(passed.InChild.Name)
		}

		providerAddr := addrs.AbsProviderConfig{
			Module:   cfg.Path,
			Provider: childTy,
			Alias:    passed.InChild.Alias,
		}

		localAddr, localName := localNames[name]
		if localName {
			providerAddr = localAddr
		}

		aliasAddr, configAlias := configAliases[name]
		if configAlias {
			providerAddr = aliasAddr
		}

		_, emptyConfig := emptyConfigs[name]

		if !(localName || configAlias || emptyConfig) {
			severity := hcl.DiagError

			// we still allow default configs, so switch to a warning if the incoming provider is a default
			if providerAddr.Provider.IsDefault() {
				severity = hcl.DiagWarning
			}

			diags = append(diags, &hcl.Diagnostic{
				Severity: severity,
				Summary:  fmt.Sprintf("Provider %s is undefined", name),
				Detail: fmt.Sprintf("Module %s does not declare a provider named %s.\n", cfg.Path, name) +
					fmt.Sprintf("If you wish to specify a provider configuration for the module, add an entry for %s in the required_providers block within the module.", name),
				Subject: &passed.InChild.NameRange,
			})
		}

		// The provider being passed in must also be of the correct type.
		pTy := passed.InParent.providerType
		if pTy.IsZero() {
			// While we would like to ensure required_providers exists here,
			// implied default configuration is still allowed.
			pTy = addrs.NewDefaultProvider(passed.InParent.Name)
		}

		// use the full address for a nice diagnostic output
		parentAddr := addrs.AbsProviderConfig{
			Module:   cfg.Parent.Path,
			Provider: pTy,
			Alias:    passed.InParent.Alias,
		}

		if cfg.Parent.Module.ProviderRequirements != nil {
			req, defined := cfg.Parent.Module.ProviderRequirements.RequiredProviders[name]
			if defined {
				parentAddr.Provider = req.Type
			}
		}

		if !providerAddr.Provider.Equals(parentAddr.Provider) {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Invalid type for provider %s", providerAddr),
				Detail: fmt.Sprintf("Cannot use configuration from %s for %s. ", parentAddr, providerAddr) +
					"The given provider configuration is for a different provider type.",
				Subject: &passed.InChild.NameRange,
			})
		}
	}

	// Empty configurations are no longer needed
	for name, src := range emptyConfigs {
		detail := fmt.Sprintf("Remove the %s provider block from %s.", name, cfg.Path)

		isAlias := strings.Contains(name, ".")
		_, isConfigAlias := configAliases[name]
		_, isLocalName := localNames[name]

		if isAlias && !isConfigAlias {
			localName := strings.Split(name, ".")[0]
			detail = fmt.Sprintf("Remove the %s provider block from %s. Add %s to the list of configuration_aliases for %s in required_providers to define the provider configuration name.", name, cfg.Path, name, localName)
		}

		if !isAlias && !isLocalName {
			// if there is no local name, add a note to include it in the
			// required_provider block
			detail += fmt.Sprintf("\nTo ensure the correct provider configuration is used, add %s to the required_providers configuration", name)
		}

		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Empty provider configuration blocks are not required",
			Detail:   detail,
			Subject:  src,
		})
	}

	return diags
}

func providerName(name, alias string) string {
	if alias != "" {
		name = name + "." + alias
	}
	return name
}
