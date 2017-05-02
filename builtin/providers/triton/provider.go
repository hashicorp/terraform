package triton

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"sort"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"account": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TRITON_ACCOUNT", "SDC_ACCOUNT"}, ""),
			},

			"url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TRITON_URL", "SDC_URL"}, "https://us-west-1.api.joyentcloud.com"),
			},

			"key_material": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TRITON_KEY_MATERIAL", "SDC_KEY_MATERIAL"}, ""),
			},

			"key_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TRITON_KEY_ID", "SDC_KEY_ID"}, ""),
			},

			"insecure_skip_tls_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TRITON_SKIP_TLS_VERIFY", ""),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"triton_firewall_rule": resourceFirewallRule(),
			"triton_machine":       resourceMachine(),
			"triton_key":           resourceKey(),
			"triton_vlan":          resourceVLAN(),
			"triton_fabric":        resourceFabric(),
		},
		ConfigureFunc: providerConfigure,
	}
}

type Config struct {
	Account               string
	KeyMaterial           string
	KeyID                 string
	URL                   string
	InsecureSkipTLSVerify bool
}

func (c Config) validate() error {
	var err *multierror.Error

	if c.URL == "" {
		err = multierror.Append(err, errors.New("URL must be configured for the Triton provider"))
	}
	if c.KeyID == "" {
		err = multierror.Append(err, errors.New("Key ID must be configured for the Triton provider"))
	}
	if c.Account == "" {
		err = multierror.Append(err, errors.New("Account must be configured for the Triton provider"))
	}

	return err.ErrorOrNil()
}

func (c Config) getTritonClient() (*triton.Client, error) {
	var signer authentication.Signer
	var err error
	if c.KeyMaterial == "" {
		signer, err = authentication.NewSSHAgentSigner(c.KeyID, c.Account)
		if err != nil {
			return nil, errwrap.Wrapf("Error Creating SSH Agent Signer: {{err}}", err)
		}
	} else {
		signer, err = authentication.NewPrivateKeySigner(c.KeyID, []byte(c.KeyMaterial), c.Account)
		if err != nil {
			return nil, errwrap.Wrapf("Error Creating SSH Private Key Signer: {{err}}", err)
		}
	}

	client, err := triton.NewClient(c.URL, c.Account, signer)
	if err != nil {
		return nil, errwrap.Wrapf("Error Creating Triton Client: {{err}}", err)
	}

	if c.InsecureSkipTLSVerify {
		client.InsecureSkipTLSVerify()
	}

	return client, nil
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Account: d.Get("account").(string),
		URL:     d.Get("url").(string),
		KeyID:   d.Get("key_id").(string),

		InsecureSkipTLSVerify: d.Get("insecure_skip_tls_verify").(bool),
	}

	if keyMaterial, ok := d.GetOk("key_material"); ok {
		config.KeyMaterial = keyMaterial.(string)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	client, err := config.getTritonClient()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func resourceExists(resource interface{}, err error) (bool, error) {
	if err != nil {
		if triton.IsResourceNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return resource != nil, nil
}

func stableMapHash(input map[string]string) string {
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	hash := md5.New()
	for _, key := range keys {
		hash.Write([]byte(key))
		hash.Write([]byte(input[key]))
	}

	return base64.StdEncoding.EncodeToString(hash.Sum([]byte{}))
}

var fastResourceTimeout = &schema.ResourceTimeout{
	Create: schema.DefaultTimeout(1 * time.Minute),
	Read:   schema.DefaultTimeout(30 * time.Second),
	Update: schema.DefaultTimeout(1 * time.Minute),
	Delete: schema.DefaultTimeout(1 * time.Minute),
}

var slowResourceTimeout = &schema.ResourceTimeout{
	Create: schema.DefaultTimeout(10 * time.Minute),
	Read:   schema.DefaultTimeout(30 * time.Second),
	Update: schema.DefaultTimeout(10 * time.Minute),
	Delete: schema.DefaultTimeout(10 * time.Minute),
}
