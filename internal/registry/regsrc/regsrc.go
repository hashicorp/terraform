// Package regsrc provides helpers for working with source strings that identify
// resources within a Terraform registry.
package regsrc

var (
	// PublicRegistryHost is a FriendlyHost that represents the public registry.
	PublicRegistryHost = NewFriendlyHost("registry.terraform.io")
)
