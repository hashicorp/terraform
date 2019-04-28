package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The address of the REST endpoint",
			},
			"update_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "POST",
				Description: "HTTP method to use when updating state",
			},
			"lock_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The address of the lock REST endpoint",
			},
			"unlock_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The address of the unlock REST endpoint",
			},
			"lock_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "LOCK",
				Description: "The HTTP method to use when locking",
			},
			"unlock_method": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "UNLOCK",
				Description: "The HTTP method to use when unlocking",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The username for HTTP basic authentication",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The password for HTTP basic authentication",
			},
			"cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the certificate file for TLS authentication",
			},
			"key_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the key file for TLS authentication",
			},
			"ca_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The path to the CA file for TLS authentication",
			},
			"skip_cert_verification": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to skip TLS verification.",
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

	client := cleanhttp.DefaultPooledClient()

	if data.Get("skip_cert_verification").(bool) {
		// ignores TLS verification
		client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	if data.Get("cert_file").(string) != "" {
		// Load the client cert
		cert, err := tls.LoadX509KeyPair(data.Get("cert_file").(string), data.Get("key_file").(string))
		if err != nil {
			return fmt.Errorf("failed to load client certificate: %s", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if data.Get("ca_file").(string) != "" {
			// Load the CA cert
			caCert, err := ioutil.ReadFile(data.Get("ca_file").(string))
			if err != nil {
				return fmt.Errorf("failed to load CA file: %s", err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}

		tlsConfig.BuildNameToCertificate()
		client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
	}

	b.client = &httpClient{
		URL:          updateURL,
		UpdateMethod: updateMethod,

		LockURL:      lockURL,
		LockMethod:   lockMethod,
		UnlockURL:    unlockURL,
		UnlockMethod: unlockMethod,

		Username: data.Get("username").(string),
		Password: data.Get("password").(string),

		// accessible only for testing use
		Client: client,
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
