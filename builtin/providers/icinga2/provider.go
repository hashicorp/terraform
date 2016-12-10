package icinga2

import (
	"fmt"
	"net/url"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/lrsmith/go-icinga2-api/iapi"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_URL", nil),
				Description: descriptions["api_url"],
			},
			"api_user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_USER", nil),
				Description: descriptions["api_user"],
			},
			"api_password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ICINGA2_API_PASSWORD", nil),
				Description: descriptions["api_password"],
			},
			"insecure_skip_tls_verify": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: EnvBoolDefaultFunc("ICINGA2_INSECURE_SKIP_TLS_VERIFY", false),
				Description: descriptions["insecure_skip_tls_verify"],
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"icinga2_host":         resourceIcinga2Host(),
			"icinga2_hostgroup":    resourceIcinga2Hostgroup(),
			"icinga2_checkcommand": resourceIcinga2Checkcommand(),
			"icinga2_service":      resourceIcinga2Service(),
		},
		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {

	config, _ := iapi.New(
		d.Get("api_user").(string),
		d.Get("api_password").(string),
		d.Get("api_url").(string),
		d.Get("insecure_skip_tls_verify").(bool),
	)

	err := validateURL(d.Get("api_url").(string))

	if err := config.Connect(); err != nil {
		return nil, err
	}

	return config, err
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"api_url":                  "The address of the Icinga2 server.\n",
		"api_user":                 "The user to authenticate to the Icinga2 Server as.\n",
		"api_password":             "The password for authenticating to the Icinga2 server.\n",
		"insecure_skip_tls_verify": "Disable TLS verify when connecting to Icinga2 Server\n",
	}
}

func validateURL(urlString string) error {

	//ICINGA2_API_URL=https://127.0.0.1:4665/v1
	tokens, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	if tokens.Scheme != "https" {
		return fmt.Errorf("Error : Requests are only allowed to use the HTTPS protocol so that traffic remains encrypted.")
	}

	if tokens.Path != "/v1" {
		return fmt.Errorf("Error : Invalid API version %s specified. Only v1 is currently supported.", tokens.Path)
	}

	return nil
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
