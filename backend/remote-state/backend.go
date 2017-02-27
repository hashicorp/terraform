// Package remotestate implements a Backend for remote state implementations
// from the state/remote package that also implement a backend schema for
// configuration.
package remotestate

import (
	"context"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

// Backend implements backend.Backend for remote state backends.
//
// All exported fields should be set. This struct should only be used
// by implementers of backends, not by consumers. If you're consuming, please
// use a higher level package such as Consul backends.
type Backend struct {
	// Backend should be set to the configuration schema. ConfigureFunc
	// should not be set on the schema.
	*schema.Backend

	// ConfigureFunc takes the ctx from a schema.Backend and returns a
	// fully configured remote client to use for state operations.
	ConfigureFunc func(ctx context.Context) (remote.Client, error)

	client remote.Client
}

func (b *Backend) Configure(rc *terraform.ResourceConfig) error {
	// Set our configureFunc manually
	b.Backend.ConfigureFunc = func(ctx context.Context) error {
		c, err := b.ConfigureFunc(ctx)
		if err != nil {
			return err
		}

		// Set the client for later
		b.client = c
		return nil
	}

	// Run the normal configuration
	return b.Backend.Configure(rc)
}

func (b *Backend) States() ([]string, error) {
	return nil, backend.ErrNamedStatesNotSupported
}

func (b *Backend) DeleteState(name string) error {
	return backend.ErrNamedStatesNotSupported
}

func (b *Backend) State(name string) (state.State, error) {
	// This shouldn't happen
	if b.client == nil {
		panic("nil remote client")
	}

	if name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}

	s := &remote.State{Client: b.client}
	return s, nil
}
