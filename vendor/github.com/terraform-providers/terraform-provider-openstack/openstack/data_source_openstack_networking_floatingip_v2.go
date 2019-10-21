package openstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNetworkingFloatingIPV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingFloatingIPV2Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"address": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"pool": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"port_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"fixed_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"status": {
				Type:     schema.TypeString,
				Optional: true,
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

func dataSourceNetworkingFloatingIPV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	listOpts := floatingips.ListOpts{}

	if v, ok := d.GetOk("description"); ok {
		listOpts.Description = v.(string)
	}

	if v, ok := d.GetOk("address"); ok {
		listOpts.FloatingIP = v.(string)
	}

	if v, ok := d.GetOk("tenant_id"); ok {
		listOpts.TenantID = v.(string)
	}

	if v, ok := d.GetOk("pool"); ok {
		listOpts.FloatingNetworkID = v.(string)
	}

	if v, ok := d.GetOk("port_id"); ok {
		listOpts.PortID = v.(string)
	}

	if v, ok := d.GetOk("fixed_ip"); ok {
		listOpts.FixedIP = v.(string)
	}

	if v, ok := d.GetOk("status"); ok {
		listOpts.Status = v.(string)
	}

	tags := networkV2AttributesTags(d)
	if len(tags) > 0 {
		listOpts.Tags = strings.Join(tags, ",")
	}

	pages, err := floatingips.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to list openstack_networking_floatingips_v2: %s", err)
	}

	allFloatingIPs, err := floatingips.ExtractFloatingIPs(pages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve openstack_networking_floatingips_v2: %s", err)
	}

	if len(allFloatingIPs) < 1 {
		return fmt.Errorf("No openstack_networking_floatingip_v2 found")
	}

	if len(allFloatingIPs) > 1 {
		return fmt.Errorf("More than one openstack_networking_floatingip_v2 found")
	}

	fip := allFloatingIPs[0]

	log.Printf("[DEBUG] Retrieved openstack_networking_floatingip_v2 %s: %+v", fip.ID, fip)
	d.SetId(fip.ID)

	d.Set("description", fip.Description)
	d.Set("address", fip.FloatingIP)
	d.Set("pool", fip.FloatingNetworkID)
	d.Set("port_id", fip.PortID)
	d.Set("fixed_ip", fip.FixedIP)
	d.Set("tenant_id", fip.TenantID)
	d.Set("status", fip.Status)
	d.Set("all_tags", fip.Tags)
	d.Set("region", GetRegion(d, config))

	return nil
}
