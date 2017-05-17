package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/pools"
)

func resourceMemberV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceMemberV2Create,
		Read:   resourceMemberV2Read,
		Update: resourceMemberV2Update,
		Delete: resourceMemberV2Delete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

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

			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"protocol_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"weight": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 1 {
						errors = append(errors, fmt.Errorf(
							"Only numbers greater than 0 are supported values for 'weight'"))
					}
					return
				},
			},

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceMemberV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	adminStateUp := d.Get("admin_state_up").(bool)
	createOpts := pools.CreateMemberOpts{
		Name:         d.Get("name").(string),
		TenantID:     d.Get("tenant_id").(string),
		Address:      d.Get("address").(string),
		ProtocolPort: d.Get("protocol_port").(int),
		Weight:       d.Get("weight").(int),
		AdminStateUp: &adminStateUp,
	}

	// Must omit if not set
	if v, ok := d.GetOk("subnet_id"); ok {
		createOpts.SubnetID = v.(string)
	}

	poolID := d.Get("pool_id").(string)

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	var member *pools.Member
	err = resource.Retry(10*time.Minute, func() *resource.RetryError {
		var err error
		log.Printf("[DEBUG] Attempting to create LBaaSV2 member")
		member, err = pools.CreateMember(networkingClient, poolID, createOpts).Extract()
		if err != nil {
			switch errCode := err.(type) {
			case gophercloud.ErrDefault500:
				log.Printf("[DEBUG] OpenStack LBaaSV2 member is still creating.")
				return resource.RetryableError(err)
			case gophercloud.ErrUnexpectedResponseCode:
				if errCode.Actual == 409 {
					log.Printf("[DEBUG] OpenStack LBaaSV2 member is still creating.")
					return resource.RetryableError(err)
				}

			default:
				return resource.NonRetryableError(err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating OpenStack LBaaSV2 member: %s", err)
	}
	log.Printf("[INFO] member ID: %s", member.ID)

	log.Printf("[DEBUG] Waiting for Openstack LBaaSV2 member (%s) to become available.", member.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForMemberActive(networkingClient, poolID, member.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(member.ID)

	return resourceMemberV2Read(d, meta)
}

func resourceMemberV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	member, err := pools.GetMember(networkingClient, d.Get("pool_id").(string), d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LBV2 Member")
	}

	log.Printf("[DEBUG] Retrieved OpenStack LBaaSV2 Member %s: %+v", d.Id(), member)

	d.Set("name", member.Name)
	d.Set("weight", member.Weight)
	d.Set("admin_state_up", member.AdminStateUp)
	d.Set("tenant_id", member.TenantID)
	d.Set("subnet_id", member.SubnetID)
	d.Set("address", member.Address)
	d.Set("protocol_port", member.ProtocolPort)
	d.Set("id", member.ID)

	return nil
}

func resourceMemberV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts pools.UpdateMemberOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("weight") {
		updateOpts.Weight = d.Get("weight").(int)
	}
	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}

	log.Printf("[DEBUG] Updating OpenStack LBaaSV2 Member %s with options: %+v", d.Id(), updateOpts)

	_, err = pools.UpdateMember(networkingClient, d.Get("pool_id").(string), d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LBaaSV2 Member: %s", err)
	}

	return resourceMemberV2Read(d, meta)
}

func resourceMemberV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "PENDING_DELETE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForMemberDelete(networkingClient, d.Get("pool_id").(string), d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LBaaSV2 Member: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForMemberActive(networkingClient *gophercloud.ServiceClient, poolID string, memberID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		member, err := pools.GetMember(networkingClient, poolID, memberID).Extract()
		if err != nil {
			return nil, "", err
		}

		// The member resource has no Status attribute, so a successful Get is the best we can do
		log.Printf("[DEBUG] OpenStack LBaaSV2 Member: %+v", member)
		return member, "ACTIVE", nil
	}
}

func waitForMemberDelete(networkingClient *gophercloud.ServiceClient, poolID string, memberID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack LBaaSV2 Member %s", memberID)

		member, err := pools.GetMember(networkingClient, poolID, memberID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 Member %s", memberID)
				return member, "DELETED", nil
			}
			return member, "ACTIVE", err
		}

		log.Printf("[DEBUG] Openstack LBaaSV2 Member: %+v", member)
		err = pools.DeleteMember(networkingClient, poolID, memberID).ExtractErr()
		if err != nil {
			switch errCode := err.(type) {
			case gophercloud.ErrDefault404:
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 Member %s", memberID)
				return member, "DELETED", nil
			case gophercloud.ErrDefault500:
				log.Printf("[DEBUG] OpenStack LBaaSV2 Member (%s) is still in use.", memberID)
				return member, "PENDING_DELETE", nil
			case gophercloud.ErrUnexpectedResponseCode:
				if errCode.Actual == 409 {
					log.Printf("[DEBUG] OpenStack LBaaSV2 Member (%s) is still in use.", memberID)
					return member, "PENDING_DELETE", nil
				}

			default:
				return member, "ACTIVE", err
			}
		}

		log.Printf("[DEBUG] OpenStack LBaaSV2 Member %s still active.", memberID)
		return member, "ACTIVE", nil
	}
}
