package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
)

func resourceLoadBalancerV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceLoadBalancerV2Create,
		Read:   resourceLoadBalancerV2Read,
		Update: resourceLoadBalancerV2Update,
		Delete: resourceLoadBalancerV2Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"vip_subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"vip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"vip_port_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"flavor": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"provider": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceLoadBalancerV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	adminStateUp := d.Get("admin_state_up").(bool)
	createOpts := loadbalancers.CreateOpts{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		VipSubnetID:  d.Get("vip_subnet_id").(string),
		TenantID:     d.Get("tenant_id").(string),
		VipAddress:   d.Get("vip_address").(string),
		AdminStateUp: &adminStateUp,
		Flavor:       d.Get("flavor").(string),
		Provider:     d.Get("provider").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	lb, err := loadbalancers.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LoadBalancer: %s", err)
	}
	log.Printf("[INFO] LoadBalancer ID: %s", lb.ID)

	log.Printf("[DEBUG] Waiting for Openstack LoadBalancer (%s) to become available.", lb.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForLoadBalancerActive(networkingClient, lb.ID),
		Timeout:    20 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(lb.ID)

	return resourceLoadBalancerV2Read(d, meta)
}

func resourceLoadBalancerV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	lb, err := loadbalancers.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LoadBalancerV2")
	}

	log.Printf("[DEBUG] Retrieved OpenStack LBaaSV2 LoadBalancer %s: %+v", d.Id(), lb)

	d.Set("name", lb.Name)
	d.Set("description", lb.Description)
	d.Set("vip_subnet_id", lb.VipSubnetID)
	d.Set("tenant_id", lb.TenantID)
	d.Set("vip_address", lb.VipAddress)
	d.Set("vip_port_id", lb.VipPortID)
	d.Set("admin_state_up", lb.AdminStateUp)
	d.Set("flavor", lb.Flavor)
	d.Set("provider", lb.Provider)

	return nil
}

func resourceLoadBalancerV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts loadbalancers.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		updateOpts.Description = d.Get("description").(string)
	}
	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}

	log.Printf("[DEBUG] Updating OpenStack LBaaSV2 LoadBalancer %s with options: %+v", d.Id(), updateOpts)

	_, err = loadbalancers.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LBaaSV2 LoadBalancer: %s", err)
	}

	return resourceLoadBalancerV2Read(d, meta)
}

func resourceLoadBalancerV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "PENDING_DELETE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForLoadBalancerDelete(networkingClient, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LBaaSV2 LoadBalancer: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForLoadBalancerActive(networkingClient *gophercloud.ServiceClient, lbID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		lb, err := loadbalancers.Get(networkingClient, lbID).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack LBaaSV2 LoadBalancer: %+v", lb)
		if lb.ProvisioningStatus == "ACTIVE" {
			return lb, "ACTIVE", nil
		}

		return lb, lb.ProvisioningStatus, nil
	}
}

func waitForLoadBalancerDelete(networkingClient *gophercloud.ServiceClient, lbID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack LBaaSV2 LoadBalancer %s", lbID)

		lb, err := loadbalancers.Get(networkingClient, lbID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 LoadBalancer %s", lbID)
				return lb, "DELETED", nil
			}
			return lb, "ACTIVE", err
		}

		log.Printf("[DEBUG] Openstack LoadBalancerV2: %+v", lb)
		err = loadbalancers.Delete(networkingClient, lbID).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 LoadBalancer %s", lbID)
				return lb, "DELETED", nil
			}

			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					log.Printf("[DEBUG] OpenStack LBaaSV2 LoadBalancer (%s) is still in use.", lbID)
					return lb, "ACTIVE", nil
				}
			}

			return lb, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack LBaaSV2 LoadBalancer (%s) still active.", lbID)
		return lb, "ACTIVE", nil
	}
}
