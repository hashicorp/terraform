// +build core

// This file is included whenever the 'core' build tag is specified. This is
// used by make core-dev and make core-test to compile a build significantly
// more quickly, but it will not include any provider or provisioner plugins.

package command

import "github.com/hashicorp/terraform/plugin"

var InternalProviders = map[string]plugin.ProviderFunc{}

var InternalProvisioners = map[string]plugin.ProvisionerFunc{}
