package cmd

import (
	"context"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"base_command": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "base command, to be called with one of PUT, GET, DELETE, LOCK, or UNLOCK as the only argument",
			},
			"state_transfer_file": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "path to the file that passes state between terraform and the base_command",
			},
			"lock_transfer_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "path to the file that passes lock between terraform and the base_command",
			},
		},
	}

	b := &Backend{Backend: s}
	b.Backend.ConfigureFunc = b.configure
	return b
}

type Backend struct {
	*schema.Backend
	client *CmdClient
}

func (b *Backend) configure(ctx context.Context) error {
	data := schema.FromContextBackendConfig(ctx)

	baseCmd := data.Get("base_command").(string)
	statesTransferFile := data.Get("state_transfer_file").(string)
	lockTransferFile := data.Get("lock_transfer_file").(string)

	b.client = &CmdClient{
		baseCmd:            baseCmd,
		statesTransferFile: statesTransferFile,
		lockTransferFile:   lockTransferFile,
	}
	return nil
}

func (b *Backend) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *Backend) DeleteWorkspace(string) error {
	return backend.ErrWorkspacesNotSupported
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}
	return &remote.State{
		Client: b.client,
	}, nil
}
