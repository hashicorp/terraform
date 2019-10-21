package openstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

func resourceNetworkingSubnetRouteV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingSubnetRouteV2Create,
		Read:   resourceNetworkingSubnetRouteV2Read,
		Delete: resourceNetworkingSubnetRouteV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination_cidr": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"next_hop": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNetworkingSubnetRouteV2Create(d *schema.ResourceData, meta interface{}) error {

	subnetId := d.Get("subnet_id").(string)
	osMutexKV.Lock(subnetId)
	defer osMutexKV.Unlock(subnetId)

	var destCidr string = d.Get("destination_cidr").(string)
	var nextHop string = d.Get("next_hop").(string)

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	subnet, err := subnets.Get(networkingClient, subnetId).Extract()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving OpenStack Neutron Subnet: %s", err)
	}

	var updateOpts subnets.UpdateOpts
	var routeExists bool = false

	var rts []subnets.HostRoute = subnet.HostRoutes
	for _, r := range rts {

		if r.DestinationCIDR == destCidr && r.NextHop == nextHop {
			routeExists = true
			break
		}
	}

	if routeExists {
		return fmt.Errorf("Subnet %s has route already", subnetId)
	}

	if destCidr != "" && nextHop != "" {
		r := subnets.HostRoute{DestinationCIDR: destCidr, NextHop: nextHop}
		log.Printf("[INFO] Adding route %s", r)
		rts = append(rts, r)
	}

	updateOpts.HostRoutes = &rts

	log.Printf("[DEBUG] Updating Subnet %s with options: %+v", subnetId, updateOpts)

	_, err = subnets.Update(networkingClient, subnetId, updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack Neutron Subnet: %s", err)
	}
	id := fmt.Sprintf("%s-route-%s-%s", subnetId, destCidr, nextHop)
	d.SetId(id)

	return resourceNetworkingSubnetRouteV2Read(d, meta)
}

func resourceNetworkingSubnetRouteV2Read(d *schema.ResourceData, meta interface{}) error {

	subnetId := d.Get("subnet_id").(string)

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	destCidr := d.Get("destination_cidr").(string)
	nextHop := d.Get("next_hop").(string)

	routeIDParts := []string{}
	if d.Id() != "" && strings.Contains(d.Id(), "-route-") {
		routeIDParts = strings.Split(d.Id(), "-route-")
		routeLastIDParts := strings.Split(routeIDParts[1], "-")

		if subnetId == "" {
			subnetId = routeIDParts[0]
			d.Set("subnet_id", subnetId)
		}
		if destCidr == "" {
			destCidr = routeLastIDParts[0]
		}
		if nextHop == "" {
			nextHop = routeLastIDParts[1]
		}
	}

	subnet, err := subnets.Get(networkingClient, subnetId).Extract()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving OpenStack Neutron Subnet: %s", err)
	}

	log.Printf("[DEBUG] Retrieved Subnet %s: %+v", subnetId, subnet)

	var exists bool
	for _, r := range subnet.HostRoutes {
		if r.DestinationCIDR == destCidr && r.NextHop == nextHop {
			exists = true
		}
	}

	if !exists {
		return fmt.Errorf("Route doesn't exist")
	}

	d.Set("next_hop", nextHop)
	d.Set("destination_cidr", destCidr)

	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceNetworkingSubnetRouteV2Delete(d *schema.ResourceData, meta interface{}) error {

	subnetId := d.Get("subnet_id").(string)
	osMutexKV.Lock(subnetId)
	defer osMutexKV.Unlock(subnetId)

	config := meta.(*Config)

	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	subnet, err := subnets.Get(networkingClient, subnetId).Extract()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			return nil
		}

		return fmt.Errorf("Error retrieving OpenStack Neutron Subnet: %s", err)
	}

	var updateOpts subnets.UpdateOpts

	var destCidr string = d.Get("destination_cidr").(string)
	var nextHop string = d.Get("next_hop").(string)

	var oldRts []subnets.HostRoute = subnet.HostRoutes
	var newRts []subnets.HostRoute

	for _, r := range oldRts {

		if r.DestinationCIDR != destCidr || r.NextHop != nextHop {
			newRts = append(newRts, r)
		}
	}

	if len(oldRts) == len(newRts) {
		return fmt.Errorf("Route did not exist already")
	}

	r := subnets.HostRoute{DestinationCIDR: destCidr, NextHop: nextHop}
	log.Printf("[INFO] Deleting route %s", r)
	updateOpts.HostRoutes = &newRts

	log.Printf("[DEBUG] Updating Subnet %s with options: %+v", subnetId, updateOpts)

	_, err = subnets.Update(networkingClient, subnetId, updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack Neutron Subnet: %s", err)
	}

	return nil
}
