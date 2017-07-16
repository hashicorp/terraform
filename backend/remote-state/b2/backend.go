package b2

import (
	"context"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"

	"gopkg.in/kothar/go-backblaze.v0"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the B2 bucket",
			},

			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The path to the state file inside the bucket",
			},

			"account_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Backblaze Account ID",
			},

			"application_key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "B2 Application Key",
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend

	b2 *backblaze.B2

	bucketName string
	keyName    string
}

func (b *Backend) configure(ctx context.Context) error {
	if b.b2 != nil {
		return nil
	}

	data := schema.FromContextBackendConfig(ctx)

	b.bucketName = data.Get("bucket").(string)
	b.keyName = data.Get("key").(string)

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      data.Get("account_id").(string),
		ApplicationKey: data.Get("application_key").(string),
	})

	if err != nil {
		return err
	}

	b.b2 = b2

	return nil
}
