package config

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/mitchellh/hashstructure"
)

// Terraform is the Terraform meta-configuration that can be present
// in configuration files for configuring Terraform itself.
type Terraform struct {
	RequiredVersion string   `hcl:"required_version"` // Required Terraform version (constraint)
	Backend         *Backend // See Backend struct docs
}

// Validate performs the validation for just the Terraform configuration.
func (t *Terraform) Validate() []error {
	var errs []error

	if raw := t.RequiredVersion; raw != "" {
		// Check that the value has no interpolations
		rc, err := NewRawConfig(map[string]interface{}{
			"root": raw,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"terraform.required_version: %s", err))
		} else if len(rc.Interpolations) > 0 {
			errs = append(errs, fmt.Errorf(
				"terraform.required_version: cannot contain interpolations"))
		} else {
			// Check it is valid
			_, err := version.NewConstraint(raw)
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"terraform.required_version: invalid syntax: %s", err))
			}
		}
	}

	if t.Backend != nil {
		errs = append(errs, t.Backend.Validate()...)
	}

	return errs
}

// Merge t with t2.
// Any conflicting fields are overwritten by t2.
func (t *Terraform) Merge(t2 *Terraform) {
	if t2.RequiredVersion != "" {
		t.RequiredVersion = t2.RequiredVersion
	}

	if t2.Backend != nil {
		t.Backend = t2.Backend
	}
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

// Rehash returns a unique content hash for this backend's configuration
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

func (b *Backend) Validate() []error {
	if len(b.RawConfig.Interpolations) > 0 {
		return []error{fmt.Errorf(strings.TrimSpace(errBackendInterpolations))}
	}

	return nil
}

const errBackendInterpolations = `
terraform.backend: configuration cannot contain interpolations

The backend configuration is loaded by Terraform extremely early, before
the core of Terraform can be initialized. This is necessary because the backend
dictates the behavior of that core. The core is what handles interpolation
processing. Because of this, interpolations cannot be used in backend
configuration.

If you'd like to parameterize backend configuration, we recommend using
partial configuration with the "-backend-config" flag to "terraform init".
`
