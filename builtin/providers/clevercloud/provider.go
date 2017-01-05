package clevercloud

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/samber/go-clevercloud-api/clever"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLEVERCLOUD_ENDPOINT", "https://api.clever-cloud.com/v2"),
				Description: "Root URL of the targetted Clever-Cloud API.",
			},
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLEVERCLOUD_AUTH_TOKEN", nil),
				Description: "Auth token to use with the Clever-Cloud API.",
			},
			"secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLEVERCLOUD_AUTH_SECRET", nil),
				Description: "Secret key to use with the Clever-Cloud API.",
			},
			"org_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLEVERCLOUD_ORG_ID", nil),
				Description: "Organisation id to use with the Clever-Cloud API.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			// addons
			"clevercloud_addon_postgresql": resourceCleverCloudAddonPostgreSQL(),
			"clevercloud_addon_mysql":      resourceCleverCloudAddonMySQL(),
			"clevercloud_addon_redis":      resourceCleverCloudAddonRedis(),
			"clevercloud_addon_mongodb":    resourceCleverCloudAddonMongoDB(),

			// runtimes
			"clevercloud_application_python":  resourceCleverCloudApplicationPython(),
			"clevercloud_application_node":    resourceCleverCloudApplicationNode(),
			"clevercloud_application_ruby":    resourceCleverCloudApplicationRuby(),
			"clevercloud_application_golang":  resourceCleverCloudApplicationGolang(),
			"clevercloud_application_java":    resourceCleverCloudApplicationJava(),
			"clevercloud_application_php":     resourceCleverCloudApplicationPhp(),
			"clevercloud_application_static":  resourceCleverCloudApplicationStatic(),
			"clevercloud_application_rust":    resourceCleverCloudApplicationRust(),
			"clevercloud_application_haskell": resourceCleverCloudApplicationHaskell(),
			"clevercloud_application_docker":  resourceCleverCloudApplicationDocker(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &clever.ClientConfig{
		Endpoint:   d.Get("endpoint").(string),
		OrgId:      d.Get("org_id").(string),
		AuthToken:  d.Get("token").(string),
		AuthSecret: d.Get("secret").(string),
	}

	client, err := clever.NewClient(config)
	if err != nil {
		return nil, err
	}

	// Only to check if organisation exists
	_, err = client.GetOrganisation()
	if err != nil {
		return nil, err
	}

	return client, nil
}
