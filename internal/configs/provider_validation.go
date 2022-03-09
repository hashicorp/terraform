package configs

import (
	"fmt"
	"sort"
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
// noProviderConfigRange argument is passed down the call stack, indicating
// that the module call, or a parent module call, has used a feature (at the
// specified source location) that precludes providers from being configured at
// all within the module.
func validateProviderConfigs(parentCall *ModuleCall, cfg *Config, noProviderConfigRange *hcl.Range) (diags hcl.Diagnostics) {
	mod := cfg.Module

	for name, child := range cfg.Children {
		mc := mod.ModuleCalls[name]

		// if the module call has any of count, for_each or depends_on,
		// providers are prohibited from being configured in this module, or
		// any module beneath this module.
		// NOTE: If noProviderConfigRange was already set but we encounter
		// a nested conflicting argument then we'll overwrite the caller's
		// range, which allows us to report the problem as close to its
		// cause as possible.
		switch {
		case mc.Count != nil:
			noProviderConfigRange = mc.Count.Range().Ptr()
		case mc.ForEach != nil:
			noProviderConfigRange = mc.ForEach.Range().Ptr()
		case mc.DependsOn != nil:
			if len(mc.DependsOn) > 0 {
				noProviderConfigRange = mc.DependsOn[0].SourceRange().Ptr()
			} else {
				// Weird! We'll just use the call itself, then.
				noProviderConfigRange = mc.DeclRange.Ptr()
			}
		}
		diags = append(diags, validateProviderConfigs(mc, child, noProviderConfigRange)...)
	}

	// the set of provider configuration names passed into the module, with the
	// source range of the provider assignment in the module call.
	passedIn := map[string]PassedProviderConfig{}

	// the set of empty configurations that could be proxy configurations, with
	// the source range of the empty configuration block.
	emptyConfigs := map[string]hcl.Range{}

	// the set of provider with a defined configuration, with the source range
	// of the configuration block declaration.
	configured := map[string]hcl.Range{}

	// the set of configuration_aliases defined in the required_providers
	// block, with the fully qualified provider type.
	configAliases := map[string]addrs.AbsProviderConfig{}

	// the set of provider names defined in the required_providers block, and
	// their provider types.
	localNames := map[string]addrs.Provider{}

	for _, pc := range mod.ProviderConfigs {
		name := providerName(pc.Name, pc.Alias)
		// Validate the config against an empty schema to see if it's empty.
		_, pcConfigDiags := pc.Config.Content(&hcl.BodySchema{})
		if pcConfigDiags.HasErrors() || pc.Version.Required != nil {
			configured[name] = pc.DeclRange
		} else {
			emptyConfigs[name] = pc.DeclRange
		}
	}

	if mod.ProviderRequirements != nil {
		for _, req := range mod.ProviderRequirements.RequiredProviders {
			localNames[req.Name] = req.Type
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
				defAddr := addrs.NewDefaultProvider(name)
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Reference to undefined provider",
					Detail: fmt.Sprintf(
						"There is no explicit declaration for local provider name %q in %s, so Terraform is assuming you mean to pass a configuration for provider %q.\n\nTo clarify your intent and silence this warning, add to %s a required_providers entry named %q with source = %q, or a different source address if appropriate.",
						name, moduleText, defAddr.ForDisplay(),
						parentModuleText, name, defAddr.ForDisplay(),
					),
					Subject: &passed.InParent.NameRange,
				})
				continue
			}

			// Now we may have named this provider within the module, but
			// there won't be a configuration available at runtime if the
			// parent module did not pass one in.
			if !cfg.Path.IsRoot() && !(confOK || passedOK) {
				defAddr := addrs.NewDefaultProvider(name)
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Missing required provider configuration",
					Detail: fmt.Sprintf(
						"The configuration for %s expects to inherit a configuration for provider %s with local name %q, but %s doesn't pass a configuration under that name.\n\nTo satisfy this requirement, add an entry for %q to the \"providers\" argument in the module %q block.",
						moduleText, defAddr.ForDisplay(), name, parentModuleText,
						name, parentCall.Name,
					),
					Subject: parentCall.DeclRange.Ptr(),
				})
			}
		}
	}

	if cfg.Path.IsRoot() {
		// nothing else to do in the root module
		return diags
	}

	// there cannot be any configurations if no provider config is allowed
	if len(configured) > 0 && noProviderConfigRange != nil {
		// We report this from the perspective of the use of count, for_each,
		// or depends_on rather than from inside the module, because the
		// recipient of this message is more likely to be the author of the
		// calling module (trying to use an older module that hasn't been
		// updated yet) than of the called module.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Module is incompatible with count, for_each, and depends_on",
			Detail: fmt.Sprintf(
				"The module at %s is a legacy module which contains its own local provider configurations, and so calls to it may not use the count, for_each, or depends_on arguments.\n\nIf you also control the module %q, consider updating this module to instead expect provider configurations to be passed by its caller.",
				cfg.Path, cfg.SourceAddr,
			),
			Subject: noProviderConfigRange,
		})
	}

	// now check that the user is not attempting to override a config
	for name := range configured {
		if passed, ok := passedIn[name]; ok {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot override provider configuration",
				Detail: fmt.Sprintf(
					"The configuration of %s has its own local configuration for %s, and so it cannot accept an overridden configuration provided by %s.",
					moduleText, name, parentModuleText,
				),
				Subject: &passed.InChild.NameRange,
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
			Summary:  "Missing required provider configuration",
			Detail: fmt.Sprintf(
				"The child module requires an additional configuration for provider %s, with the local name %q.\n\nRefer to the module's documentation to understand the intended purpose of this additional provider configuration, and then add an entry for %s in the \"providers\" meta-argument in the module block to choose which provider configuration the module should use for that purpose.",
				providerAddr.Provider.ForDisplay(), name,
				name,
			),
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
			providerAddr.Provider = localAddr
		}

		aliasAddr, configAlias := configAliases[name]
		if configAlias {
			providerAddr = aliasAddr
		}

		_, emptyConfig := emptyConfigs[name]

		if !(localName || configAlias || emptyConfig) {

			// we still allow default configs, so switch to a warning if the incoming provider is a default
			if providerAddr.Provider.IsDefault() {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Reference to undefined provider",
					Detail: fmt.Sprintf(
						"There is no explicit declaration for local provider name %q in %s, so Terraform is assuming you mean to pass a configuration for %q.\n\nIf you also control the child module, add a required_providers entry named %q with the source address %q.",
						name, moduleText, providerAddr.Provider.ForDisplay(),
						name, providerAddr.Provider.ForDisplay(),
					),
					Subject: &passed.InChild.NameRange,
				})
			} else {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Reference to undefined provider",
					Detail: fmt.Sprintf(
						"The child module does not declare any provider requirement with the local name %q.\n\nIf you also control the child module, you can add a required_providers entry named %q with the source address %q to accept this provider configuration.",
						name, name, providerAddr.Provider.ForDisplay(),
					),
					Subject: &passed.InChild.NameRange,
				})
			}
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
			// If this module declares the same source address for a different
			// local name then we'll prefer to suggest changing to match
			// the child module's chosen name, assuming that it was the local
			// name that was wrong rather than the source address.
			var otherLocalName string
			for localName, sourceAddr := range localNames {
				if sourceAddr.Equals(parentAddr.Provider) {
					otherLocalName = localName
					break
				}
			}

			const errSummary = "Provider type mismatch"
			if otherLocalName != "" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errSummary,
					Detail: fmt.Sprintf(
						"The assigned configuration is for provider %q, but local name %q in %s represents %q.\n\nTo pass this configuration to the child module, use the local name %q instead.",
						parentAddr.Provider.ForDisplay(), passed.InChild.Name,
						parentModuleText, providerAddr.Provider.ForDisplay(),
						otherLocalName,
					),
					Subject: &passed.InChild.NameRange,
				})
			} else {
				// If there is no declared requirement for the provider the
				// caller is trying to pass under any name then we'll instead
				// report it as an unsuitable configuration to pass into the
				// child module's provider configuration slot.
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errSummary,
					Detail: fmt.Sprintf(
						"The local name %q in %s represents provider %q, but %q in %s represents %q.\n\nEach provider has its own distinct configuration schema and provider types, so this module's %q can be assigned only a configuration for %s, which is not required by %s.",
						passed.InParent, parentModuleText, parentAddr.Provider.ForDisplay(),
						passed.InChild, moduleText, providerAddr.Provider.ForDisplay(),
						passed.InChild, providerAddr.Provider.ForDisplay(),
						moduleText,
					),
					Subject: passed.InParent.NameRange.Ptr(),
				})
			}
		}
	}

	// Empty configurations are no longer needed. Since the replacement for
	// this calls for one entry per provider rather than one entry per
	// provider _configuration_, we'll first gather them up by provider
	// and then report a single warning for each, whereby we can show a direct
	// example of what the replacement should look like.
	type ProviderReqSuggestion struct {
		SourceAddr      addrs.Provider
		SourceRanges    []hcl.Range
		RequiredConfigs []string
		AliasCount      int
	}
	providerReqSuggestions := make(map[string]*ProviderReqSuggestion)
	for name, src := range emptyConfigs {
		providerLocalName := name
		if idx := strings.IndexByte(providerLocalName, '.'); idx >= 0 {
			providerLocalName = providerLocalName[:idx]
		}

		sourceAddr, ok := localNames[name]
		if !ok {
			sourceAddr = addrs.NewDefaultProvider(providerLocalName)
		}

		suggestion := providerReqSuggestions[providerLocalName]
		if suggestion == nil {
			providerReqSuggestions[providerLocalName] = &ProviderReqSuggestion{
				SourceAddr: sourceAddr,
			}
			suggestion = providerReqSuggestions[providerLocalName]
		}

		if providerLocalName != name {
			// It's an aliased provider config, then.
			suggestion.AliasCount++
		}

		suggestion.RequiredConfigs = append(suggestion.RequiredConfigs, name)
		suggestion.SourceRanges = append(suggestion.SourceRanges, src)
	}
	for name, suggestion := range providerReqSuggestions {
		var buf strings.Builder

		fmt.Fprintf(
			&buf,
			"Earlier versions of Terraform used empty provider blocks (\"proxy provider configurations\") for child modules to declare their need to be passed a provider configuration by their callers. That approach was ambiguous and is now deprecated.\n\nIf you control this module, you can migrate to the new declaration syntax by removing all of the empty provider %q blocks and then adding or updating an entry like the following to the required_providers block of %s:\n",
			name, moduleText,
		)
		fmt.Fprintf(&buf, "    %s = {\n", name)
		fmt.Fprintf(&buf, "      source = %q\n", suggestion.SourceAddr.ForDisplay())
		if suggestion.AliasCount > 0 {
			// A lexical sort is fine because all of these strings are
			// guaranteed to start with the same provider local name, and
			// so we're only really sorting by the alias part.
			sort.Strings(suggestion.RequiredConfigs)
			fmt.Fprintln(&buf, "      configuration_aliases = [")
			for _, addrStr := range suggestion.RequiredConfigs {
				fmt.Fprintf(&buf, "        %s,\n", addrStr)
			}
			fmt.Fprintln(&buf, "      ]")

		}
		fmt.Fprint(&buf, "    }")

		// We're arbitrarily going to just take the one source range that
		// sorts earliest here. Multiple should be rare, so this is only to
		// ensure that we produce a deterministic result in the edge case.
		sort.Slice(suggestion.SourceRanges, func(i, j int) bool {
			return suggestion.SourceRanges[i].String() < suggestion.SourceRanges[j].String()
		})
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Redundant empty provider block",
			Detail:   buf.String(),
			Subject:  suggestion.SourceRanges[0].Ptr(),
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
