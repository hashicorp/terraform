// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package supplymode

// ProviderSupplyMode describes how a provider is being supplied to Terraform.
type ProviderSupplyMode string

const (
	// Unset value.
	ProviderSupplyModeUnset ProviderSupplyMode = ""

	// The provider is built into the Terraform binary.
	ProviderSupplyModeBuiltIn ProviderSupplyMode = "built_in"

	// The provider is unmanaged by Terraform, supplied via TF_REATTACH_PROVIDERS by the user/environment.
	ProviderSupplyModeReattached ProviderSupplyMode = "reattached"

	// The provider is overridden by a development configuration.
	ProviderSupplyModeDevOverride ProviderSupplyMode = "dev_override"

	// The provider is managed by Terraform. This is the "normal" and most common case.
	ProviderSupplyModeManaged ProviderSupplyMode = "managed_by_terraform"
)
