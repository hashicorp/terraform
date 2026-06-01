// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providercache"
)

var _ providercache.InstallerHook = stateStorageProviderInstallHook{}

type stateStorageProviderInstallHook struct {
	provider     tfaddr.Provider
	priorVersion *providerreqs.Version
	supplyMode   getproviders.ProviderSupplyMode
	reconfigure  bool
}

func (h stateStorageProviderInstallHook) ProviderVersionSelected(ctx context.Context, provider addrs.Provider, version string) error {
	if !provider.Equals(h.provider) {
		return nil // irrelevant
	}

	if h.priorVersion == nil {
		return nil // not an upgrade, install for first time
	}

	// if h.supplyMode != getproviders.ManagedByTerraform {
	// 	return nil // not managed by Terraform, so upgrades won't change this provider
	// }

	if h.reconfigure {
		return nil // user has opted out of state migration so no error
	}

	v := providerreqs.MustParseVersion(version)
	if v.Same(*h.priorVersion) {
		return nil // not an upgrade, same version selected
	}

	return fmt.Errorf(`Cannot upgrade the provider used for state storage during "terraform init -upgrade".

While upgrading providers Terraform attempted to upgrade the %s (%q) provider, which is used by the state_store block in your configuration.
Please use "terraform state migrate -upgrade" to upgrade the state store provider and navigate migrating your state between the two versions. You can then re-attempt "terraform init -upgrade" to upgrade the rest of your providers.

If you do not intend to upgrade the state store provider, please update your configuration to pin to the current version (%s), and re-run "terraform init -upgrade" to upgrade the rest of your providers.
`,
		provider.Type,
		provider.ForDisplay(),
		h.priorVersion,
	)
}
