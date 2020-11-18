package plugin

import (
	"github.com/hashicorp/terraform/internal/legacy/terraform"
)

func testProviderFixed(p terraform.ResourceProvider) ProviderFunc {
	return func() terraform.ResourceProvider {
		return p
	}
}

func testProvisionerFixed(p terraform.ResourceProvisioner) ProvisionerFunc {
	return func() terraform.ResourceProvisioner {
		return p
	}
}
