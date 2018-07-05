package terraform

import (
	"github.com/hashicorp/terraform/configs/configschema"
)

// ResourceProvisioner is an interface that must be implemented by any
// resource provisioner: the thing that initializes resources in
// a Terraform configuration.
type ResourceProvisioner interface {
	// GetConfigSchema returns the schema for the provisioner type's main
	// configuration block. This is called prior to Validate to enable some
	// basic structural validation to be performed automatically and to allow
	// the configuration to be properly extracted from potentially-ambiguous
	// configuration file formats.
	GetConfigSchema() (*configschema.Block, error)

	// Validate is called once at the beginning with the raw
	// configuration (no interpolation done) and can return a list of warnings
	// and/or errors.
	//
	// This is called once per resource.
	//
	// This should not assume any of the values in the resource configuration
	// are valid since it is possible they have to be interpolated still.
	// The primary use case of this call is to check that the required keys
	// are set and that the general structure is correct.
	Validate(*ResourceConfig) ([]string, []error)

	// Apply runs the provisioner on a specific resource and returns the new
	// resource state along with an error. Instead of a diff, the ResourceConfig
	// is provided since provisioners only run after a resource has been
	// newly created.
	Apply(UIOutput, *InstanceState, *ResourceConfig) error

	// Stop is called when the provisioner should halt any in-flight actions.
	//
	// This can be used to make a nicer Ctrl-C experience for Terraform.
	// Even if this isn't implemented to do anything (just returns nil),
	// Terraform will still cleanly stop after the currently executing
	// graph node is complete. However, this API can be used to make more
	// efficient halts.
	//
	// Stop doesn't have to and shouldn't block waiting for in-flight actions
	// to complete. It should take any action it wants and return immediately
	// acknowledging it has received the stop request. Terraform core will
	// automatically not make any further API calls to the provider soon
	// after Stop is called (technically exactly once the currently executing
	// graph nodes are complete).
	//
	// The error returned, if non-nil, is assumed to mean that signaling the
	// stop somehow failed and that the user should expect potentially waiting
	// a longer period of time.
	Stop() error
}

// ResourceProvisionerCloser is an interface that provisioners that can close
// connections that aren't needed anymore must implement.
type ResourceProvisionerCloser interface {
	Close() error
}

// ResourceProvisionerFactory is a function type that creates a new instance
// of a resource provisioner.
type ResourceProvisionerFactory func() (ResourceProvisioner, error)
