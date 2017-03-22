package s3

import (
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	keyEnvPrefix = "-env:"
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

	return &remote.State{Client: b.client}, nil
}
