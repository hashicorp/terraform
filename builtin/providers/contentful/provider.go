package contentful

import (
	"os"

	contentful "github.com/contentful-labs/contentful-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider does shit
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"cma_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONTENTFUL_MANAGEMENT_TOKEN", nil),
				Description: "The Contentful Management API token",
			},
			"organization_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONTENTFUL_ORGANIZATION_ID", nil),
				Description: "The organization ID",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"contentful_space":       resourceContentfulSpace(),
			"contentful_contenttype": resourceContentfulContentType(),
			"contentful_apikey":      resourceContentfulAPIKey(),
			"contentful_webhook":     resourceContentfulWebhook(),
			"contentful_locale":      resourceContentfulLocale(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	cma := contentful.NewCMA(d.Get("cma_token").(string))
	cma.SetOrganization(d.Get("organization_id").(string))

	if os.Getenv("TF_LOG") != "" {
		cma.Debug = true
	}

	return cma, nil
}
