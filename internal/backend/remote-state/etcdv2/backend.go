// legacy etcd2.x backend

package etcdv2

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	etcdapi "go.etcd.io/etcd/client"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The path where to store the state",
			},
			"endpoints": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "A space-separated list of the etcd endpoints",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Username",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Password",
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend

	client etcdapi.Client
	path   string
}

func (b *Backend) configure(ctx context.Context) error {
	data := schema.FromContextBackendConfig(ctx)

	b.path = data.Get("path").(string)

	endpoints := data.Get("endpoints").(string)
	username := data.Get("username").(string)
	password := data.Get("password").(string)

	config := etcdapi.Config{
		Endpoints: strings.Split(endpoints, " "),
		Username:  username,
		Password:  password,
	}

	client, err := etcdapi.New(config)
	if err != nil {
		return err
	}

	b.client = client
	return nil
}

func (b *Backend) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *Backend) DeleteWorkspace(string) error {
	return backend.ErrWorkspacesNotSupported
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}
	return &remote.State{
		Client: &EtcdClient{
			Client: b.client,
			Path:   b.path,
		},
	}, nil
}
