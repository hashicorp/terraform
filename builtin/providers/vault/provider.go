package vault

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

// Provider returns a schema.Provider for managing Packet infrastructure.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"ca_cert": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"ca_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"auth_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "token",
			},

			"auth_config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			"allow_unverified_ssl": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"vault_audit_backend":  resourceVaultAuditBackend(),
			"vault_auth_backend":   resourceVaultAuthBackend(),
			"vault_secret_backend": resourceVaultSecretBackend(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := api.DefaultConfig()
	if v, ok := d.GetOk("address"); ok {
		config.Address = v.(string)
	}

	// Config setup is adapted from Vault's command/meta.go
	err := config.ReadEnvironment()
	if err != nil {
		return nil, err
	}

	tlsConfig := config.HttpClient.Transport.(*http.Transport).TLSClientConfig
	tlsConfig.InsecureSkipVerify = d.Get("allow_unverified_ssl").(bool)

	var certPool *x509.CertPool
	if v := d.Get("ca_cert").(string); v != "" {
		certPool, err = api.LoadCACert(v)
	} else if v := d.Get("ca_path").(string); v != "" {
		certPool, err = api.LoadCAPath(v)
	}
	if err != nil {
		return nil, fmt.Errorf("Error setting up CA path: %s", err)
	}

	if certPool != nil {
		tlsConfig.RootCAs = certPool
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	authMethod := d.Get("auth_method").(string)
	authConfig := make(map[string]string)
	for k, v := range d.Get("auth_config").(map[string]interface{}) {
		authConfig[k] = v.(string)
	}

	var authToken string
	switch authMethod {
	case "token":
		authToken = authConfig["token"]
	case "cert":
		if authConfig["client_cert"] == "" {
			return nil, fmt.Errorf(
				"Missing required field for cert auth: client_cert")
		}
		if authConfig["client_key"] == "" {
			return nil, fmt.Errorf(
				"Missing required field for cert auth: client_key")
		}

		tlsCert, err := tls.X509KeyPair(
			[]byte(authConfig["client_cert"]), []byte(authConfig["client_key"]))
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}

		mount := "cert"
		if v := authConfig["mount"]; v != "" {
			mount = v
		}
		path := fmt.Sprintf("auth/%s/login", mount)
		secret, err := client.Logical().Write(path, nil)
		if err != nil {
			return nil, err
		}
		if secret == nil {
			return "", fmt.Errorf(
				"Error while attempting client-cert auth with Vault: " +
					"empty response from credential provider")
		}
		authToken = secret.Auth.ClientToken
	}

	if authToken != "" {
		client.SetToken(authToken)
	}

	return client, nil
}
