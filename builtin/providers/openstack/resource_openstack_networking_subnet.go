package openstack

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jrperritt/terraform/helper/hashcode"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
)

func resourceNetworkingSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingSubnetCreate,
		Read:   resourceNetworkingSubnetRead,
		Update: resourceNetworkingSubnetUpdate,
		Delete: resourceNetworkingSubnetDelete,

		Schema: map[string]*schema.Schema{
			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"allocation_pools": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"end": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"gateway_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"ip_version": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"enable_dhcp": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"dns_nameservers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"host_routes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination_cidr": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"next_hop": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceNetworkingSubnetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	createOpts := subnets.CreateOpts{
		NetworkID:       d.Get("network_id").(string),
		CIDR:            d.Get("cidr").(string),
		Name:            d.Get("name").(string),
		TenantID:        d.Get("tenant_id").(string),
		AllocationPools: resourceSubnetAllocationPools(d),
		GatewayIP:       d.Get("gateway_ip").(string),
		IPVersion:       d.Get("ip_version").(int),
		DNSNameservers:  resourceSubnetDNSNameservers(d),
		HostRoutes:      resourceSubnetHostRoutes(d),
	}

	edRaw := d.Get("enable_dhcp").(string)
	if edRaw != "" {
		ed, err := strconv.ParseBool(edRaw)
		if err != nil {
			return fmt.Errorf("enable_dhcp, if provided, must be either 'true' or 'false'")
		}
		createOpts.EnableDHCP = &ed
	}

	log.Printf("[INFO] Requesting subnet creation")
	s, err := subnets.Create(osClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron subnet: %s", err)
	}
	log.Printf("[INFO] Subnet ID: %s", s.ID)

	d.SetId(s.ID)

	return resourceNetworkingSubnetRead(d, meta)
}

func resourceNetworkingSubnetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	s, err := subnets.Get(osClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack Neutron Subnet: %s", err)
	}

	log.Printf("[DEBUG] Retreived Subnet %s: %+v", d.Id(), s)

	d.Set("newtork_id", s.NetworkID)
	d.Set("cidr", s.CIDR)
	d.Set("ip_version", s.IPVersion)

	if _, exists := d.GetOk("name"); exists {
		if d.HasChange("name") {
			d.Set("name", s.Name)
		}
	} else {
		d.Set("name", "")
	}

	if _, exists := d.GetOk("tenant_id"); exists {
		if d.HasChange("tenant_id") {
			d.Set("tenant_id", s.Name)
		}
	} else {
		d.Set("tenant_id", "")
	}

	if _, exists := d.GetOk("allocation_pools"); exists {
		d.Set("allocation_pools", s.AllocationPools)
	}

	if _, exists := d.GetOk("gateway_ip"); exists {
		if d.HasChange("gateway_ip") {
			d.Set("gateway_ip", s.Name)
		}
	} else {
		d.Set("gateway_ip", "")
	}

	if _, exists := d.GetOk("enable_dhcp"); exists {
		if d.HasChange("enable_dhcp") {
			d.Set("enable_dhcp", strconv.FormatBool(s.EnableDHCP))
		}
	} else {
		d.Set("enable_dhcp", "")
	}

	if _, exists := d.GetOk("dns_nameservers"); exists {
		d.Set("dns_nameservers", s.DNSNameservers)
	}

	if _, exists := d.GetOk("host_routes"); exists {
		d.Set("host_routes", s.HostRoutes)
	}

	return nil
}

func resourceNetworkingSubnetUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	var updateOpts subnets.UpdateOpts

	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}

	if d.HasChange("gateway_ip") {
		updateOpts.GatewayIP = d.Get("gateway_ip").(string)
	}

	if d.HasChange("dns_nameservers") {
		updateOpts.DNSNameservers = resourceSubnetDNSNameservers(d)
	}

	if d.HasChange("host_routes") {
		updateOpts.HostRoutes = resourceSubnetHostRoutes(d)
	}

	if d.HasChange("enable_dhcp") {
		edRaw := d.Get("enable_dhcp").(string)
		if edRaw != "" {
			ed, err := strconv.ParseBool(edRaw)
			if err != nil {
				return fmt.Errorf("enable_dhcp, if provided, must be either 'true' or 'false'")
			}
			updateOpts.EnableDHCP = &ed
		}
	}

	log.Printf("[DEBUG] Updating Subnet %s with options: %+v", d.Id(), updateOpts)

	_, err := subnets.Update(osClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack Neutron Subnet: %s", err)
	}

	return resourceNetworkingSubnetRead(d, meta)
}

func resourceNetworkingSubnetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	err := subnets.Delete(osClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Subnet: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceSubnetAllocationPools(d *schema.ResourceData) []subnets.AllocationPool {
	rawAPs := d.Get("allocation_pools").([]interface{})
	aps := make([]subnets.AllocationPool, len(rawAPs))
	for i, raw := range rawAPs {
		rawMap := raw.(map[string]interface{})
		aps[i] = subnets.AllocationPool{
			Start: rawMap["start"].(string),
			End:   rawMap["end"].(string),
		}
	}
	return aps
}

func resourceSubnetDNSNameservers(d *schema.ResourceData) []string {
	rawDNSN := d.Get("dns_nameservers").(*schema.Set)
	dnsn := make([]string, rawDNSN.Len())
	for i, raw := range rawDNSN.List() {
		dnsn[i] = raw.(string)
	}
	return dnsn
}

func resourceSubnetHostRoutes(d *schema.ResourceData) []subnets.HostRoute {
	rawHR := d.Get("host_routes").([]interface{})
	hr := make([]subnets.HostRoute, len(rawHR))
	for i, raw := range rawHR {
		rawMap := raw.(map[string]interface{})
		hr[i] = subnets.HostRoute{
			DestinationCIDR: rawMap["destination_cidr"].(string),
			NextHop:         rawMap["next_hop"].(string),
		}
	}
	return hr
}
