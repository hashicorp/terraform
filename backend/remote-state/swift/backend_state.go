package swift

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

	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
	}

	return &remote.State{Client: client}, nil
}
