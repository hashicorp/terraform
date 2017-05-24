package rackspace

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	osSubnets "github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
	rsSubnets "github.com/rackspace/gophercloud/rackspace/networking/v2/subnets"
)

func resourceNetworkingSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingSubnetCreate,
		Read:   resourceNetworkingSubnetRead,
		Update: resourceNetworkingSubnetUpdate,
		Delete: resourceNetworkingSubnetDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("RS_REGION_NAME"),
			},
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
			"host_routes": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination": &schema.Schema{
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
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	createOpts := osSubnets.CreateOpts{
		NetworkID:       d.Get("network_id").(string),
		CIDR:            d.Get("cidr").(string),
		Name:            d.Get("name").(string),
		AllocationPools: resourceSubnetAllocationPools(d),
		GatewayIP:       d.Get("gateway_ip").(string),
		IPVersion:       d.Get("ip_version").(int),
		HostRoutes:      resourceSubnetHostRoutes(d),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	s, err := rsSubnets.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating Rackspace Neutron subnet: %s", err)
	}
	log.Printf("[INFO] Subnet ID: %s", s.ID)

	d.SetId(s.ID)

	return resourceNetworkingSubnetRead(d, meta)
}

func resourceNetworkingSubnetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	s, err := rsSubnets.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "subnet")
	}

	log.Printf("[DEBUG] Retreived Subnet %s: %+v", d.Id(), s)

	d.Set("newtork_id", s.NetworkID)
	d.Set("cidr", s.CIDR)
	d.Set("ip_version", s.IPVersion)
	d.Set("name", s.Name)
	d.Set("allocation_pools", s.AllocationPools)
	d.Set("gateway_ip", s.GatewayIP)
	d.Set("host_routes", s.HostRoutes)

	return nil

}

func resourceNetworkingSubnetUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts osSubnets.UpdateOpts

	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}

	if d.HasChange("gateway_ip") {
		updateOpts.GatewayIP = d.Get("gateway_ip").(string)
	}

	log.Printf("[DEBUG] Updating Rackspace subnet %s with options: %+v", d.Id(), updateOpts)

	_, err = rsSubnets.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating Rackspace Neutron Subnet: %s", err)
	}

	return resourceNetworkingSubnetRead(d, meta)
}

func resourceNetworkingSubnetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingClient(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating Rackspace networking client: %s", err)
	}

	err = rsSubnets.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting Rackspace Neutron Subnet: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceSubnetAllocationPools(d *schema.ResourceData) []osSubnets.AllocationPool {
	rawAPs := d.Get("allocation_pools").([]interface{})
	aps := make([]osSubnets.AllocationPool, len(rawAPs))
	for i, raw := range rawAPs {
		rawMap := raw.(map[string]interface{})
		aps[i] = osSubnets.AllocationPool{
			Start: rawMap["start"].(string),
			End:   rawMap["end"].(string),
		}
	}
	return aps
}

func resourceSubnetHostRoutes(d *schema.ResourceData) []osSubnets.HostRoute {
	rawHR := d.Get("host_routes").([]interface{})
	hr := make([]osSubnets.HostRoute, len(rawHR))
	for i, raw := range rawHR {
		rawMap := raw.(map[string]interface{})
		hr[i] = osSubnets.HostRoute{
			DestinationCIDR: rawMap["destination"].(string),
			NextHop:         rawMap["next_hop"].(string),
		}
	}
	return hr
}
