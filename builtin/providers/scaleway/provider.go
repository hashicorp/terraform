package scaleway

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func scwrcConfig() (map[string]string, error) {
	f, err := os.Open(fmt.Sprintf("%s/.scwrc", os.Getenv("HOME")))
	if err != nil {
		return nil, err
	}

	defer f.Close()
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var conf = make(map[string]string)
	if err := json.Unmarshal(bs, &conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func envWithScwrcFallbackFunc(envKey, fileKey string, dv interface{}) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(envKey); v != "" {
			return v, nil
		}

		if conf, err := scwrcConfig(); err == nil {
			return conf[fileKey], nil
		}

		return dv, nil
	}
}

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envWithScwrcFallbackFunc("SCALEWAY_ACCESS_KEY", "token", nil),
				Description: "The API key for Scaleway API operations.",
			},
			"organization": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envWithScwrcFallbackFunc("SCALEWAY_ORGANIZATION", "organization", nil),
				Description: "The Organization ID for Scaleway API operations.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"scaleway_server":              resourceScalewayServer(),
			"scaleway_ip":                  resourceScalewayIP(),
			"scaleway_security_group":      resourceScalewaySecurityGroup(),
			"scaleway_security_group_rule": resourceScalewaySecurityGroupRule(),
			"scaleway_volume":              resourceScalewayVolume(),
			"scaleway_volume_attachment":   resourceScalewayVolumeAttachment(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Organization: d.Get("organization").(string),
		APIKey:       d.Get("access_key").(string),
	}

	return config.Client()
}
