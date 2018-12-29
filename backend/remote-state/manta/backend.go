package manta

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	triton "github.com/joyent/triton-go"
	"github.com/joyent/triton-go/authentication"
	"github.com/joyent/triton-go/storage"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"account": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TRITON_ACCOUNT", "SDC_ACCOUNT"}, ""),
			},

			"user": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"TRITON_USER", "SDC_USER"}, ""),
			},

			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"MANTA_URL"}, "https://us-east.manta.joyent.com"),
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

			"path": {
				Type:     schema.TypeString,
				Required: true,
			},

			"object_name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "terraform.tfstate",
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend
	data *schema.ResourceData

	// The fields below are set from configure
	storageClient *storage.StorageClient
	path          string
	objectName    string
}

type BackendConfig struct {
	AccountId   string
	Username    string
	KeyId       string
	AccountUrl  string
	KeyMaterial string
	SkipTls     bool
}

func (b *Backend) configure(ctx context.Context) error {
	if b.path != "" {
		return nil
	}

	data := schema.FromContextBackendConfig(ctx)

	config := &BackendConfig{
		AccountId:  data.Get("account").(string),
		AccountUrl: data.Get("url").(string),
		KeyId:      data.Get("key_id").(string),
		SkipTls:    data.Get("insecure_skip_tls_verify").(bool),
	}

	if v, ok := data.GetOk("user"); ok {
		config.Username = v.(string)
	}

	if v, ok := data.GetOk("key_material"); ok {
		config.KeyMaterial = v.(string)
	}

	b.path = data.Get("path").(string)
	b.objectName = data.Get("object_name").(string)

	// If object_name is not set, try the deprecated objectName.
	if b.objectName == "" {
		b.objectName = data.Get("objectName").(string)
	}

	var validationError *multierror.Error

	if data.Get("account").(string) == "" {
		validationError = multierror.Append(validationError, errors.New("`Account` must be configured for the Triton provider"))
	}
	if data.Get("key_id").(string) == "" {
		validationError = multierror.Append(validationError, errors.New("`Key ID` must be configured for the Triton provider"))
	}
	if b.path == "" {
		validationError = multierror.Append(validationError, errors.New("`Path` must be configured for the Triton provider"))
	}

	if validationError != nil {
		return validationError
	}

	var signer authentication.Signer
	var err error

	if config.KeyMaterial == "" {
		input := authentication.SSHAgentSignerInput{
			KeyID:       config.KeyId,
			AccountName: config.AccountId,
			Username:    config.Username,
		}
		signer, err = authentication.NewSSHAgentSigner(input)
		if err != nil {
			return errwrap.Wrapf("Error Creating SSH Agent Signer: {{err}}", err)
		}
	} else {
		var keyBytes []byte
		if _, err = os.Stat(config.KeyMaterial); err == nil {
			keyBytes, err = ioutil.ReadFile(config.KeyMaterial)
			if err != nil {
				return fmt.Errorf("Error reading key material from %s: %s",
					config.KeyMaterial, err)
			}
			block, _ := pem.Decode(keyBytes)
			if block == nil {
				return fmt.Errorf(
					"Failed to read key material '%s': no key found", config.KeyMaterial)
			}

			if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
				return fmt.Errorf(
					"Failed to read key '%s': password protected keys are\n"+
						"not currently supported. Please decrypt the key prior to use.", config.KeyMaterial)
			}

		} else {
			keyBytes = []byte(config.KeyMaterial)
		}

		input := authentication.PrivateKeySignerInput{
			KeyID:              config.KeyId,
			PrivateKeyMaterial: keyBytes,
			AccountName:        config.AccountId,
			Username:           config.Username,
		}

		signer, err = authentication.NewPrivateKeySigner(input)
		if err != nil {
			return errwrap.Wrapf("Error Creating SSH Private Key Signer: {{err}}", err)
		}
	}

	clientConfig := &triton.ClientConfig{
		MantaURL:    config.AccountUrl,
		AccountName: config.AccountId,
		Username:    config.Username,
		Signers:     []authentication.Signer{signer},
	}
	triton, err := storage.NewClient(clientConfig)
	if err != nil {
		return err
	}

	b.storageClient = triton

	return nil
}
