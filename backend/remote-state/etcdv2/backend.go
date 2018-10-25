// legacy etcd2.x backend

package etcdv2

import (
	"context"
	"strings"

	etcdapi "github.com/coreos/etcd/client"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
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
				Description: "A space-separated list of the etcd endpoints<Paste>",
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
	return nil, backend.ErrNamedStatesNotSupported
}

func (b *Backend) DeleteWorkspace(string) error {
	return backend.ErrNamedStatesNotSupported
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}
	return &remote.State{
		Client: &EtcdClient{
			Client: b.client,
			Path:   b.path,
		},
	}, nil
}
