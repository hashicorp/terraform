package consul

import (
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func (b *Backend) States() ([]string, error) {
	return nil, backend.ErrNamedStatesNotSupported
}

func (b *Backend) DeleteState(name string) error {
	return backend.ErrNamedStatesNotSupported
}

func (b *Backend) State(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}

	// Get the Consul API client
	client, err := b.clientRaw()
	if err != nil {
		return nil, err
	}

	// Determine the path of the data
	path := b.configData.Get("path").(string)

	// Build the remote state client
	return &remote.State{
		Client: &RemoteClient{
			Client: client,
			Path:   path,
		},
	}, nil
}
