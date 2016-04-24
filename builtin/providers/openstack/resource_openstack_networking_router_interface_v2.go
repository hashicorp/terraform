package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
)

func resourceNetworkingRouterInterfaceV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingRouterInterfaceV2Create,
		Read:   resourceNetworkingRouterInterfaceV2Read,
		Delete: resourceNetworkingRouterInterfaceV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"router_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"port_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNetworkingRouterInterfaceV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := routers.InterfaceOpts{
		SubnetID: d.Get("subnet_id").(string),
		PortID:   d.Get("port_id").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := routers.AddInterface(networkingClient, d.Get("router_id").(string), createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron router interface: %s", err)
	}
	log.Printf("[INFO] Router interface Port ID: %s", n.PortID)

	log.Printf("[DEBUG] Waiting for Router Interface (%s) to become available", n.PortID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD", "PENDING_CREATE", "PENDING_UPDATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForRouterInterfaceActive(networkingClient, n.PortID),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	d.SetId(n.PortID)

	return resourceNetworkingRouterInterfaceV2Read(d, meta)
}

func resourceNetworkingRouterInterfaceV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	n, err := ports.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		httpError, ok := err.(*gophercloud.UnexpectedResponseCodeError)
		if !ok {
			return fmt.Errorf("Error retrieving OpenStack Neutron Router Interface: %s", err)
		}

		if httpError.Actual == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving OpenStack Neutron Router Interface: %s", err)
	}

	log.Printf("[DEBUG] Retreived Router Interface %s: %+v", d.Id(), n)

	return nil
}

func resourceNetworkingRouterInterfaceV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForRouterInterfaceDelete(networkingClient, d),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Router Interface: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForRouterInterfaceActive(networkingClient *gophercloud.ServiceClient, rId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r, err := ports.Get(networkingClient, rId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack Neutron Router Interface: %+v", r)
		return r, r.Status, nil
	}
}

func waitForRouterInterfaceDelete(networkingClient *gophercloud.ServiceClient, d *schema.ResourceData) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		routerId := d.Get("router_id").(string)
		routerInterfaceId := d.Id()

		log.Printf("[DEBUG] Attempting to delete OpenStack Router Interface %s.\n", routerInterfaceId)

		removeOpts := routers.InterfaceOpts{
			SubnetID: d.Get("subnet_id").(string),
			PortID:   d.Get("port_id").(string),
		}

		r, err := ports.Get(networkingClient, routerInterfaceId).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router Interface %s", routerInterfaceId)
				return r, "DELETED", nil
			}
		}

		_, err = routers.RemoveInterface(networkingClient, routerId, removeOpts).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router Interface %s", routerInterfaceId)
				return r, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack Router Interface %s still active.\n", routerInterfaceId)
		return r, "ACTIVE", nil
	}
}
