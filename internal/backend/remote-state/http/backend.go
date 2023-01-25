package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_ADDRESS", nil),
				Description: "The address of the REST endpoint",
			},
			"update_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_UPDATE_METHOD", "POST"),
				Description: "HTTP method to use when updating state",
			},
			"lock_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_LOCK_ADDRESS", nil),
				Description: "The address of the lock REST endpoint",
			},
			"unlock_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_UNLOCK_ADDRESS", nil),
				Description: "The address of the unlock REST endpoint",
			},
			"workspaces_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACES_ADDRESS", nil),
				Description: "The address of the workspaces REST endpoint",
			},
			"lock_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_LOCK_METHOD", "LOCK"),
				Description: "The HTTP method to use when locking",
			},
			"unlock_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_UNLOCK_METHOD", "UNLOCK"),
				Description: "The HTTP method to use when unlocking",
			},
			"workspaces_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACES_METHOD", "OPTIONS"),
				Description: "The HTTP method to use when listing workspaces",
			},
			"workspaces": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACES", false),
				Description: "Wether to enable workspaces use on REST endpoint",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_USERNAME", nil),
				Description: "The username for HTTP basic authentication",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_PASSWORD", nil),
				Description: "The password for HTTP basic authentication",
			},
			"skip_cert_verification": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to skip TLS verification.",
			},
			"retry_max": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_RETRY_MAX", 2),
				Description: "The number of HTTP request retries.",
			},
			"retry_wait_min": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_RETRY_WAIT_MIN", 1),
				Description: "The minimum time in seconds to wait between HTTP request attempts.",
			},
			"retry_wait_max": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_RETRY_WAIT_MAX", 30),
				Description: "The maximum time in seconds to wait between HTTP request attempts.",
			},
		},
	}

	b := &Backend{Backend: s}
	b.Backend.ConfigureFunc = b.configure
	return b
}

type Backend struct {
	*schema.Backend

	client *httpClient
}

func (b *Backend) configure(ctx context.Context) error {
	data := schema.FromContextBackendConfig(ctx)

	address := data.Get("address").(string)
	updateURL, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("failed to parse address URL: %s", err)
	}
	if updateURL.Scheme != "http" && updateURL.Scheme != "https" {
		return fmt.Errorf("address must be HTTP or HTTPS")
	}

	updateMethod := data.Get("update_method").(string)

	var lockURL *url.URL
	if v, ok := data.GetOk("lock_address"); ok && v.(string) != "" {
		var err error
		lockURL, err = url.Parse(v.(string))
		if err != nil {
			return fmt.Errorf("failed to parse lockAddress URL: %s", err)
		}
		if lockURL.Scheme != "http" && lockURL.Scheme != "https" {
			return fmt.Errorf("lockAddress must be HTTP or HTTPS")
		}
	}

	lockMethod := data.Get("lock_method").(string)

	var unlockURL *url.URL
	if v, ok := data.GetOk("unlock_address"); ok && v.(string) != "" {
		var err error
		unlockURL, err = url.Parse(v.(string))
		if err != nil {
			return fmt.Errorf("failed to parse unlockAddress URL: %s", err)
		}
		if unlockURL.Scheme != "http" && unlockURL.Scheme != "https" {
			return fmt.Errorf("unlockAddress must be HTTP or HTTPS")
		}
	}

	unlockMethod := data.Get("unlock_method").(string)

	var workspacesURL *url.URL
	if v, ok := data.GetOk("workspaces_address"); ok && v.(string) != "" {
		var err error
		workspacesURL, err = url.Parse(v.(string))
		if err != nil {
			return fmt.Errorf("failed to parse workspacesAddress URL: %s", err)
		}
		if workspacesURL.Scheme != "http" && workspacesURL.Scheme != "https" {
			return fmt.Errorf("workspacesAddress must be HTTP or HTTPS")
		}
	}

	workspacesMethod := data.Get("workspaces_method").(string)

	if data.Get("workspaces").(bool) {
		// Use default address if no workspaces_address is provided
		if workspacesURL == nil {
			workspacesURL = updateURL
		}
	}

	client := cleanhttp.DefaultPooledClient()

	if data.Get("skip_cert_verification").(bool) {
		// ignores TLS verification
		client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rClient := retryablehttp.NewClient()
	rClient.HTTPClient = client
	rClient.RetryMax = data.Get("retry_max").(int)
	rClient.RetryWaitMin = time.Duration(data.Get("retry_wait_min").(int)) * time.Second
	rClient.RetryWaitMax = time.Duration(data.Get("retry_wait_max").(int)) * time.Second
	rClient.Logger = log.New(logging.LogOutput(), "", log.Flags())

	b.client = &httpClient{
		URL:          updateURL,
		UpdateMethod: updateMethod,

		LockURL:          lockURL,
		LockMethod:       lockMethod,
		UnlockURL:        unlockURL,
		UnlockMethod:     unlockMethod,
		WorkspacesURL:    workspacesURL,
		WorkspacesMethod: workspacesMethod,

		Username: data.Get("username").(string),
		Password: data.Get("password").(string),

		// accessible only for testing use
		Client: rClient,
	}
	return nil
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	if name != backend.DefaultStateName {
		b.client.workspace = name
	}

	return &remote.State{Client: b.client}, nil
}

func (b *Backend) Workspaces() ([]string, error) {
	if b.client.WorkspacesURL == nil {
		return nil, backend.ErrWorkspacesNotSupported
	}

	workspaces, err := b.client.Workspaces()
	if err != nil {
		return nil, err
	}

	result := make([]string, 1, len(workspaces)+1)
	result[0] = backend.DefaultStateName
	result = append(result, workspaces...)
	return result, nil
}

func (b *Backend) DeleteWorkspace(name string, force bool) error {
	if b.client.WorkspacesURL == nil {
		return backend.ErrWorkspacesNotSupported
	}

	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	return b.client.Delete()
}
