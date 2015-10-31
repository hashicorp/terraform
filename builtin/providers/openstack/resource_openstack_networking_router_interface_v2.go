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
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},
			"router_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
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
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := routers.AddInterface(networkingClient, d.Get("router_id").(string), createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron router interface: %s", err)
	}
	log.Printf("[INFO] Router interface Port ID: %s", n.PortID)

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
		Target:     "DELETED",
		Refresh:    NetworkingRouterInterfaceV2StateRefreshFunc(networkingClient, d),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Router Interface: %s", err)
	}

	d.SetId("")
	return nil
}

func NetworkingRouterInterfaceV2StateRefreshFunc(networkingClient *gophercloud.ServiceClient, d *schema.ResourceData) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		routerId := d.Get("router_id").(string)
		routerInterfaceId := d.Id()

		log.Printf("[DEBUG] Attempting to delete OpenStack Router Interface %s.\n", routerInterfaceId)

		removeOpts := routers.InterfaceOpts{
			SubnetID: d.Get("subnet_id").(string),
		}

		r, err := ports.Get(networkingClient, routerInterfaceId).Extract()
		if err != nil {
			err = CheckDeleted(d, err, "OpenStack Router Interface")
			if err != nil {
				return r, "", err
			} else {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router Interface %s", routerInterfaceId)
				return r, "DELETED", nil
			}
		}

		_, err = routers.RemoveInterface(networkingClient, routerId, removeOpts).Extract()
		if err != nil {
			err = CheckDeleted(d, err, "OpenStack Router Interface")
			if err != nil {
				return r, "", err
			} else {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router Interface %s", routerInterfaceId)
				return r, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack Router Interface %s still active.\n", routerInterfaceId)
		return r, "ACTIVE", nil
	}
}
