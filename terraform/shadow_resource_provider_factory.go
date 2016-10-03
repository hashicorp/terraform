package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/shadow"
)

// shadowResourceProviderFactory is a helper that takes an actual, original
// map of ResourceProvider factories and provides methods to create mappings
// for shadowed resource providers.
type shadowResourceProviderFactory struct {
	// Original is the original factory map
	Original map[string]ResourceProviderFactory

	shadows shadow.KeyedValue
}

type shadowResourceProviderFactoryEntry struct {
	Real   ResourceProvider
	Shadow shadowResourceProvider
	Err    error
}

// RealMap returns the factory map for the "real" side of the shadow. This
// is the side that does actual work.
// TODO: test
func (f *shadowResourceProviderFactory) RealMap() map[string]ResourceProviderFactory {
	m := make(map[string]ResourceProviderFactory)
	for k, _ := range f.Original {
		m[k] = f.realFactory(k)
	}

	return m
}

// ShadowMap returns the factory map for the "shadow" side of the shadow. This
// is the side that doesn't do any actual work but does compare results
// with the real side.
// TODO: test
func (f *shadowResourceProviderFactory) ShadowMap() map[string]ResourceProviderFactory {
	m := make(map[string]ResourceProviderFactory)
	for k, _ := range f.Original {
		m[k] = f.shadowFactory(k)
	}

	return m
}

func (f *shadowResourceProviderFactory) realFactory(n string) ResourceProviderFactory {
	return func() (ResourceProvider, error) {
		// Get the original factory function
		originalF, ok := f.Original[n]
		if !ok {
			return nil, fmt.Errorf("unknown provider initialized: %s", n)
		}

		// Build the entry
		var entry shadowResourceProviderFactoryEntry

		// Initialize it
		p, err := originalF()
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
		f.shadows.SetValue(n, &entry)

		// Return
		return entry.Real, entry.Err
	}
}

func (f *shadowResourceProviderFactory) shadowFactory(n string) ResourceProviderFactory {
	return func() (ResourceProvider, error) {
		// Get the value
		raw := f.shadows.Value(n)
		if raw == nil {
			return nil, fmt.Errorf(
				"Nil shadow value for provider %q. Please report this bug.",
				n)
		}

		entry, ok := raw.(*shadowResourceProviderFactoryEntry)
		if !ok {
			return nil, fmt.Errorf("Unknown value for shadow provider: %#v", raw)
		}

		// Return
		return entry.Shadow, entry.Err
	}
}
