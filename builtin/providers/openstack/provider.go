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
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_AUTH_TOKEN"),
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
			"insecure": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"endpoint_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_ENDPOINT_TYPE"),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"openstack_blockstorage_volume_v1":         resourceBlockStorageVolumeV1(),
			"openstack_compute_instance_v2":            resourceComputeInstanceV2(),
			"openstack_compute_keypair_v2":             resourceComputeKeypairV2(),
			"openstack_compute_secgroup_v2":            resourceComputeSecGroupV2(),
			"openstack_compute_servergroup_v2":         resourceComputeServerGroupV2(),
			"openstack_compute_floatingip_v2":          resourceComputeFloatingIPV2(),
			"openstack_fw_firewall_v1":                 resourceFWFirewallV1(),
			"openstack_fw_policy_v1":                   resourceFWPolicyV1(),
			"openstack_fw_rule_v1":                     resourceFWRuleV1(),
			"openstack_lb_monitor_v1":                  resourceLBMonitorV1(),
			"openstack_lb_pool_v1":                     resourceLBPoolV1(),
			"openstack_lb_vip_v1":                      resourceLBVipV1(),
			"openstack_networking_network_v2":          resourceNetworkingNetworkV2(),
			"openstack_networking_subnet_v2":           resourceNetworkingSubnetV2(),
			"openstack_networking_floatingip_v2":       resourceNetworkingFloatingIPV2(),
			"openstack_networking_port_v2":             resourceNetworkingPortV2(),
			"openstack_networking_router_v2":           resourceNetworkingRouterV2(),
			"openstack_networking_router_interface_v2": resourceNetworkingRouterInterfaceV2(),
			"openstack_objectstorage_container_v1":     resourceObjectStorageContainerV1(),
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
		Insecure:         d.Get("insecure").(bool),
		EndpointType:     d.Get("endpoint_type").(string),
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

func envDefaultFuncAllowMissing(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		v := os.Getenv(k)
		return v, nil
	}
}
