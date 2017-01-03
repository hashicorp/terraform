package circonus

import (
	"bytes"
	"fmt"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

const (
	apiURLAttr = "api_url"
	keyAttr    = "key"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			apiURLAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://api.circonus.com/v2",
				Description: "URL of the Circonus API",
			},
			keyAttr: {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CIRCONUS_API_TOKEN", nil),
				Description: "API token used to authenticate with the Circonus API",
				Sensitive:   true,
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"circonus_account": dataSourceCirconusAccount(),
			"circonus_broker":  dataSourceCirconusBroker(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"circonus_check": resourceCheckBundle(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &api.Config{
		URL:      d.Get(apiURLAttr).(string),
		TokenKey: d.Get(keyAttr).(string),
		TokenApp: tfAppName(),
	}

	c, err := api.NewAPI(config)
	if err != nil {
		return nil, errwrap.Wrapf("Error initializing Circonus: %s", err)
	}

	return c, nil
}

func tfAppName() string {
	const VersionPrerelease = terraform.VersionPrerelease
	var versionString bytes.Buffer

	fmt.Fprintf(&versionString, "Terraform v%s", terraform.Version)
	if terraform.VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "-%s", terraform.VersionPrerelease)
	}

	return versionString.String()
}
