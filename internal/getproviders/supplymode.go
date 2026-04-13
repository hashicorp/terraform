// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package getproviders

// ProviderSupplyMode describes how a provider is being supplied to Terraform.
type ProviderSupplyMode string

const (
	// Unset value.
	Unset ProviderSupplyMode = ""

	// The provider is built into the Terraform binary.
	BuiltIn ProviderSupplyMode = "built_in"

	// The provider is unmanaged by Terraform, supplied via TF_REATTACH_PROVIDERS by the user/environment.
	Reattached ProviderSupplyMode = "reattached"

	// The provider is overridden by a development configuration.
	DevOverride ProviderSupplyMode = "dev_override"

	// The provider is managed by Terraform. This is the "normal" and most common case.
	ManagedByTerraform ProviderSupplyMode = "managed_by_terraform"
)

// NotManagedByTerraform returns true if the provider supply mode is any of the modes that aren't managed by Terraform (built-in, reattached, dev override).
// The term "unmanaged" in the past has been used to refer to only reattached providers, however reattached, dev_override and built-in providers are all
// providers that Terraform will not record in the lock file and manage version data for.
//
// This method is intended to be used when that distinction is important.
func (m ProviderSupplyMode) NotManagedByTerraform() bool {
	return m != ManagedByTerraform
}

func DetermineProviderSupplyMode(isDevOverride bool, isReattached bool, isBuiltin bool) ProviderSupplyMode {
	switch {
	case isBuiltin:
		return BuiltIn
	case isReattached:
		return Reattached
	case isDevOverride:
		return DevOverride
	default:
		// assume provider is managed if nothing indicates otherwise.
		return ManagedByTerraform
	}
}
