package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/hashicorp/terraform/states/remote"
	"github.com/hashicorp/terraform/states/statemgr"
)

var (
	ErrWorkspaceDisabled = errors.New("workspace_enabled is not true, workspaces disabled")
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
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_SKIP_CERT", false),
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
			"workspace_enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACE_ENABLED", false),
				Description: "Enable workspace support.",
			},
			"workspace_path_element": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACE_PATH_ELEMENT", "<workspace>"),
				Description: "The URL path string to replace with the active workspace name.",
			},
			"workspace_list_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACE_LIST_ADDRESS", nil),
				Description: "The address of the workspace list REST endpoint.",
			},
			"workspace_list_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACE_LIST_METHOD", "GET"),
				Description: "The HTTP method to use when fetching workspace list",
			},
			"workspace_delete_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACE_DELETE_ADDRESS", nil),
				Description: "The address of the workspace delete REST endpoint.",
			},
			"workspace_delete_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TF_HTTP_WORKSPACE_DELETE_METHOD", "DELETE"),
				Description: "The HTTP method to use when deleting a workspace.",
			},
			"headers": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				DefaultFunc: func() (interface{}, error) {
					v, err := schema.EnvDefaultFunc("TF_HTTP_HEADERS", nil)()
					if err != nil || v == nil {
						return nil, err
					}
					decode := map[string]interface{}{}
					err = json.Unmarshal([]byte(v.(string)), &decode)
					if err != nil {
						return nil, err
					}
					return decode, nil
				},
				Elem: &schema.Schema{
					Type:        schema.TypeString,
					Description: "Header Value",
				},
			},
		},
	}

	b := &Backend{Backend: s}
	b.Backend.ConfigureFunc = b.configure
	return b
}

type Backend struct {
	*schema.Backend

	workspaceEnabled     bool
	workspacePathElement string

	updateURL *url.URL
	lockURL   *url.URL
	unlockURL *url.URL

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
	b.updateURL = updateURL

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
	b.lockURL = lockURL

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
	b.unlockURL = unlockURL

	headers := map[string]string{}
	rawHeaders := data.Get("headers").(map[string]interface{})
	if rawHeaders != nil {
		for k, v := range rawHeaders {
			headers[k] = v.(string)
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

	b.client = &httpClient{
		Headers:      headers,
		URL:          updateURL,
		UpdateMethod: updateMethod,

		LockURL:      lockURL,
		LockMethod:   lockMethod,
		UnlockURL:    unlockURL,
		UnlockMethod: unlockMethod,

		Username: data.Get("username").(string),
		Password: data.Get("password").(string),

		// accessible only for testing use
		Client: rClient,
	}

	b.workspaceEnabled = data.Get("workspace_enabled").(bool)

	if b.workspaceEnabled {
		b.workspacePathElement = data.Get("workspace_path_element").(string)
		if b.workspacePathElement == "" {
			return fmt.Errorf("workspace_path_element required when workspace_enabled is true")
		}

		workspaceListURL, err := url.Parse(data.Get("workspace_list_address").(string))
		if err != nil {
			return fmt.Errorf("failed to parse workspace_list_address URL: %s", err)
		}
		if workspaceListURL.Scheme != "http" && workspaceListURL.Scheme != "https" {
			return fmt.Errorf("workspace_list_address must be HTTP or HTTPS")
		}
		workspaceListMethod := data.Get("workspace_list_method").(string)

		// optional
		var workspaceDeleteURL *url.URL
		if v, ok := data.GetOk("workspace_delete_address"); ok && v.(string) != "" {
			var err error
			workspaceDeleteURL, err = url.Parse(data.Get("workspace_delete_address").(string))
			if err != nil {
				return fmt.Errorf("failed to parse workspace_delete_address URL: %s", err)
			}
			if workspaceDeleteURL.Scheme != "http" && workspaceDeleteURL.Scheme != "https" {
				return fmt.Errorf("workspace_delete_address must be HTTP or HTTPS")
			}
		} else { // default to stateUrl
			u := *updateURL
			workspaceDeleteURL = &u
		}
		workspaceDeleteMethod := data.Get("workspace_delete_method").(string)

		b.client.WorkspaceListURL = workspaceListURL
		b.client.WorkspaceListMethod = workspaceListMethod
		b.client.WorkspaceDeleteURL = workspaceDeleteURL
		b.client.WorkspaceDeleteMethod = workspaceDeleteMethod
	}

	return nil
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	if b.workspaceEnabled {
		updateUrl, err := b.workspaceUrlSubstitute(b.updateURL, b.workspacePathElement, name)
		if err != nil {
			return nil, err
		}
		b.client.URL = updateUrl

		lockUrl, err := b.workspaceUrlSubstitute(b.lockURL, b.workspacePathElement, name)
		if err != nil {
			return nil, err
		}
		b.client.LockURL = lockUrl

		unlockUrl, err := b.workspaceUrlSubstitute(b.unlockURL, b.workspacePathElement, name)
		if err != nil {
			return nil, err
		}
		b.client.UnlockURL = unlockUrl

	} else {
		if name != backend.DefaultStateName {
			return nil, ErrWorkspaceDisabled
		}
	}

	return &remote.State{Client: b.client}, nil
}

func (b *Backend) Workspaces() ([]string, error) {
	if !b.workspaceEnabled {
		return nil, ErrWorkspaceDisabled
	}
	return b.client.WorkspaceList()
}

func (b *Backend) workspaceUrlSubstitute(u *url.URL, old string, new string) (*url.URL, error) {
	origPath := u.RawPath
	if origPath == "" {
		origPath = u.Path
	}
	newPath := strings.ReplaceAll(origPath, old, new)
	newUrl, err := u.Parse(newPath)
	if err != nil {
		return nil, err
	}
	return newUrl, nil
}

func (b *Backend) DeleteWorkspace(del string) error {
	if !b.workspaceEnabled {
		return ErrWorkspaceDisabled
	}
	u, err := b.workspaceUrlSubstitute(b.client.WorkspaceDeleteURL, b.workspacePathElement, del)
	if err != nil {
		return err
	}
	return b.client.WorkspaceDelete(u)
}
