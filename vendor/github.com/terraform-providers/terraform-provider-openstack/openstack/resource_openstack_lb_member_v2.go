package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

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
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
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
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
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

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	// Wait for LB to become active before continuing
	poolID := d.Get("pool_id").(string)
	timeout := d.Timeout(schema.TimeoutCreate)
	err = waitForLBV2viaPool(networkingClient, poolID, "ACTIVE", timeout)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Attempting to create member")
	var member *pools.Member
	err = resource.Retry(timeout, func() *resource.RetryError {
		member, err = pools.CreateMember(networkingClient, poolID, createOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating member: %s", err)
	}

	// Wait for LB to become ACTIVE again
	err = waitForLBV2viaPool(networkingClient, poolID, "ACTIVE", timeout)
	if err != nil {
		return err
	}

	d.SetId(member.ID)

	return resourceMemberV2Read(d, meta)
}

func resourceMemberV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	member, err := pools.GetMember(networkingClient, d.Get("pool_id").(string), d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "member")
	}

	log.Printf("[DEBUG] Retrieved member %s: %#v", d.Id(), member)

	d.Set("name", member.Name)
	d.Set("weight", member.Weight)
	d.Set("admin_state_up", member.AdminStateUp)
	d.Set("tenant_id", member.TenantID)
	d.Set("subnet_id", member.SubnetID)
	d.Set("address", member.Address)
	d.Set("protocol_port", member.ProtocolPort)
	d.Set("id", member.ID)
	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceMemberV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
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

	// Wait for LB to become active before continuing
	poolID := d.Get("pool_id").(string)
	timeout := d.Timeout(schema.TimeoutUpdate)
	err = waitForLBV2viaPool(networkingClient, poolID, "ACTIVE", timeout)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updating member %s with options: %#v", d.Id(), updateOpts)
	err = resource.Retry(timeout, func() *resource.RetryError {
		_, err = pools.UpdateMember(networkingClient, poolID, d.Id(), updateOpts).Extract()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Unable to update member %s: %s", d.Id(), err)
	}

	err = waitForLBV2viaPool(networkingClient, poolID, "ACTIVE", timeout)
	if err != nil {
		return err
	}

	return resourceMemberV2Read(d, meta)
}

func resourceMemberV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	// Wait for Pool to become active before continuing
	poolID := d.Get("pool_id").(string)
	timeout := d.Timeout(schema.TimeoutDelete)
	err = waitForLBV2viaPool(networkingClient, poolID, "ACTIVE", timeout)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Attempting to delete member %s", d.Id())
	err = resource.Retry(timeout, func() *resource.RetryError {
		err = pools.DeleteMember(networkingClient, poolID, d.Id()).ExtractErr()
		if err != nil {
			return checkForRetryableError(err)
		}
		return nil
	})

	// Wait for LB to become ACTIVE
	err = waitForLBV2viaPool(networkingClient, poolID, "ACTIVE", timeout)
	if err != nil {
		return err
	}

	return nil
}
