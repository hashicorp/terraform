package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNetworkingFloatingIPV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingFloatingIPV2Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"pool": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"port_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"fixed_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceNetworkingFloatingIPV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))

	listOpts := floatingips.ListOpts{}

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

	pages, err := floatingips.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to list Floating IPs: %s", err)
	}

	allFloatingIPs, err := floatingips.ExtractFloatingIPs(pages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve Floating IPs: %s", err)
	}

	if len(allFloatingIPs) < 1 {
		return fmt.Errorf("No Floating IP found")
	}

	if len(allFloatingIPs) > 1 {
		return fmt.Errorf("More than one Floating IP found")
	}

	fip := allFloatingIPs[0]

	log.Printf("[DEBUG] Retrieved Floating IP %s: %+v", fip.ID, fip)
	d.SetId(fip.ID)

	d.Set("address", fip.FloatingIP)
	d.Set("pool", fip.FloatingNetworkID)
	d.Set("port_id", fip.PortID)
	d.Set("fixed_ip", fip.FixedIP)
	d.Set("tenant_id", fip.TenantID)
	d.Set("status", fip.Status)
	d.Set("region", GetRegion(d, config))

	return nil
}
