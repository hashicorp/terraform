package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/members"
)

func resourceLBMemberV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBMemberV1Create,
		Read:   resourceLBMemberV1Read,
		Update: resourceLBMemberV1Update,
		Delete: resourceLBMemberV1Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"weight": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
		},
	}
}

func resourceLBMemberV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := members.CreateOpts{
		TenantID:     d.Get("tenant_id").(string),
		PoolID:       d.Get("pool_id").(string),
		Address:      d.Get("address").(string),
		ProtocolPort: d.Get("port").(int),
	}

	log.Printf("[DEBUG] OpenStack LB Member Create Options: %#v", createOpts)
	m, err := members.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB member: %s", err)
	}
	log.Printf("[INFO] LB member ID: %s", m.ID)

	log.Printf("[DEBUG] Waiting for OpenStack LB member (%s) to become available.", m.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE", "INACTIVE"},
		Refresh:    waitForLBMemberActive(networkingClient, m.ID),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(m.ID)

	// Due to the way Gophercloud is currently set up, AdminStateUp must be set post-create
	updateOpts := members.UpdateOpts{
		AdminStateUp: d.Get("admin_state_up").(bool),
	}

	log.Printf("[DEBUG] OpenStack LB Member Update Options: %#v", createOpts)
	m, err = members.Update(networkingClient, m.ID, updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB member: %s", err)
	}

	return resourceLBMemberV1Read(d, meta)
}

func resourceLBMemberV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	m, err := members.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LB member")
	}

	log.Printf("[DEBUG] Retreived OpenStack LB member %s: %+v", d.Id(), m)

	d.Set("weight", m.Weight)
	d.Set("admin_state_up", m.AdminStateUp)

	return nil
}

func resourceLBMemberV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts members.UpdateOpts
	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = asu
	}

	log.Printf("[DEBUG] Updating LB member %s with options: %+v", d.Id(), updateOpts)

	_, err = members.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB member: %s", err)
	}

	return resourceLBMemberV1Read(d, meta)
}

func resourceLBMemberV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	err = members.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		CheckDeleted(d, err, "LB member")
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "PENDING_DELETE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForLBMemberDelete(networkingClient, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB member: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForLBMemberActive(networkingClient *gophercloud.ServiceClient, memberId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		m, err := members.Get(networkingClient, memberId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack LB member: %+v", m)
		if m.Status == "ACTIVE" {
			return m, "ACTIVE", nil
		}

		return m, m.Status, nil
	}
}

func waitForLBMemberDelete(networkingClient *gophercloud.ServiceClient, memberId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack LB member %s", memberId)

		m, err := members.Get(networkingClient, memberId).Extract()
		if err != nil {
			errCode, ok := err.(*gophercloud.UnexpectedResponseCodeError)
			if !ok {
				return m, "ACTIVE", err
			}
			if errCode.Actual == 404 {
				log.Printf("[DEBUG] Successfully deleted OpenStack LB member %s", memberId)
				return m, "DELETED", nil
			}
		}

		log.Printf("[DEBUG] OpenStack LB member %s still active.", memberId)
		return m, "ACTIVE", nil
	}

}
