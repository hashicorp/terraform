package artifactory

import (
	"context"
	"fmt"

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
				Description: "Username for state file access",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARTIFACTORY_PASSWORD", nil),
				Description: "Password for state file access",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARTIFACTORY_URL", nil),
				Description: "Artfactory base URL for state file access",
			},
			"repo": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The repository name for state file access",
			},
			"subpath": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path within the repository for state file access",
			},
			"lock_username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("LOCK_ARTIFACTORY_USERNAME", nil),
				Description: "username for lock file creation",
			},
			"lock_password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("LOCK_ARTIFACTORY_PASSWORD", nil),
				Description: "password for lock file creation",
			},
			"unlock_username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("UNLOCK_ARTIFACTORY_USERNAME", nil),
				Description: "username for lock file removal",
			},
			"unlock_password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("UNLOCK_ARTIFACTORY_PASSWORD", nil),
				Description: "password for lock file removal",
			},
			"lock_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("LOCK_ARTIFACTORY_URL", nil),
				Description: "artfactory base URL for lock file access",
			},
			"lock_repo": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The repository for lock file access",
			},
			"lock_subpath": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "path within the repository for lock file access",
			},
			"lock_readback_wait": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "milliseconds to wait before lock readback. default 0, means no readback",
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
	if userName == "" {
		return fmt.Errorf("username is missing")
	}
	if password == "" {
		return fmt.Errorf("password is missing")
	}
	if url == "" {
		return fmt.Errorf("url is missing")
	}
	if repo == "" {
		return fmt.Errorf("repo is missing")
	}
	if subpath == "" {
		return fmt.Errorf("subpath is missing")
	}

	lockUserName := data.Get("lock_username").(string)
	lockPassword := data.Get("lock_password").(string)
	unlockUserName := data.Get("unlock_username").(string)
	unlockPassword := data.Get("unlock_password").(string)
	lockUrl := data.Get("lock_url").(string)
	lockRepo := data.Get("lock_repo").(string)
	lockSubpath := data.Get("lock_subpath").(string)
	lockReadbackWait := data.Get("lock_readback_wait").(int)
	if unlockUserName == "" && lockUserName != "" {
		unlockUserName = lockUserName
	}
	if unlockPassword == "" && lockPassword != "" {
		unlockPassword = lockPassword
	}
	if lockUserName != "" || lockPassword != "" ||
		unlockUserName != "" || unlockPassword != "" ||
		lockUrl != "" || lockRepo != "" || lockSubpath != "" {
		if lockUserName == "" {
			return fmt.Errorf("lock_username is missing")
		}
		if lockPassword == "" {
			return fmt.Errorf("lock_password is missing")
		}
		if lockUrl == "" {
			return fmt.Errorf("lock_url is missing")
		}
		if lockRepo == "" {
			return fmt.Errorf("lock_repo is missing")
		}
		if lockSubpath == "" {
			return fmt.Errorf("lock_subpath is missing")
		}
	}

	clientConf := &artifactory.ClientConfig{
		BaseURL:  url,
		Username: userName,
		Password: password,
	}
	nativeClient := artifactory.NewClient(clientConf)

	lockClientConf := &artifactory.ClientConfig{
		BaseURL:  lockUrl,
		Username: lockUserName,
		Password: lockPassword,
	}
	lockNativeClient := artifactory.NewClient(lockClientConf)

	unlockClientConf := &artifactory.ClientConfig{
		BaseURL:  lockUrl,
		Username: unlockUserName,
		Password: unlockPassword,
	}
	unlockNativeClient := artifactory.NewClient(unlockClientConf)

	b.client = &ArtifactoryClient{
		nativeClient:       &nativeClient,
		lockNativeClient:   &lockNativeClient,
		unlockNativeClient: &unlockNativeClient,
		userName:           userName,
		password:           password,
		url:                url,
		repo:               repo,
		subpath:            subpath,
		lockUserName:       lockUserName,
		lockPassword:       lockPassword,
		unlockUserName:     unlockUserName,
		unlockPassword:     unlockPassword,
		lockUrl:            lockUrl,
		lockRepo:           lockRepo,
		lockSubpath:        lockSubpath,
		lockReadbackWait:   lockReadbackWait,
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
