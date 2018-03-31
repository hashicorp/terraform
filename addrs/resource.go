package addrs

// Resource is an address for a resource block within configuration, which
// contains potentially-multiple resource instances if that configuration
// block uses "count" or "for_each".
type Resource struct {
	referenceable
	Mode ResourceMode
	Type string
	Name string
}

// Instance produces the address for a specific instance of the receiver
// that is idenfied by the given key.
func (r Resource) Instance(key InstanceKey) ResourceInstance {
	return ResourceInstance{
		Resource: r,
		Key:      key,
	}
}

// Absolute returns an AbsResource from the receiver and the given module
// instance address.
func (r Resource) Absolute(module ModuleInstance) AbsResource {
	return AbsResource{
		Module:   module,
		Resource: r,
	}
}

// ResourceInstance is an address for a specific instance of a resource.
// When a resource is defined in configuration with "count" or "for_each" it
// produces zero or more instances, which can be addressed using this type.
type ResourceInstance struct {
	referenceable
	Resource Resource
	Key      InstanceKey
}

// Absolute returns an AbsResourceInstance from the receiver and the given module
// instance address.
func (r ResourceInstance) Absolute(module ModuleInstance) AbsResourceInstance {
	return AbsResourceInstance{
		Module:   module,
		Resource: r,
	}
}

// Resource returns the address of a particular resource within the receiver.
func (m ModuleInstance) Resource(mode ResourceMode, typeName string, name string) AbsResource {
	return AbsResource{
		Module: m,
		Resource: Resource{
			Mode: mode,
			Type: typeName,
			Name: name,
		},
	}
}

// AbsResource is an absolute address for a resource under a given module path.
type AbsResource struct {
	Module   ModuleInstance
	Resource Resource
}

// AbsResourceInstance is an absolute address for a resource instance under a
// given module path.
type AbsResourceInstance struct {
	Module   ModuleInstance
	Resource ResourceInstance
}

// ResourceInstance returns the address of a particular resource instance within the receiver.
func (m ModuleInstance) ResourceInstance(mode ResourceMode, typeName string, name string, key InstanceKey) AbsResourceInstance {
	return AbsResourceInstance{
		Module: m,
		Resource: ResourceInstance{
			Resource: Resource{
				Mode: mode,
				Type: typeName,
				Name: name,
			},
			Key: key,
		},
	}
}

// ResourceMode defines which lifecycle applies to a given resource. Each
// resource lifecycle has a slightly different address format.
type ResourceMode rune

//go:generate stringer -type ResourceMode

const (
	// InvalidResourceMode is the zero value of ResourceMode and is not
	// a valid resource mode.
	InvalidResourceMode ResourceMode = 0

	// ManagedResourceMode indicates a managed resource, as defined by
	// "resource" blocks in configuration.
	ManagedResourceMode ResourceMode = 'M'

	// DataResourceMode indicates a data resource, as defined by
	// "data" blocks in configuration.
	DataResourceMode ResourceMode = 'D'
)
