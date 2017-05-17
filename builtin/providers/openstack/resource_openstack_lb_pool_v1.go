package openstack

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas/members"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas/pools"
	"github.com/gophercloud/gophercloud/pagination"
)

func resourceLBPoolV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBPoolV1Create,
		Read:   resourceLBPoolV1Read,
		Update: resourceLBPoolV1Update,
		Delete: resourceLBPoolV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
			"lb_provider": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"member": &schema.Schema{
				Type:       schema.TypeSet,
				Deprecated: "Use openstack_lb_member_v1 instead. This attribute will be removed in a future version.",
				Optional:   true,
				Elem: &schema.Resource{
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
						"admin_state_up": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
							ForceNew: false,
						},
					},
				},
				Set: resourceLBMemberV1Hash,
			},
			"monitor_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceLBPoolV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := pools.CreateOpts{
		Name:     d.Get("name").(string),
		SubnetID: d.Get("subnet_id").(string),
		TenantID: d.Get("tenant_id").(string),
		Provider: d.Get("lb_provider").(string),
	}

	if v, ok := d.GetOk("protocol"); ok {
		protocol := resourceLBPoolV1DetermineProtocol(v.(string))
		createOpts.Protocol = protocol
	}

	if v, ok := d.GetOk("lb_method"); ok {
		lbMethod := resourceLBPoolV1DetermineLBMethod(v.(string))
		createOpts.LBMethod = lbMethod
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	p, err := pools.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB pool: %s", err)
	}
	log.Printf("[INFO] LB Pool ID: %s", p.ID)

	log.Printf("[DEBUG] Waiting for OpenStack LB pool (%s) to become available.", p.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForLBPoolActive(networkingClient, p.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(p.ID)

	if mIDs := resourcePoolMonitorIDsV1(d); mIDs != nil {
		for _, mID := range mIDs {
			_, err := pools.AssociateMonitor(networkingClient, p.ID, mID).Extract()
			if err != nil {
				return fmt.Errorf("Error associating monitor (%s) with OpenStack LB pool (%s): %s", mID, p.ID, err)
			}
		}
	}

	if memberOpts := resourcePoolMembersV1(d); memberOpts != nil {
		for _, memberOpt := range memberOpts {
			_, err := members.Create(networkingClient, memberOpt).Extract()
			if err != nil {
				return fmt.Errorf("Error creating OpenStack LB member: %s", err)
			}
		}
	}

	return resourceLBPoolV1Read(d, meta)
}

func resourceLBPoolV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	p, err := pools.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LB pool")
	}

	log.Printf("[DEBUG] Retrieved OpenStack LB Pool %s: %+v", d.Id(), p)

	d.Set("name", p.Name)
	d.Set("protocol", p.Protocol)
	d.Set("subnet_id", p.SubnetID)
	d.Set("lb_method", p.LBMethod)
	d.Set("lb_provider", p.Provider)
	d.Set("tenant_id", p.TenantID)
	d.Set("monitor_ids", p.MonitorIDs)
	d.Set("member_ids", p.MemberIDs)
	d.Set("region", GetRegion(d))

	return nil
}

func resourceLBPoolV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts pools.UpdateOpts
	// If either option changed, update both.
	// Gophercloud complains if one is empty.
	if d.HasChange("name") || d.HasChange("lb_method") {
		updateOpts.Name = d.Get("name").(string)

		lbMethod := resourceLBPoolV1DetermineLBMethod(d.Get("lb_method").(string))
		updateOpts.LBMethod = lbMethod
	}

	log.Printf("[DEBUG] Updating OpenStack LB Pool %s with options: %+v", d.Id(), updateOpts)

	_, err = pools.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB Pool: %s", err)
	}

	if d.HasChange("monitor_ids") {
		oldMIDsRaw, newMIDsRaw := d.GetChange("monitor_ids")
		oldMIDsSet, newMIDsSet := oldMIDsRaw.(*schema.Set), newMIDsRaw.(*schema.Set)
		monitorsToAdd := newMIDsSet.Difference(oldMIDsSet)
		monitorsToRemove := oldMIDsSet.Difference(newMIDsSet)

		log.Printf("[DEBUG] Monitors to add: %v", monitorsToAdd)

		log.Printf("[DEBUG] Monitors to remove: %v", monitorsToRemove)

		for _, m := range monitorsToAdd.List() {
			_, err := pools.AssociateMonitor(networkingClient, d.Id(), m.(string)).Extract()
			if err != nil {
				return fmt.Errorf("Error associating monitor (%s) with OpenStack server (%s): %s", m.(string), d.Id(), err)
			}
			log.Printf("[DEBUG] Associated monitor (%s) with pool (%s)", m.(string), d.Id())
		}

		for _, m := range monitorsToRemove.List() {
			_, err := pools.DisassociateMonitor(networkingClient, d.Id(), m.(string)).Extract()
			if err != nil {
				return fmt.Errorf("Error disassociating monitor (%s) from OpenStack server (%s): %s", m.(string), d.Id(), err)
			}
			log.Printf("[DEBUG] Disassociated monitor (%s) from pool (%s)", m.(string), d.Id())
		}
	}

	if d.HasChange("member") {
		oldMembersRaw, newMembersRaw := d.GetChange("member")
		oldMembersSet, newMembersSet := oldMembersRaw.(*schema.Set), newMembersRaw.(*schema.Set)
		membersToAdd := newMembersSet.Difference(oldMembersSet)
		membersToRemove := oldMembersSet.Difference(newMembersSet)

		log.Printf("[DEBUG] Members to add: %v", membersToAdd)

		log.Printf("[DEBUG] Members to remove: %v", membersToRemove)

		for _, m := range membersToRemove.List() {
			oldMember := resourcePoolMemberV1(d, m)
			listOpts := members.ListOpts{
				PoolID:       d.Id(),
				Address:      oldMember.Address,
				ProtocolPort: oldMember.ProtocolPort,
			}
			err = members.List(networkingClient, listOpts).EachPage(func(page pagination.Page) (bool, error) {
				extractedMembers, err := members.ExtractMembers(page)
				if err != nil {
					return false, err
				}
				for _, member := range extractedMembers {
					err := members.Delete(networkingClient, member.ID).ExtractErr()
					if err != nil {
						return false, fmt.Errorf("Error deleting member (%s) from OpenStack LB pool (%s): %s", member.ID, d.Id(), err)
					}
					log.Printf("[DEBUG] Deleted member (%s) from pool (%s)", member.ID, d.Id())
				}
				return true, nil
			})
		}

		for _, m := range membersToAdd.List() {
			createOpts := resourcePoolMemberV1(d, m)
			newMember, err := members.Create(networkingClient, createOpts).Extract()
			if err != nil {
				return fmt.Errorf("Error creating LB member: %s", err)
			}
			log.Printf("[DEBUG] Created member (%s) in OpenStack LB pool (%s)", newMember.ID, d.Id())
		}
	}

	return resourceLBPoolV1Read(d, meta)
}

