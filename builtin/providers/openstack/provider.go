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
			},
			"user_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_USERNAME"),
			},
			"user_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_USERID"),
			},
			"tenant_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_TENANT_ID"),
			},
			"tenant_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_TENANT_NAME"),
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_PASSWORD"),
			},
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_API_KEY"),
			},
			"domain_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_DOMAIN_ID"),
			},
			"domain_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFunc("OS_DOMAIN_NAME"),
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
		IdentityEndpoint: d.Get("auth_url").(string),
		Username:         d.Get("user_name").(string),
		UserID:           d.Get("user_id").(string),
		Password:         d.Get("password").(string),
		APIKey:           d.Get("api_key").(string),
		TenantID:         d.Get("tenant_id").(string),
		TenantName:       d.Get("tenant_name").(string),
		DomainID:         d.Get("domain_id").(string),
		DomainName:       d.Get("domain_name").(string),
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
