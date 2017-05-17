package dnsimple

import (
	"errors"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DNSIMPLE_EMAIL", ""),
				Description: "The DNSimple account email address.",
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DNSIMPLE_TOKEN", nil),
				Description: "The API v2 token for API operations.",
			},

			"account": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DNSIMPLE_ACCOUNT", nil),
				Description: "The account for API operations.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"dnsimple_record": resourceDNSimpleRecord(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	// DNSimple API v1 requires email+token to authenticate.
	// DNSimple API v2 requires only an OAuth token and in this particular case
	// the reference of the account for API operations (to avoid fetching it in real time).
	//
	// v2 is not backward compatible with v1, therefore return an error in case email is set,
	// to inform the user to upgrade to v2. Also, v1 token is not the same of v2.
	if email := d.Get("email").(string); email != "" {
		return nil, errors.New(
			"DNSimple API v2 requires an account identifier and the new OAuth token. " +
				"Please upgrade your configuration.")
	}

	config := Config{
		Token:   d.Get("token").(string),
		Account: d.Get("account").(string),
	}

	return config.Client()
}
