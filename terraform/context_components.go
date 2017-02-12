package terraform

import (
	"fmt"
)

// contextComponentFactory is the interface that Context uses
// to initialize various components such as providers and provisioners.
// This factory gets more information than the raw maps using to initialize
// a Context. This information is used for debugging.
type contextComponentFactory interface {
	// ResourceProvider creates a new ResourceProvider with the given
	// type. The "uid" is a unique identifier for this provider being
	// initialized that can be used for internal tracking.
	ResourceProvider(typ, uid string) (ResourceProvider, error)
	ResourceProviders() []string

	// ResourceProvisioner creates a new ResourceProvisioner with the
	// given type. The "uid" is a unique identifier for this provisioner
	// being initialized that can be used for internal tracking.
	ResourceProvisioner(typ, uid string) (ResourceProvisioner, error)
	ResourceProvisioners() []string
}

// basicComponentFactory just calls a factory from a map directly.
type basicComponentFactory struct {
	providers    map[string]ResourceProviderFactory
	provisioners map[string]ResourceProvisionerFactory
}

func (c *basicComponentFactory) ResourceProviders() []string {
	result := make([]string, len(c.providers))
	for k, _ := range c.providers {
		result = append(result, k)
	}

	return result
}

func (c *basicComponentFactory) ResourceProvisioners() []string {
	result := make([]string, len(c.provisioners))
	for k, _ := range c.provisioners {
		result = append(result, k)
	}

	return result
}

func (c *basicComponentFactory) ResourceProvider(typ, uid string) (ResourceProvider, error) {
	f, ok := c.providers[typ]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", typ)
	}

	return f()
}

func (c *basicComponentFactory) ResourceProvisioner(typ, uid string) (ResourceProvisioner, error) {
	f, ok := c.provisioners[typ]
	if !ok {
		return nil, fmt.Errorf("unknown provisioner %q", typ)
	}

	return f()
}
