package addrs

import (
	"fmt"
	"strings"
)

// Resource is an address for a resource block within configuration, which
// contains potentially-multiple resource instances if that configuration
// block uses "count" or "for_each".
type Resource struct {
	referenceable
	Mode ResourceMode
	Type string
	Name string
}

func (r Resource) String() string {
	switch r.Mode {
	case ManagedResourceMode:
		return fmt.Sprintf("%s.%s", r.Type, r.Name)
	case DataResourceMode:
		return fmt.Sprintf("data.%s.%s", r.Type, r.Name)
	default:
		// Should never happen, but we'll return a string here rather than
		// crashing just in case it does.
		return fmt.Sprintf("<invalid>.%s.%s", r.Type, r.Name)
	}
}

func (r Resource) Equal(o Resource) bool {
	return r.String() == o.String()
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

// DefaultProviderConfig returns the address of the provider configuration
// that should be used for the resource identified by the reciever if it
// does not have a provider configuration address explicitly set in
// configuration.
//
// This method is not able to verify that such a configuration exists, nor
// represent the behavior of automatically inheriting certain provider
// configurations from parent modules. It just does a static analysis of the
// receiving address and returns an address to start from, relative to the
// same module that contains the resource.
func (r Resource) DefaultProviderConfig() ProviderConfig {
	typeName := r.Type
	if under := strings.Index(typeName, "_"); under != -1 {
		typeName = typeName[:under]
	}

	return ProviderConfig{
		Type: NewLegacyProvider(typeName),
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

func (r ResourceInstance) ContainingResource() Resource {
	return r.Resource
}

func (r ResourceInstance) String() string {
	if r.Key == NoKey {
		return r.Resource.String()
	}
	return r.Resource.String() + r.Key.String()
}

func (r ResourceInstance) Equal(o ResourceInstance) bool {
	return r.String() == o.String()
}

// Absolute returns an AbsResourceInstance from the receiver and the given module
// instance address.
func (r ResourceInstance) Absolute(module ModuleInstance) AbsResourceInstance {
	return AbsResourceInstance{
		Module:   module,
		Resource: r,
	}
}

// AbsResource is an absolute address for a resource under a given module path.
type AbsResource struct {
	targetable
	Module   ModuleInstance
	Resource Resource
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

// Instance produces the address for a specific instance of the receiver
// that is idenfied by the given key.
func (r AbsResource) Instance(key InstanceKey) AbsResourceInstance {
	return AbsResourceInstance{
		Module:   r.Module,
		Resource: r.Resource.Instance(key),
	}
}

// TargetContains implements Targetable by returning true if the given other
// address is either equal to the receiver or is an instance of the
// receiver.
func (r AbsResource) TargetContains(other Targetable) bool {
	switch to := other.(type) {

	case AbsResource:
		// We'll use our stringification as a cheat-ish way to test for equality.
		return to.String() == r.String()

	case AbsResourceInstance:
		return r.TargetContains(to.ContainingResource())

	default:
		return false

	}
}

func (r AbsResource) String() string {
	if len(r.Module) == 0 {
		return r.Resource.String()
	}
	return fmt.Sprintf("%s.%s", r.Module.String(), r.Resource.String())
}

func (r AbsResource) Equal(o AbsResource) bool {
	return r.String() == o.String()
}

// AbsResourceInstance is an absolute address for a resource instance under a
// given module path.
type AbsResourceInstance struct {
	targetable
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

// ContainingResource returns the address of the resource that contains the
// receving resource instance. In other words, it discards the key portion
// of the address to produce an AbsResource value.
func (r AbsResourceInstance) ContainingResource() AbsResource {
	return AbsResource{
		Module:   r.Module,
		Resource: r.Resource.ContainingResource(),
	}
}

// TargetContains implements Targetable by returning true if the given other
// address is equal to the receiver.
func (r AbsResourceInstance) TargetContains(other Targetable) bool {
	switch to := other.(type) {

	case AbsResourceInstance:
		// We'll use our stringification as a cheat-ish way to test for equality.
		return to.String() == r.String()

	default:
		return false

	}
}

func (r AbsResourceInstance) String() string {
	if len(r.Module) == 0 {
		return r.Resource.String()
	}
	return fmt.Sprintf("%s.%s", r.Module.String(), r.Resource.String())
}

func (r AbsResourceInstance) Equal(o AbsResourceInstance) bool {
	return r.String() == o.String()
}

// Less returns true if the receiver should sort before the given other value
// in a sorted list of addresses.
func (r AbsResourceInstance) Less(o AbsResourceInstance) bool {
	switch {

	case len(r.Module) != len(o.Module):
		return len(r.Module) < len(o.Module)

	case r.Module.String() != o.Module.String():
		return r.Module.Less(o.Module)

	case r.Resource.Resource.Mode != o.Resource.Resource.Mode:
		return r.Resource.Resource.Mode == DataResourceMode

	case r.Resource.Resource.Type != o.Resource.Resource.Type:
		return r.Resource.Resource.Type < o.Resource.Resource.Type

	case r.Resource.Resource.Name != o.Resource.Resource.Name:
		return r.Resource.Resource.Name < o.Resource.Resource.Name

	case r.Resource.Key != o.Resource.Key:
		return InstanceKeyLess(r.Resource.Key, o.Resource.Key)

	default:
		return false

	}
}

// ResourceMode defines which lifecycle applies to a given resource. Each
// resource lifecycle has a slightly different address format.
type ResourceMode rune

//go:generate go run golang.org/x/tools/cmd/stringer -type ResourceMode

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
