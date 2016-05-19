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
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"pool": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_POOL_NAME", nil),
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"port_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceNetworkFloatingIPV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	networkID, err := getNetworkID(d, meta, d.Get("pool").(string))
	if err != nil {
		return fmt.Errorf("Error retrieving floating IP pool name: %s", err)
	}

	createOpts := floatingips.CreateOpts{
		FloatingNetworkID: networkID,
		PortID:            d.Get("port_id").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	floatingIP, err := floatingips.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error allocating floating IP: %s", err)
	}

	log.Printf("[DEBUG] Waiting for OpenStack Neutron Floating IP (%s) to become available.", floatingIP.ID)

	stateConf := &resource.StateChangeConf{
		Target:     []string{"ACTIVE"},
		Refresh:    waitForFloatingIPActive(networkingClient, floatingIP.ID),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	d.SetId(floatingIP.ID)

	return resourceNetworkFloatingIPV2Read(d, meta)
}

func resourceNetworkFloatingIPV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	floatingIP, err := floatingips.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "floating IP")
	}

	d.Set("address", floatingIP.FloatingIP)

	return nil
}

func resourceNetworkFloatingIPV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	if d.HasChange("port_id") {
		var updateOpts floatingips.UpdateOpts
		_, n := d.GetChange("port_id")
		newPortID := n.(string)
		updateOpts.PortID = newPortID

		log.Printf("[DEBUG] Update Options: %#v", updateOpts)

		_, err = floatingips.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating floating IP: %s", err)
		}

	}

	return resourceNetworkFloatingIPV2Read(d, meta)
}

func resourceNetworkFloatingIPV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	err = floatingips.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return CheckDeleted(d, err, "floating IP")
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForFloatingIPDelete(networkingClient, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
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
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return "", fmt.Errorf("Error creating OpenStack network client: %s", err)
	}

	opts := networks.ListOpts{Name: networkName}
	pages, err := networks.List(networkingClient, opts).AllPages()
	if err != nil {
		return "", err
	}

	networks, err := networks.ExtractNetworks(pages)
	if err != nil {
		return "", err
	}

	for _, n := range networks {
		if n.Name == networkName {
			log.Printf("[DEBUG] Found Network ID for Network %s: %s", networkName, n.ID)
			return n.ID, nil
		}
	}

	return "", fmt.Errorf("Unable to Network ID for Network %s", networkName)
}

func waitForFloatingIPActive(networkingClient *gophercloud.ServiceClient, fId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		f, err := floatingips.Get(networkingClient, fId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack Neutron Floating IP: %+v", f)
		if f.Status == "DOWN" || f.Status == "ACTIVE" {
			return f, "ACTIVE", nil
		}

		return f, "", nil
	}
}

func waitForFloatingIPDelete(networkingClient *gophercloud.ServiceClient, fId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack Floating IP %s.\n", fId)

		f, err := floatingips.Get(networkingClient, fId).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return f, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Floating IP %s", fId)
				return f, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack Floating IP %s still active.\n", fId)
		return f, "ACTIVE", nil
	}
}
