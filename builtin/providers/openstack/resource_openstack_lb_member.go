package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/members"
)

func resourceLBMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBMemberCreate,
		Read:   resourceLBMemberRead,
		Update: resourceLBMemberUpdate,
		Delete: resourceLBMemberDelete,

		Schema: map[string]*schema.Schema{
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

			"pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

func resourceLBMemberCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	createOpts := members.CreateOpts{
		//TenantID:     d.Get("tenant_id").(string),
		Address:      d.Get("address").(string),
		ProtocolPort: d.Get("port").(int),
		PoolID:       d.Get("pool_id").(string),
	}

	log.Printf("[INFO] Requesting lb member creation")
	p, err := members.Create(osClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB member: %s", err)
	}
	log.Printf("[INFO] LB Member ID: %s", p.ID)

	d.SetId(p.ID)

	return resourceLBMemberRead(d, meta)
}

func resourceLBMemberRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	p, err := members.Get(osClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack LB Member: %s", err)
	}

	log.Printf("[DEBUG] Retreived OpenStack LB Member %s: %+v", d.Id(), p)

	d.Set("address", p.Address)
	d.Set("port", p.ProtocolPort)
	d.Set("pool_id", p.PoolID)

	if _, exists := d.GetOk("tenant_id"); exists {
		if d.HasChange("tenant_id") {
			d.Set("tenant_id", p.TenantID)
		}
	} else {
		d.Set("tenant_id", "")
	}

	return nil
}

func resourceLBMemberUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	var updateOpts members.UpdateOpts
	if d.HasChange("admin_state_up") {
		updateOpts.AdminStateUp = d.Get("admin_state_up").(bool)
	}

	log.Printf("[DEBUG] Updating OpenStack LB Member %s with options: %+v", d.Id(), updateOpts)

	_, err := members.Update(osClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB Member: %s", err)
	}

	return resourceLBMemberRead(d, meta)
}

func resourceLBMemberDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	err := members.Delete(osClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB Member: %s", err)
	}

	d.SetId("")
	return nil
}
