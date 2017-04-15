package config

import "github.com/blang/semver"

// ProviderVersionConstraint presents a constraint for a particular
// provider, identified by its full name.
type ProviderVersionConstraint struct {
	Constraint   string
	ProviderType string
}

// ProviderVersionConstraints is a map from provider full name to its associated
// ProviderVersionConstraint, as produced by Config.RequiredProviders.
type ProviderVersionConstraints map[string]ProviderVersionConstraint

// RequiredProviders returns the ProviderVersionConstraints for this
// module.
//
// This includes both providers that are explicitly requested by provider
// blocks and those that are used implicitly by instantiating one of their
// resource types. In the latter case, the returned semver Range will
// accept any version of the provider.
func (c *Config) RequiredProviders() ProviderVersionConstraints {
	ret := make(ProviderVersionConstraints, len(c.ProviderConfigs))

	configs := c.ProviderConfigsByFullName()

	// In order to find the *implied* dependencies (those without explicit
	// "provider" blocks) we need to walk over all of the resources and
	// cross-reference with the provider configs.
	for _, rc := range c.Resources {
		providerName := rc.ProviderFullName()
		var providerType string

		// Default to (effectively) no constraint whatsoever, but we might
		// override if there's an explicit constraint in config.
		constraint := ">=0.0.0"

		config, ok := configs[providerName]
		if ok {
			if config.Version != "" {
				constraint = config.Version
			}
			providerType = config.Name
		} else {
			providerType = providerName
		}

		ret[providerName] = ProviderVersionConstraint{
			ProviderType: providerType,
			Constraint:   constraint,
		}
	}

	return ret
}

// RequiredRanges returns a semver.Range for each distinct provider type in
// the constraint map. If the same provider type appears more than once
// (e.g. because aliases are in use) then their respective constraints are
// combined such that they must *all* apply.
//
// The result of this method can be passed to the
// PluginMetaSet.ConstrainVersions method within the plugin/discovery
// package in order to filter down the available plugins to those which
// satisfy the given constraints.
//
// This function will panic if any of the constraints within cannot be
// parsed as semver ranges. This is guaranteed to never happen for a
// constraint set that was built from a configuration that passed validation.
func (cons ProviderVersionConstraints) RequiredRanges() map[string]semver.Range {
	ret := make(map[string]semver.Range, len(cons))

	for _, con := range cons {
		spec := semver.MustParseRange(con.Constraint)
		if existing, exists := ret[con.ProviderType]; exists {
			ret[con.ProviderType] = existing.AND(spec)
		} else {
			ret[con.ProviderType] = spec
		}
	}

	return ret
}

// ProviderConfigsByFullName returns a map from provider full names (as
// returned by ProviderConfig.FullName()) to the corresponding provider
// configs.
//
// This function returns no new information than what's already in
// c.ProviderConfigs, but returns it in a more convenient shape. If there
// is more than one provider config with the same full name then the result
// is undefined, but that is guaranteed not to happen for any config that
// has passed validation.
func (c *Config) ProviderConfigsByFullName() map[string]*ProviderConfig {
	ret := make(map[string]*ProviderConfig, len(c.ProviderConfigs))

	for _, pc := range c.ProviderConfigs {
		ret[pc.FullName()] = pc
	}

	return ret
}
