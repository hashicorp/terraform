package config

import (
	"github.com/mitchellh/hashstructure"
)

// Terraform is the Terraform meta-configuration that can be present
// in configuration files for configuring Terraform itself.
type Terraform struct {
	RequiredVersion string   `hcl:"required_version"` // Required Terraform version (constraint)
	Backend         *Backend // See Backend struct docs
}

// Backend is the configuration for the "backend" to use with Terraform.
// A backend is responsible for all major behavior of Terraform's core.
// The abstraction layer above the core (the "backend") allows for behavior
// such as remote operation.
type Backend struct {
	Type      string
	RawConfig *RawConfig

	// Hash is a unique hash code representing the original configuration
	// of the backend. This won't be recomputed unless Rehash is called.
	Hash uint64
}

// Hash returns a unique content hash for this backend's configuration
// as a uint64 value.
func (b *Backend) Rehash() uint64 {
	// If we have no backend, the value is zero
	if b == nil {
		return 0
	}

	// Use hashstructure to hash only our type with the config.
	code, err := hashstructure.Hash(map[string]interface{}{
		"type":   b.Type,
		"config": b.RawConfig.Raw,
	}, nil)

	// This should never happen since we have just some basic primitives
	// so panic if there is an error.
	if err != nil {
		panic(err)
	}

	return code
}
