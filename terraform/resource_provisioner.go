package terraform

// ResourceProvisioner is an interface that must be implemented by any
// resource provisioner: the thing that initializes resources in
// a Terraform configuration.
type ResourceProvisioner interface {
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
}

// ResourceProvisionerCloser is an interface that provisioners that can close
// connections that aren't needed anymore must implement.
type ResourceProvisionerCloser interface {
	Close() error
}

// ResourceProvisionerFactory is a function type that creates a new instance
// of a resource provisioner.
type ResourceProvisionerFactory func() (ResourceProvisioner, error)
