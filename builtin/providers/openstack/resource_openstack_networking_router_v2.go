package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
)

func resourceNetworkingRouterV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingRouterV2Create,
		Read:   resourceNetworkingRouterV2Read,
		Update: resourceNetworkingRouterV2Update,
		Delete: resourceNetworkingRouterV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
			"external_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
		},
	}
}

func resourceNetworkingRouterV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := routers.CreateOpts{
		Name:     d.Get("name").(string),
		TenantID: d.Get("tenant_id").(string),
	}

	if asuRaw, ok := d.GetOk("admin_state_up"); ok {
		asu := asuRaw.(bool)
		createOpts.AdminStateUp = &asu
	}

	externalGateway := d.Get("external_gateway").(string)
	if externalGateway != "" {
		gatewayInfo := routers.GatewayInfo{
			NetworkID: externalGateway,
		}
		createOpts.GatewayInfo = &gatewayInfo
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := routers.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack Neutron router: %s", err)
	}
	log.Printf("[INFO] Router ID: %s", n.ID)

	log.Printf("[DEBUG] Waiting for OpenStack Neutron Router (%s) to become available", n.ID)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"BUILD", "PENDING_CREATE", "PENDING_UPDATE"},
		Target:     "ACTIVE",
		Refresh:    waitForRouterActive(networkingClient, n.ID),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	d.SetId(n.ID)

	return resourceNetworkingRouterV2Read(d, meta)
}

func resourceNetworkingRouterV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	n, err := routers.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		httpError, ok := err.(*gophercloud.UnexpectedResponseCodeError)
		if !ok {
			return fmt.Errorf("Error retrieving OpenStack Neutron Router: %s", err)
		}

		if httpError.Actual == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving OpenStack Neutron Router: %s", err)
	}

	log.Printf("[DEBUG] Retreived Router %s: %+v", d.Id(), n)

	d.Set("name", n.Name)
	d.Set("admin_state_up", n.AdminStateUp)
	d.Set("tenant_id", n.TenantID)
	d.Set("external_gateway", n.GatewayInfo.NetworkID)

	return nil
}

func resourceNetworkingRouterV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts routers.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}

	log.Printf("[DEBUG] Updating Router %s with options: %+v", d.Id(), updateOpts)

	_, err = routers.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack Neutron Router: %s", err)
	}

	return resourceNetworkingRouterV2Read(d, meta)
}

func resourceNetworkingRouterV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     "DELETED",
		Refresh:    waitForRouterDelete(networkingClient, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack Neutron Router: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForRouterActive(networkingClient *gophercloud.ServiceClient, routerId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r, err := routers.Get(networkingClient, routerId).Extract()
		if err != nil {
			return nil, r.Status, err
		}

		log.Printf("[DEBUG] OpenStack Neutron Router: %+v", r)
		return r, r.Status, nil
	}
}

func waitForRouterDelete(networkingClient *gophercloud.ServiceClient, routerId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack Router %s.\n", routerId)

		r, err := routers.Get(networkingClient, routerId).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router %s", routerId)
				return r, "DELETED", nil
			}
		}

		err = routers.Delete(networkingClient, routerId).ExtractErr()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return r, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack Router %s", routerId)
				return r, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack Router %s still active.\n", routerId)
		return r, "ACTIVE", nil
	}
}
