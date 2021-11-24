package artifactory

import (
	"context"
	"fmt"
	"net/http"
	"os"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/utils/log"
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

	client     *ArtifactoryClient
	configData *schema.ResourceData
}

func (b *Backend) configure(ctx context.Context) error {
	b.configData = schema.FromContextBackendConfig(ctx)
	b.client = &ArtifactoryClient{}
	data := b.configData
	// jfrog-client-go requires setting the logger using one's own
	log.SetLogger(log.NewLogger(log.ERROR, os.Stderr))
	rtDetails := auth.NewArtifactoryDetails()

	if v, ok := data.GetOk("username"); ok && v.(string) != "" {
		rtDetails.SetUser(v.(string))
	}
	if v, ok := data.GetOk("password"); ok && v.(string) != "" {
		rtDetails.SetPassword(v.(string))
	}
	if v, ok := data.GetOk("url"); ok && v.(string) != "" {
		// url should be end in "/artifactory".
		// https://www.terraform.io/docs/language/settings/backends/artifactory.html
		// but jfrog-client-go expects url to end in "/artifactory/"
		rtDetails.SetUrl(v.(string) + "/")
	}
	if v, ok := data.GetOk("repo"); ok && v.(string) != "" {
		b.client.repo = v.(string)
	}
	if v, ok := data.GetOk("subpath"); ok && v.(string) != "" {
		b.client.subpath = v.(string)
	}

	httpClient := http.DefaultClient
	httpClient.Transport = cleanhttp.DefaultPooledTransport()

	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(rtDetails).
		SetContext(ctx).
		SetHttpClient(httpClient).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build an artifactory service config: %v", err)
	}
	rtManager, err := artifactory.New(serviceConfig)
	if err != nil {
		return fmt.Errorf("failed to create a artifactory client: %v", err)
	}
	b.client.nativeClient = rtManager

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
		Client: b.client,
	}, nil
}
