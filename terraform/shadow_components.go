package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/helper/shadow"
)

// newShadowComponentFactory creates a shadowed contextComponentFactory
// so that requests to create new components result in both a real and
// shadow side.
func newShadowComponentFactory(
	f contextComponentFactory) (contextComponentFactory, *shadowComponentFactory) {
	// Create the shared data
	shared := &shadowComponentFactoryShared{contextComponentFactory: f}

	// Create the real side
	real := &shadowComponentFactoryReal{
		shadowComponentFactoryShared: shared,
	}

	// Create the shadow
	shadow := &shadowComponentFactory{
		shadowComponentFactoryShared: shared,
	}

	return real, shadow
}

// shadowComponentFactory is the shadow side. Any components created
// with this factory are fake and will not cause real work to happen.
type shadowComponentFactory struct {
	*shadowComponentFactoryShared
}

func (f *shadowComponentFactory) ResourceProvider(
	n, uid string) (ResourceProvider, error) {
	_, shadow, err := f.shadowComponentFactoryShared.ResourceProvider(n, uid)
	return shadow, err
}

func (f *shadowComponentFactory) ResourceProvisioner(
	n, uid string) (ResourceProvisioner, error) {
	_, shadow, err := f.shadowComponentFactoryShared.ResourceProvisioner(n, uid)
	return shadow, err
}

// shadowComponentFactoryReal is the real side of the component factory.
// Operations here result in real components that do real work.
type shadowComponentFactoryReal struct {
	*shadowComponentFactoryShared
}

func (f *shadowComponentFactoryReal) ResourceProvider(
	n, uid string) (ResourceProvider, error) {
	real, _, err := f.shadowComponentFactoryShared.ResourceProvider(n, uid)
	return real, err
}

func (f *shadowComponentFactoryReal) ResourceProvisioner(
	n, uid string) (ResourceProvisioner, error) {
	real, _, err := f.shadowComponentFactoryShared.ResourceProvisioner(n, uid)
	return real, err
}

// shadowComponentFactoryShared is shared data between the two factories.
type shadowComponentFactoryShared struct {
	contextComponentFactory

	providers    shadow.KeyedValue
	provisioners shadow.KeyedValue
	lock         sync.Mutex
}

// shadowResourceProviderFactoryEntry is the entry that is stored in
// the Shadows key/value for a provider.
type shadowComponentFactoryProviderEntry struct {
	Real   ResourceProvider
	Shadow shadowResourceProvider
	Err    error
}

type shadowComponentFactoryProvisionerEntry struct {
	Real   ResourceProvisioner
	Shadow ResourceProvisioner
	Err    error
}

func (f *shadowComponentFactoryShared) ResourceProvider(
	n, uid string) (ResourceProvider, shadowResourceProvider, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Determine if we already have a value
	raw, ok := f.providers.ValueOk(uid)
	if !ok {
		// Build the entry
		var entry shadowComponentFactoryProviderEntry

		// No value, initialize. Create the original
		p, err := f.contextComponentFactory.ResourceProvider(n, uid)
		if err != nil {
			entry.Err = err
			p = nil // Just to be sure
		}

		if p != nil {
			// Create the shadow
			real, shadow := newShadowResourceProvider(p)
			entry.Real = real
			entry.Shadow = shadow
		}

		// Store the value
		f.providers.SetValue(uid, &entry)
		raw = &entry
	}

	// Read the entry
	entry, ok := raw.(*shadowComponentFactoryProviderEntry)
	if !ok {
		return nil, nil, fmt.Errorf("Unknown value for shadow provider: %#v", raw)
	}

	// Return
	return entry.Real, entry.Shadow, entry.Err
}

func (f *shadowComponentFactoryShared) ResourceProvisioner(
	n, uid string) (ResourceProvisioner, ResourceProvisioner, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Determine if we already have a value
	raw, ok := f.provisioners.ValueOk(uid)
	if !ok {
		// Build the entry
		var entry shadowComponentFactoryProvisionerEntry

		// No value, initialize. Create the original
		p, err := f.contextComponentFactory.ResourceProvisioner(n, uid)
		if err != nil {
			entry.Err = err
			p = nil // Just to be sure
		}

		if p != nil {
			// For now, just create a mock since we don't support provisioners yet
			real := p
			shadow := new(MockResourceProvisioner)
			entry.Real = real
			entry.Shadow = shadow
		}

		// Store the value
		f.provisioners.SetValue(uid, &entry)
		raw = &entry
	}

	// Read the entry
	entry, ok := raw.(*shadowComponentFactoryProvisionerEntry)
	if !ok {
		return nil, nil, fmt.Errorf("Unknown value for shadow provisioner: %#v", raw)
	}

	// Return
	return entry.Real, entry.Shadow, entry.Err
}
