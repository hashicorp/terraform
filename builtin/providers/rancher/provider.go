package rancher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

type CLIConfig struct {
	AccessKey   string `json:"accessKey"`
	SecretKey   string `json:"secretKey"`
	URL         string `json:"url"`
	Environment string `json:"environment"`
	Path        string `json:"path,omitempty"`
}

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_URL", ""),
				Description: descriptions["api_url"],
			},
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_ACCESS_KEY", ""),
				Description: descriptions["access_key"],
			},
			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_SECRET_KEY", ""),
				Description: descriptions["secret_key"],
			},
			"config": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RANCHER_CLIENT_CONFIG", ""),
				Description: descriptions["config"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"rancher_certificate":         resourceRancherCertificate(),
			"rancher_environment":         resourceRancherEnvironment(),
			"rancher_host":                resourceRancherHost(),
			"rancher_registration_token":  resourceRancherRegistrationToken(),
			"rancher_registry":            resourceRancherRegistry(),
			"rancher_registry_credential": resourceRancherRegistryCredential(),
			"rancher_stack":               resourceRancherStack(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"access_key": "API Key used to authenticate with the rancher server",

		"secret_key": "API secret used to authenticate with the rancher server",

		"api_url": "The URL to the rancher API",

		"config": "Path to the Rancher client cli.json config file",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	apiURL := d.Get("api_url").(string)
	accessKey := d.Get("access_key").(string)
	secretKey := d.Get("secret_key").(string)

	if configFile := d.Get("config").(string); configFile != "" {
		config, err := loadConfig(configFile)
		if err != nil {
			return config, err
		}

		if apiURL == "" && config.URL != "" {
			u, err := url.Parse(config.URL)
			if err != nil {
				return config, err
			}
			apiURL = u.Scheme + "://" + u.Host
		}

		if accessKey == "" {
			accessKey = config.AccessKey
		}

		if secretKey == "" {
			secretKey = config.SecretKey
		}
	}

	if apiURL == "" {
		return &Config{}, fmt.Errorf("No api_url provided")
	}

	config := &Config{
		APIURL:    apiURL + "/v1",
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	_, err := config.GlobalClient()

	return config, err
}

func loadConfig(path string) (CLIConfig, error) {
	config := CLIConfig{
		Path: path,
	}

	content, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	} else if err != nil {
		return config, err
	}

	err = json.Unmarshal(content, &config)
	config.Path = path

	return config, err
}