func resourceLBPoolV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	// Make sure all monitors are disassociated first
	if v, ok := d.GetOk("monitor_ids"); ok {
		if monitorIDList, ok := v.([]interface{}); ok {
			for _, monitorID := range monitorIDList {
				mID := monitorID.(string)
				log.Printf("[DEBUG] Attempting to disassociate monitor %s from pool %s", mID, d.Id())
				if res := pools.DisassociateMonitor(networkingClient, d.Id(), mID); res.Err != nil {
					return fmt.Errorf("Error disassociating monitor %s from pool %s: %s", mID, d.Id(), err)
				}
			}
		}
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "PENDING_DELETE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForLBPoolDelete(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB Pool: %s", err)
	}

	d.SetId("")
	return nil
}

func resourcePoolMonitorIDsV1(d *schema.ResourceData) []string {
	mIDsRaw := d.Get("monitor_ids").(*schema.Set)
	mIDs := make([]string, mIDsRaw.Len())
	for i, raw := range mIDsRaw.List() {
		mIDs[i] = raw.(string)
	}
	return mIDs
}

func resourcePoolMembersV1(d *schema.ResourceData) []members.CreateOpts {
	memberOptsRaw := d.Get("member").(*schema.Set)
	memberOpts := make([]members.CreateOpts, memberOptsRaw.Len())
	for i, raw := range memberOptsRaw.List() {
		rawMap := raw.(map[string]interface{})
		memberOpts[i] = members.CreateOpts{
			TenantID:     rawMap["tenant_id"].(string),
			Address:      rawMap["address"].(string),
			ProtocolPort: rawMap["port"].(int),
			PoolID:       d.Id(),
		}
	}
	return memberOpts
}

func resourcePoolMemberV1(d *schema.ResourceData, raw interface{}) members.CreateOpts {
	rawMap := raw.(map[string]interface{})
	return members.CreateOpts{
		TenantID:     rawMap["tenant_id"].(string),
		Address:      rawMap["address"].(string),
		ProtocolPort: rawMap["port"].(int),
		PoolID:       d.Id(),
	}
}

func resourceLBMemberV1Hash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["region"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["tenant_id"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["address"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))

	return hashcode.String(buf.String())
}

func resourceLBPoolV1DetermineProtocol(v string) pools.LBProtocol {
	var protocol pools.LBProtocol
	switch v {
	case "TCP":
		protocol = pools.ProtocolTCP
	case "HTTP":
		protocol = pools.ProtocolHTTP
	case "HTTPS":
		protocol = pools.ProtocolHTTPS
	}

	return protocol
}

func resourceLBPoolV1DetermineLBMethod(v string) pools.LBMethod {
	var lbMethod pools.LBMethod
	switch v {
	case "ROUND_ROBIN":
		lbMethod = pools.LBMethodRoundRobin
	case "LEAST_CONNECTIONS":
		lbMethod = pools.LBMethodLeastConnections
	}

	return lbMethod
}

func waitForLBPoolActive(networkingClient *gophercloud.ServiceClient, poolId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		p, err := pools.Get(networkingClient, poolId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack LB Pool: %+v", p)
		if p.Status == "ACTIVE" {
			return p, "ACTIVE", nil
		}

		return p, p.Status, nil
	}
}

func waitForLBPoolDelete(networkingClient *gophercloud.ServiceClient, poolId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack LB Pool %s", poolId)

		p, err := pools.Get(networkingClient, poolId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LB Pool %s", poolId)
				return p, "DELETED", nil
			}
			return p, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack LB Pool: %+v", p)
		err = pools.Delete(networkingClient, poolId).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LB Pool %s", poolId)
				return p, "DELETED", nil
			}
			return p, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack LB Pool %s still active.", poolId)
		return p, "ACTIVE", nil
	}

}
