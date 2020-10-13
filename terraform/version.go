package terraform

import (
	"github.com/hashicorp/terraform/version"
)

// Deprecated: Providers should use schema.Provider.TerraformVersion instead
func VersionString() string {
	return version.String()
}
