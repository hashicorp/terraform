package openstack

import (
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// This is a global MutexKV for use within this plugin.
var osMutexKV = mutexkv.NewMutexKV()

// Provider returns a schema.Provider for OpenStack.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"auth_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_AUTH_URL", nil),
			},
			"user_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_USERNAME", ""),
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
				DefaultFunc: schema.EnvDefaultFunc("OS_TENANT_NAME", nil),
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_PASSWORD", ""),
			},
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_AUTH_TOKEN", ""),
			},
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_API_KEY", ""),
			},
			"domain_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_DOMAIN_ID", ""),
			},
			"domain_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_DOMAIN_NAME", ""),
			},
			"insecure": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"endpoint_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_ENDPOINT_TYPE", ""),
			},
			"cacert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_CACERT", ""),
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
			"openstack_lb_member_v1":                   resourceLBMemberV1(),
			"openstack_lb_monitor_v1":                  resourceLBMonitorV1(),
			"openstack_lb_pool_v1":                     resourceLBPoolV1(),
			"openstack_lb_vip_v1":                      resourceLBVipV1(),
			"openstack_networking_network_v2":          resourceNetworkingNetworkV2(),
			"openstack_networking_subnet_v2":           resourceNetworkingSubnetV2(),
			"openstack_networking_floatingip_v2":       resourceNetworkingFloatingIPV2(),
			"openstack_networking_port_v2":             resourceNetworkingPortV2(),
			"openstack_networking_router_v2":           resourceNetworkingRouterV2(),
			"openstack_networking_router_interface_v2": resourceNetworkingRouterInterfaceV2(),
			"openstack_networking_router_route_v2":     resourceNetworkingRouterRouteV2(),
			"openstack_networking_secgroup_v2":         resourceNetworkingSecGroupV2(),
			"openstack_networking_secgroup_rule_v2":    resourceNetworkingSecGroupRuleV2(),
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
		Token:            d.Get("token").(string),
		APIKey:           d.Get("api_key").(string),
		TenantID:         d.Get("tenant_id").(string),
		TenantName:       d.Get("tenant_name").(string),
		DomainID:         d.Get("domain_id").(string),
		DomainName:       d.Get("domain_name").(string),
		Insecure:         d.Get("insecure").(bool),
		EndpointType:     d.Get("endpoint_type").(string),
		CACertFile:       d.Get("cacert_file").(string),
	}

	if err := config.loadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}
