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
