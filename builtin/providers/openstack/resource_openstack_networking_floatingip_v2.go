package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/pagination"
)

func resourceNetworkingFloatingIPV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkFloatingIPV2Create,
		Read:   resourceNetworkFloatingIPV2Read,
		Update: resourceNetworkFloatingIPV2Update,
		Delete: resourceNetworkFloatingIPV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"pool": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("OS_POOL_NAME"),
			},
			"port_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceNetworkFloatingIPV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	poolID, err := getNetworkID(d, meta, d.Get("pool").(string))
	if err != nil {
		return fmt.Errorf("Error retrieving floating IP pool name: %s", err)
	}
	if len(poolID) == 0 {
		return fmt.Errorf("No network found with name: %s", d.Get("pool").(string))
	}
	createOpts := floatingips.CreateOpts{
		FloatingNetworkID: poolID,
		PortID:            d.Get("port_id").(string),
	}
	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	floatingIP, err := floatingips.Create(networkClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error allocating floating IP: %s", err)
	}

	d.SetId(floatingIP.ID)

	return resourceNetworkFloatingIPV2Read(d, meta)
}

func resourceNetworkFloatingIPV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	floatingIP, err := floatingips.Get(networkClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "floating IP")
	}

	d.Set("address", floatingIP.FloatingIP)
	d.Set("port_id", floatingIP.PortID)
	poolName, err := getNetworkName(d, meta, floatingIP.FloatingNetworkID)
	if err != nil {
		return fmt.Errorf("Error retrieving floating IP pool name: %s", err)
	}
	d.Set("pool", poolName)

	return nil
}

func resourceNetworkFloatingIPV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	var updateOpts floatingips.UpdateOpts

	if d.HasChange("port_id") {
		updateOpts.PortID = d.Get("port_id").(string)
	}

	log.Printf("[DEBUG] Update Options: %#v", updateOpts)

	_, err = floatingips.Update(networkClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating floating IP: %s", err)
	}

	return resourceNetworkFloatingIPV2Read(d, meta)
}

func resourceNetworkFloatingIPV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     "DELETED",
		Refresh:    NetworkingFloatingIPV2StateRefreshFunc(networkingClient, d),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Floating IP: %s", err)
	}

	d.SetId("")
	return nil
}

func getNetworkID(d *schema.ResourceData, meta interface{}, networkName string) (string, error) {
	config := meta.(*Config)
	networkClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return "", fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	opts := networks.ListOpts{Name: networkName}
	pager := networks.List(networkClient, opts)
	networkID := ""

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}

		for _, n := range networkList {
			if n.Name == networkName {
				networkID = n.ID
				return false, nil
			}
		}

		return true, nil
	})

	return networkID, err
}

func getNetworkName(d *schema.ResourceData, meta interface{}, networkID string) (string, error) {
	config := meta.(*Config)
	networkClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return "", fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	opts := networks.ListOpts{ID: networkID}
	pager := networks.List(networkClient, opts)
	networkName := ""

	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		networkList, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}

		for _, n := range networkList {
			if n.ID == networkID {
				networkName = n.Name
				return false, nil
			}
		}

		return true, nil
	})

	return networkName, err
}

func NetworkingFloatingIPV2StateRefreshFunc(networkingClient *gophercloud.ServiceClient, d *schema.ResourceData) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		fId := d.Id()
		log.Printf("[DEBUG] Attempting to delete OpenStack Floating IP %s.\n", fId)

		n, err := floatingips.Get(networkingClient, fId).Extract()
		if err != nil {
			err = CheckDeleted(d, err, "OpenStack Floating IP")
			if err != nil {
				return n, "", err
			} else {
				log.Printf("[DEBUG] Successfully deleted OpenStack Floating IP %s", fId)
				return n, "DELETED", nil
			}
		}

		err = floatingips.Delete(networkingClient, fId).ExtractErr()
		if err != nil {
			err = CheckDeleted(d, err, "OpenStack Floating IP")
			if err != nil {
				return n, "", err
			} else {
				log.Printf("[DEBUG] Successfully deleted OpenStack Floating IP %s", fId)
				return n, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack Floating IP %s still active.\n", fId)
		return n, "ACTIVE", nil
	}
}
