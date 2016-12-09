package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
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
	real := &shadowComponentFactory{
		shadowComponentFactoryShared: shared,
	}

	// Create the shadow
	shadow := &shadowComponentFactory{
		shadowComponentFactoryShared: shared,
		Shadow: true,
	}

	return real, shadow
}

// shadowComponentFactory is the shadow side. Any components created
// with this factory are fake and will not cause real work to happen.
//
// Unlike other shadowers, the shadow component factory will allow the
// shadow to create _any_ component even if it is never requested on the
// real side. This is because errors will happen later downstream as function
// calls are made to the shadows that are never matched on the real side.
type shadowComponentFactory struct {
	*shadowComponentFactoryShared

	Shadow bool // True if this should return the shadow
	lock   sync.Mutex
}

func (f *shadowComponentFactory) ResourceProvider(
	n, uid string) (ResourceProvider, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	real, shadow, err := f.shadowComponentFactoryShared.ResourceProvider(n, uid)
	var result ResourceProvider = real
	if f.Shadow {
		result = shadow
	}

	return result, err
}

func (f *shadowComponentFactory) ResourceProvisioner(
	n, uid string) (ResourceProvisioner, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	real, shadow, err := f.shadowComponentFactoryShared.ResourceProvisioner(n, uid)
	var result ResourceProvisioner = real
	if f.Shadow {
		result = shadow
	}

	return result, err
}

// CloseShadow is called when the _real_ side is complete. This will cause
// all future blocking operations to return immediately on the shadow to
// ensure the shadow also completes.
func (f *shadowComponentFactory) CloseShadow() error {
	// If we aren't the shadow, just return
	if !f.Shadow {
		return nil
	}

	// Lock ourselves so we don't modify state
	f.lock.Lock()
	defer f.lock.Unlock()

	// Grab our shared state
	shared := f.shadowComponentFactoryShared

	// If we're already closed, its an error
	if shared.closed {
		return fmt.Errorf("component factory shadow already closed")
	}

	// Close all the providers and provisioners and return the error
	var result error
	for _, n := range shared.providerKeys {
		_, shadow, err := shared.ResourceProvider(n, n)
		if err == nil && shadow != nil {
			if err := shadow.CloseShadow(); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}

	for _, n := range shared.provisionerKeys {
		_, shadow, err := shared.ResourceProvisioner(n, n)
		if err == nil && shadow != nil {
			if err := shadow.CloseShadow(); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}

	// Mark ourselves as closed
	shared.closed = true

	return result
}

func (f *shadowComponentFactory) ShadowError() error {
	// If we aren't the shadow, just return
	if !f.Shadow {
		return nil
	}

	// Lock ourselves so we don't modify state
	f.lock.Lock()
	defer f.lock.Unlock()

	// Grab our shared state
	shared := f.shadowComponentFactoryShared

	// If we're not closed, its an error
	if !shared.closed {
		return fmt.Errorf("component factory must be closed to retrieve errors")
	}

	// Close all the providers and provisioners and return the error
	var result error
	for _, n := range shared.providerKeys {
		_, shadow, err := shared.ResourceProvider(n, n)
		if err == nil && shadow != nil {
			if err := shadow.ShadowError(); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}

	for _, n := range shared.provisionerKeys {
		_, shadow, err := shared.ResourceProvisioner(n, n)
		if err == nil && shadow != nil {
			if err := shadow.ShadowError(); err != nil {
				result = multierror.Append(result, err)
			}
		}
	}

	return result
}

// shadowComponentFactoryShared is shared data between the two factories.
//
// It is NOT SAFE to run any function on this struct in parallel. Lock
// access to this struct.
type shadowComponentFactoryShared struct {
	contextComponentFactory

	closed          bool
	providers       shadow.KeyedValue
	providerKeys    []string
	provisioners    shadow.KeyedValue
	provisionerKeys []string
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
	Shadow shadowResourceProvisioner
	Err    error
}

func (f *shadowComponentFactoryShared) ResourceProvider(
	n, uid string) (ResourceProvider, shadowResourceProvider, error) {
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

			if f.closed {
				shadow.CloseShadow()
			}
		}

		// Store the value
		f.providers.SetValue(uid, &entry)
		f.providerKeys = append(f.providerKeys, uid)
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
	n, uid string) (ResourceProvisioner, shadowResourceProvisioner, error) {
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
			real, shadow := newShadowResourceProvisioner(p)
			entry.Real = real
			entry.Shadow = shadow

			if f.closed {
				shadow.CloseShadow()
			}
		}

		// Store the value
		f.provisioners.SetValue(uid, &entry)
		f.provisionerKeys = append(f.provisionerKeys, uid)
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
