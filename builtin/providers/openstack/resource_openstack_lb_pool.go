package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jrperritt/terraform/helper/hashcode"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/pools"
)

func resourceLBPool() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBPoolCreate,
		Read:   resourceLBPoolRead,
		Update: resourceLBPoolUpdate,
		Delete: resourceLBPoolDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"lb_method": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"monitor_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceLBPoolCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	createOpts := pools.CreateOpts{
		Name:     d.Get("name").(string),
		Protocol: d.Get("protocol").(string),
		SubnetID: d.Get("subnet_id").(string),
		LBMethod: d.Get("lb_method").(string),
		TenantID: d.Get("tenant_id").(string),
	}

	log.Printf("[INFO] Requesting lb pool creation")
	p, err := pools.Create(osClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB pool: %s", err)
	}
	log.Printf("[INFO] LB Pool ID: %s", p.ID)

	d.SetId(p.ID)

	if mIDs := resourcePoolMonitorIDs(d); mIDs != nil {
		for _, mID := range mIDs {
			_, err := pools.AssociateMonitor(osClient, p.ID, mID).Extract()
			if err != nil {
				return fmt.Errorf("Error associating monitor (%s) with OpenStack LB pool (%s): %s", mID, p.ID, err)
			}
		}
	}

	return resourceLBPoolRead(d, meta)
}

func resourceLBPoolRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	p, err := pools.Get(osClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack LB Pool: %s", err)
	}

	log.Printf("[DEBUG] Retreived OpenStack LB Pool %s: %+v", d.Id(), p)

	d.Set("name", p.Name)
	d.Set("protocol", p.Protocol)
	d.Set("subnet_id", p.SubnetID)
	d.Set("lb_method", p.LBMethod)

	if _, exists := d.GetOk("tenant_id"); exists {
		if d.HasChange("tenant_id") {
			d.Set("tenant_id", p.TenantID)
		}
	} else {
		d.Set("tenant_id", "")
	}

	d.Set("monitor_ids", p.MonitorIDs)

	return nil
}

func resourceLBPoolUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	var updateOpts pools.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("lb_method") {
		updateOpts.LBMethod = d.Get("lb_method").(string)
	}

	log.Printf("[DEBUG] Updating OpenStack LB Pool %s with options: %+v", d.Id(), updateOpts)

	_, err := pools.Update(osClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB Pool: %s", err)
	}

	if d.HasChange("monitor_ids") {
		oldMIDsRaw, newMIDsRaw := d.GetChange("security_groups")
		oldMIDsSet, newMIDsSet := oldMIDsRaw.(*schema.Set), newMIDsRaw.(*schema.Set)
		monitorsToAdd := newMIDsSet.Difference(oldMIDsSet)
		monitorsToRemove := oldMIDsSet.Difference(newMIDsSet)

		log.Printf("[DEBUG] Monitors to add: %v", monitorsToAdd)

		log.Printf("[DEBUG] Monitors to remove: %v", monitorsToRemove)

		for _, m := range monitorsToAdd.List() {
			_, err := pools.AssociateMonitor(osClient, d.Id(), m.(string)).Extract()
			if err != nil {
				return fmt.Errorf("Error associating monitor (%s) with OpenStack server (%s): %s", m.(string), d.Id(), err)
			}
			log.Printf("[DEBUG] Associated monitor (%s) with pool (%s)", m.(string), d.Id())
		}

		for _, m := range monitorsToRemove.List() {
			_, err := pools.DisassociateMonitor(osClient, d.Id(), m.(string)).Extract()
			if err != nil {
				return fmt.Errorf("Error disassociating monitor (%s) from OpenStack server (%s): %s", m.(string), d.Id(), err)
			}
			log.Printf("[DEBUG] Disassociated monitor (%s) from pool (%s)", m.(string), d.Id())
		}
	}

	return resourceLBPoolRead(d, meta)
}

func resourceLBPoolDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	err := pools.Delete(osClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB Pool: %s", err)
	}

	d.SetId("")
	return nil
}

func resourcePoolMonitorIDs(d *schema.ResourceData) []string {
	mIDsRaw := d.Get("monitor_ids").(*schema.Set)
	mIDs := make([]string, mIDsRaw.Len())
	for i, raw := range mIDsRaw.List() {
		mIDs[i] = raw.(string)
	}
	return mIDs
}
