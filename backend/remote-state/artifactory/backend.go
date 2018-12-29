package artifactory

import (
	"context"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	artifactory "github.com/lusis/go-artifactory/src/artifactory.v401"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARTIFACTORY_USERNAME", nil),
				Description: "Username",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARTIFACTORY_PASSWORD", nil),
				Description: "Password",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARTIFACTORY_URL", nil),
				Description: "Artfactory base URL",
			},
			"repo": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The repository name",
			},
			"subpath": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path within the repository",
			},
		},
	}

	b := &Backend{Backend: s}
	b.Backend.ConfigureFunc = b.configure
	return b
}

type Backend struct {
	*schema.Backend

	client *ArtifactoryClient
}

func (b *Backend) configure(ctx context.Context) error {
	data := schema.FromContextBackendConfig(ctx)

	userName := data.Get("username").(string)
	password := data.Get("password").(string)
	url := data.Get("url").(string)
	repo := data.Get("repo").(string)
	subpath := data.Get("subpath").(string)

	clientConf := &artifactory.ClientConfig{
		BaseURL:  url,
		Username: userName,
		Password: password,
	}
	nativeClient := artifactory.NewClient(clientConf)

	b.client = &ArtifactoryClient{
		nativeClient: &nativeClient,
		userName:     userName,
		password:     password,
		url:          url,
		repo:         repo,
		subpath:      subpath,
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
