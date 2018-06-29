package configupgrade

import (
	"github.com/hashicorp/terraform/terraform"
)

// Upgrader is the main type in this package, containing all of the
// dependencies that are needed to perform upgrades.
type Upgrader struct {
	Providers    terraform.ResourceProviderResolver
	Provisioners map[string]terraform.ResourceProvisionerFactory
}
