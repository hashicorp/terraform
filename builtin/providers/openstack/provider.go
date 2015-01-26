package openstack

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for OpenStack.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"auth_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("OS_AUTH_URL"),
				Description: descriptions["auth_url"],
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("OS_USERNAME"),
				Description: descriptions["username"],
			},

			"tenant_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_TENANT_NAME"),
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("OS_PASSWORD"),
				Description: descriptions["password"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"openstack_compute_instance":     resourceComputeInstance(),
			"openstack_compute_keypair":      resourceComputeKeypair(),
			"openstack_compute_secgroup":     resourceComputeSecGroup(),
			"openstack_compute_secgrouprule": resourceComputeSecGroupRule(),
			"openstack_lb_member":            resourceLBMember(),
			"openstack_lb_monitor":           resourceLBMonitor(),
			"openstack_lb_pool":              resourceLBPool(),
			"openstack_lb_vip":               resourceLBVip(),
			"openstack_networking_network":   resourceNetworkingNetwork(),
			"openstack_networking_subnet":    resourceNetworkingSubnet(),
		},

		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Region:           d.Get("region").(string),
		IdentityEndpoint: d.Get("auth_url").(string),
		Username:         d.Get("username").(string),
		Password:         d.Get("password").(string),
		TenantName:       d.Get("tenant_name").(string),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return nil, nil
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"region":   "The region where OpenStack operations will take place.",
		"auth_url": "The endpoint against which to authenticate.",
		"username": "The username with which to authenticate.",
		"password": "The password with which to authenticate.",
	}
}
