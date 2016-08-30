package icinga2

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider comment
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_URL", nil),
				Description: "Full URL for the Icinga2 Server API",
			},
			"api_user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_USER", nil),
				Description: "API User to connect to the Icinga2 API Endpoint as",
			},
			"api_password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_PASSWORD", nil),
				Description: "API User's Password",
			},
			"insecure": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: EnvBoolDefaultFunc("ICINGA2_INSECURE_SKIP_TLS_VERIFY", false),
				Description: descriptions["insecure"],
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"icinga2_host":         resourceIcinga2Host(),
			"icinga2_hostgroup":    resourceIcinga2HostGroup(),
			"icinga2_checkcommand": resourceIcinga2Checkcommand(),
		},
		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		APIURL:      d.Get("api_url").(string),
		APIUser:     d.Get("api_user").(string),
		APIPassword: d.Get("api_password").(string),
		Insecure:    d.Get("insecure").(bool),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"api_url":      "The address of the Icinga2 server.\n",
		"api_user":     "The user to authenticate to the Iccinga2 Server as.\n",
		"api_password": "The password.\n",
	}
}

// EnvBoolDefaultFunc is a helper function that returns
func EnvBoolDefaultFunc(k string, dv interface{}) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v == "true" {
			return true, nil
		}

		return false, nil
	}
}
