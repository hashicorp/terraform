package remote

import (
	"errors"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
)

// FromProvider produces a remote state object that wraps a state storage
// implementation offered by a particular provider.
func FromProvider(provider providers.Interface, storageType addrs.StateStorageType, config cty.Value) *State {
	client := &fromProviderClient{
		provider:    provider,
		storageType: storageType,
		config:      config,
	}
	return &State{
		Client: client,
	}
}

// fromProviderClient is an implementation of Client that wraps a state
// storage implementation offered by a provider.
type fromProviderClient struct {
	provider    providers.Interface
	storageType addrs.StateStorageType
	config      cty.Value
}

func (c *fromProviderClient) Get() (*Payload, error) {
	return nil, errors.New("not yet implemented")
}

func (c *fromProviderClient) Put([]byte) error {
	return errors.New("not yet implemented")
}

func (c *fromProviderClient) Delete() error {
	return errors.New("not yet implemented")
}
