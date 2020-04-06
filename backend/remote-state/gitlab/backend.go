package gitlab

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/backend"
	backend_http "github.com/hashicorp/terraform/backend/remote-state/http"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	paramBaseUrl              = "base_url"
	paramProjectId            = "project_id"
	paramStateName            = "state_name"
	paramToken                = "token"
	paramSkipCertVerification = "skip_cert_verification"
	paramRetryMax             = "retry_max"
	paramRetryWaitMin         = "retry_wait_min"
	paramRetryWaitMax         = "retry_wait_max"
)

func New() backend.Backend {
	// See https://docs.gitlab.com/ee/user/project/new_ci_build_permissions_model.html#job-token for info
	// about CI_JOB_TOKEN environment variable (used below).

	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			paramBaseUrl: {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("GITLAB_BASE_URL", "https://gitlab.com"),
				Description:  "The GitLab base API URL",
				ValidateFunc: validation.NoZeroValues,
			},
			paramProjectId: {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc("CI_PROJECT_ID", nil),
				Description:  "The unique id of a GitLab project",
				ValidateFunc: validation.NoZeroValues,
			},
			paramStateName: {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      backend.DefaultStateName,
				Description:  "The name of the state",
				InputDefault: backend.DefaultStateName,
				ValidateFunc: validation.NoZeroValues,
			},
			paramToken: {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"GITLAB_TOKEN", "CI_JOB_TOKEN"}, nil),
				Description:  "The OAuth token used to connect to GitLab",
				Sensitive:    true,
				ValidateFunc: validation.NoZeroValues,
			},
			paramSkipCertVerification: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to skip TLS verification",
			},
			paramRetryMax: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      2,
				Description:  "The number of HTTP request retries",
				ValidateFunc: validation.IntAtLeast(0),
			},
			paramRetryWaitMin: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      1,
				Description:  "The minimum time in seconds to wait between HTTP request attempts",
				ValidateFunc: validation.IntAtLeast(0),
			},
			paramRetryWaitMax: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      30,
				Description:  "The maximum time in seconds to wait between HTTP request attempts",
				ValidateFunc: validation.IntAtLeast(0),
			},
		},
	}

	b := &Backend{Backend: s}
	b.Backend.ConfigureFunc = b.configure
	return b
}

type Backend struct {
	*schema.Backend
	client remote.Client
}

func (b *Backend) configure(ctx context.Context) error {
	data := schema.FromContextBackendConfig(ctx)

	projectId := data.Get(paramProjectId).(string)
	stateName := data.Get(paramStateName).(string)
	token := data.Get(paramToken).(string)
	baseURLstr := data.Get(paramBaseUrl).(string)
	if !strings.HasSuffix(baseURLstr, "/") {
		baseURLstr += "/"
	}
	baseURLstr += terraformStatePath(projectId, stateName)
	baseURL, err := url.Parse(baseURLstr)
	if err != nil {
		return fmt.Errorf("failed to parse address URL: %v", err)
	}

	client := cleanhttp.DefaultPooledClient()

	if data.Get(paramSkipCertVerification).(bool) {
		// ignores TLS verification
		client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rClient := retryablehttp.NewClient()
	rClient.HTTPClient = client
	rClient.RetryMax = data.Get(paramRetryMax).(int)
	rClient.RetryWaitMin = time.Duration(data.Get(paramRetryWaitMin).(int)) * time.Second
	rClient.RetryWaitMax = time.Duration(data.Get(paramRetryWaitMax).(int)) * time.Second

	b.client = &backend_http.RemoteClient{
		URL:          baseURL,
		UpdateMethod: http.MethodPost,
		LockURL:      nil, // TODO
		LockMethod:   http.MethodPost,
		UnlockURL:    nil, // TODO
		UnlockMethod: http.MethodDelete,
		Client:       rClient,
		UserAgent:    "terraform/gitlab-backend",
		Username:     "terraform", // does not matter, token encodes the user
		Password:     token,
	}
	return nil
}

func (b *Backend) StateMgr(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrWorkspacesNotSupported
	}

	return &remote.State{Client: b.client}, nil
}

func (b *Backend) Workspaces() ([]string, error) {
	return nil, backend.ErrWorkspacesNotSupported
}

func (b *Backend) DeleteWorkspace(string) error {
	return backend.ErrWorkspacesNotSupported
}

func terraformStatePath(projectId, stateName string) string {
	return fmt.Sprintf("api/v4/projects/%s/terraform/state/%s", pathEscape(projectId), pathEscape(stateName))
}

func pathEscape(s string) string {
	return strings.Replace(url.PathEscape(s), ".", "%2E", -1)
}
