package configupgrade

import (
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/terraform"
)

// Upgrader is the main type in this package, containing all of the
// dependencies that are needed to perform upgrades.
type Upgrader struct {
	Providers    providers.Resolver
	Provisioners map[string]terraform.ProvisionerFactory
}
