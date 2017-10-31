package datastore

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
)

// New creates a new Datastore backend.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"project": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Google Cloud project in which to use Datastore.",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"GOOGLE_PROJECT",
					"GCLOUD_PROJECT",
					"CLOUDSDK_CORE_PROJECT",
				}, nil),
			},
			"namespace": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Datastore namespace for this backend. Leave unset to use the default namespace.",
				DefaultFunc: schema.EnvDefaultFunc("GOOGLE_DATASTORE_NAMESPACE", nil),
			},
			"credentials_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to service account JSON credentials file. Leave unset to use Application Default Credentials.",
			},
		},
	}
	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

// A Backend backed by Google Cloud Datastore.
type Backend struct {
	*schema.Backend

	ds *datastore.Client
	ns string
}

func (b *Backend) configure(ctx context.Context) error {
	data := schema.FromContextBackendConfig(ctx)

	o := []option.ClientOption{}
	if f, ok := data.GetOk("credentials_file"); ok {
		o = []option.ClientOption{option.WithCredentialsFile(f.(string))}
	}

	var err error
	p := data.Get("project").(string)
	if b.ds, err = datastore.NewClient(ctx, p, o...); err != nil {
		return fmt.Errorf("cannot initialise Google Cloud Datastore client for project %v: %v", p, err)
	}

	b.ns = data.Get("namespace").(string)

	return nil
}
