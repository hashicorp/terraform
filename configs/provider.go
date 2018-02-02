package configs

import (
	"github.com/hashicorp/hcl2/hcl"
)

// Provider represents a "provider" block in a module or file. A provider
// block is a provider configuration, and there can be zero or more
// configurations for each actual provider.
type Provider struct {
	Name       string
	Alias      string
	AliasRange hcl.Range

	Version VersionConstraint

	Config hcl.Body

	DeclRange hcl.Range
}

// ProviderRequirement represents a declaration of a dependency on a particular
// provider version without actually configuring that provider. This is used in
// child modules that expect a provider to be passed in from their parent.
type ProviderRequirement struct {
	Name        string
	Requirement VersionConstraint
}
