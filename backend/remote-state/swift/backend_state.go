package swift

import (
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func (b *Backend) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *Backend) DeleteWorkspace(name string) error {
	return backend.ErrWorkspacesNotSupported
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
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
