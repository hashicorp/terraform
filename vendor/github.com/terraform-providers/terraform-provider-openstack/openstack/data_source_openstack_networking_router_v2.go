package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNetworkingRouterV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingRouterV2Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"router_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"distributed": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"external_network_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"enable_snat": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"availability_zone_hints": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"external_fixed_ip": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceNetworkingRouterV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))

	listOpts := routers.ListOpts{}

	if v, ok := d.GetOk("router_id"); ok {
		listOpts.ID = v.(string)
	}

	if v, ok := d.GetOk("name"); ok {
		listOpts.Name = v.(string)
	}

	if v, ok := d.GetOkExists("admin_state_up"); ok {
		asu := v.(bool)
		listOpts.AdminStateUp = &asu
	}

	if v, ok := d.GetOkExists("distributed"); ok {
		dist := v.(bool)
		listOpts.Distributed = &dist
	}

	if v, ok := d.GetOk("status"); ok {
		listOpts.Status = v.(string)
	}

	if v, ok := d.GetOk("tenant_id"); ok {
		listOpts.TenantID = v.(string)
	}

	pages, err := routers.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to list Routers: %s", err)
	}

	allRouters, err := routers.ExtractRouters(pages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Routers: %s", err)
	}

	if len(allRouters) < 1 {
		return fmt.Errorf("No Router found")
	}

	if len(allRouters) > 1 {
		return fmt.Errorf("More than one Router found")
	}

	router := allRouters[0]

	log.Printf("[DEBUG] Retrieved Router %s: %+v", router.ID, router)
	d.SetId(router.ID)

	d.Set("name", router.Name)
	d.Set("admin_state_up", router.AdminStateUp)
	d.Set("distributed", router.Distributed)
	d.Set("status", router.Status)
	d.Set("tenant_id", router.TenantID)
	d.Set("external_network_id", router.GatewayInfo.NetworkID)
	d.Set("enable_snat", router.GatewayInfo.EnableSNAT)
	d.Set("region", GetRegion(d, config))

	if err := d.Set("availability_zone_hints", router.AvailabilityZoneHints); err != nil {
		log.Printf("[DEBUG] Unable to set availability_zone_hints: %s", err)
	}

	var externalFixedIPs []map[string]string
	for _, v := range router.GatewayInfo.ExternalFixedIPs {
		externalFixedIPs = append(externalFixedIPs, map[string]string{
			"subnet_id":  v.SubnetID,
			"ip_address": v.IPAddress,
		})
	}
	if err = d.Set("external_fixed_ip", externalFixedIPs); err != nil {
		log.Printf("[DEBUG] Unable to set external_fixed_ip: %s", err)
	}
	return nil
}
