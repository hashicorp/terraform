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
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
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
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"domain_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"openstack_compute_instance_v2":   resourceComputeInstanceV2(),
			"openstack_compute_keypair_v2":    resourceComputeKeypairV2(),
			"openstack_compute_secgroup_v2":   resourceComputeSecGroupV2(),
			"openstack_lb_member_v1":          resourceLBMemberV1(),
			"openstack_lb_monitor_v1":         resourceLBMonitorV1(),
			"openstack_lb_pool_v1":            resourceLBPoolV1(),
			"openstack_lb_vip_v1":             resourceLBVipV1(),
			"openstack_networking_network_v2": resourceNetworkingNetworkV2(),
			"openstack_networking_subnet_v2":  resourceNetworkingSubnetV2(),
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
