package openstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNetworkingRouterV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingRouterV2Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"router_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"admin_state_up": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"distributed": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"external_network_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"enable_snat": {
				Type:     schema.TypeBool,
				Computed: true,
				Optional: true,
			},
			"availability_zone_hints": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"external_fixed_ip": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"tags": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"all_tags": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceNetworkingRouterV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	listOpts := routers.ListOpts{}

	if v, ok := d.GetOk("router_id"); ok {
		listOpts.ID = v.(string)
	}

	if v, ok := d.GetOk("name"); ok {
		listOpts.Name = v.(string)
	}

	if v, ok := d.GetOk("description"); ok {
		listOpts.Description = v.(string)
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

	tags := networkV2AttributesTags(d)
	if len(tags) > 0 {
		listOpts.Tags = strings.Join(tags, ",")
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
	d.Set("description", router.Description)
	d.Set("admin_state_up", router.AdminStateUp)
	d.Set("distributed", router.Distributed)
	d.Set("status", router.Status)
	d.Set("tenant_id", router.TenantID)
	d.Set("external_network_id", router.GatewayInfo.NetworkID)
	d.Set("enable_snat", router.GatewayInfo.EnableSNAT)
	d.Set("all_tags", router.Tags)
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
