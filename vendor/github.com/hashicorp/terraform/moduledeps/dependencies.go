package moduledeps

import (
	"github.com/hashicorp/terraform/plugin/discovery"
)

// Providers describes a set of provider dependencies for a given module.
//
// Each named provider instance can have one version constraint.
type Providers map[ProviderInstance]ProviderDependency

// ProviderDependency describes the dependency for a particular provider
// instance, including both the set of allowed versions and the reason for
// the dependency.
type ProviderDependency struct {
	Constraints discovery.Constraints
	Reason      ProviderDependencyReason
}

// ProviderDependencyReason is an enumeration of reasons why a dependency might be
// present.
type ProviderDependencyReason int

const (
	// ProviderDependencyExplicit means that there is an explicit "provider"
	// block in the configuration for this module.
	ProviderDependencyExplicit ProviderDependencyReason = iota

	// ProviderDependencyImplicit means that there is no explicit "provider"
	// block but there is at least one resource that uses this provider.
	ProviderDependencyImplicit

	// ProviderDependencyInherited is a special case of
	// ProviderDependencyImplicit where a parent module has defined a
	// configuration for the provider that has been inherited by at least one
	// resource in this module.
	ProviderDependencyInherited

	// ProviderDependencyFromState means that this provider is not currently
	// referenced by configuration at all, but some existing instances in
	// the state still depend on it.
	ProviderDependencyFromState
)
