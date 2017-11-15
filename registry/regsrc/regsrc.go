// Package regsrc provides helpers for working with source strings that identify
// resources within a Terraform registry.
package regsrc

import "github.com/hashicorp/terraform/svchost"

var (
	// PublicRegistryHost is a FriendlyHost that represents the public registry.
	PublicRegistryHost, _ = svchost.New("registry.terraform.io")
)
