package inmem

import (
	"context"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/backend/remote-state"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

// New creates a new backend for Inmem remote state.
func New() backend.Backend {
	return &remotestate.Backend{
		ConfigureFunc: configure,

		// Set the schema
		Backend: &schema.Backend{
			Schema: map[string]*schema.Schema{
				"lock_id": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "initializes the state in a locked configuration",
				},
			},
		},
	}
}

func configure(ctx context.Context) (remote.Client, error) {
	data := schema.FromContextBackendConfig(ctx)
	if v, ok := data.GetOk("lock_id"); ok && v.(string) != "" {
		info := state.NewLockInfo()
		info.ID = v.(string)
		info.Operation = "test"
		info.Info = "test config"
		return &RemoteClient{LockInfo: info}, nil
	}
	return &RemoteClient{}, nil
}
