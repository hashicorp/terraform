package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas/vips"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceLBVipV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBVipV1Create,
		Read:   resourceLBVipV1Read,
		Update: resourceLBVipV1Update,
		Delete: resourceLBVipV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"protocol": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"pool_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"persistence": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},
			"conn_limit": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"port_id": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: false,
			},
			"floating_ip": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"admin_state_up": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
		},
	}
}

func resourceLBVipV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := vips.CreateOpts{
		Name:         d.Get("name").(string),
		SubnetID:     d.Get("subnet_id").(string),
		Protocol:     d.Get("protocol").(string),
		ProtocolPort: d.Get("port").(int),
		PoolID:       d.Get("pool_id").(string),
		TenantID:     d.Get("tenant_id").(string),
		Address:      d.Get("address").(string),
		Description:  d.Get("description").(string),
		Persistence:  resourceVipPersistenceV1(d),
		ConnLimit:    gophercloud.MaybeInt(d.Get("conn_limit").(int)),
	}

	asu := d.Get("admin_state_up").(bool)
	createOpts.AdminStateUp = &asu

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	p, err := vips.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB VIP: %s", err)
	}
	log.Printf("[INFO] LB VIP ID: %s", p.ID)

	log.Printf("[DEBUG] Waiting for OpenStack LB VIP (%s) to become available.", p.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForLBVIPActive(networkingClient, p.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	floatingIP := d.Get("floating_ip").(string)
	if floatingIP != "" {
		lbVipV1AssignFloatingIP(floatingIP, p.PortID, networkingClient)
	}

	d.SetId(p.ID)

	return resourceLBVipV1Read(d, meta)
}

func resourceLBVipV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	p, err := vips.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LB VIP")
	}

	log.Printf("[DEBUG] Retrieved OpenStack LB VIP %s: %+v", d.Id(), p)

	d.Set("name", p.Name)
	d.Set("subnet_id", p.SubnetID)
	d.Set("protocol", p.Protocol)
	d.Set("port", p.ProtocolPort)
	d.Set("pool_id", p.PoolID)
	d.Set("port_id", p.PortID)
	d.Set("tenant_id", p.TenantID)
	d.Set("address", p.Address)
	d.Set("description", p.Description)
	d.Set("conn_limit", p.ConnLimit)
	d.Set("admin_state_up", p.AdminStateUp)

	// Set the persistence method being used
	persistence := make(map[string]interface{})
	if p.Persistence.Type != "" {
		persistence["type"] = p.Persistence.Type
	}
	if p.Persistence.CookieName != "" {
		persistence["cookie_name"] = p.Persistence.CookieName
	}
	d.Set("persistence", persistence)

	d.Set("region", GetRegion(d, config))

	return nil
}

func resourceLBVipV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts vips.UpdateOpts
	if d.HasChange("name") {
		v := d.Get("name").(string)
		updateOpts.Name = &v
	}

	if d.HasChange("pool_id") {
		v := d.Get("pool_id").(string)
		updateOpts.PoolID = &v
	}

	if d.HasChange("description") {
		v := d.Get("description").(string)
		updateOpts.Description = &v
	}

	if d.HasChange("conn_limit") {
		updateOpts.ConnLimit = gophercloud.MaybeInt(d.Get("conn_limit").(int))
	}

	if d.HasChange("floating_ip") {
		portID := d.Get("port_id").(string)

		// Searching for a floating IP assigned to the VIP
		listOpts := floatingips.ListOpts{
			PortID: portID,
		}
		page, err := floatingips.List(networkingClient, listOpts).AllPages()
		if err != nil {
			return err
		}

		fips, err := floatingips.ExtractFloatingIPs(page)
		if err != nil {
			return err
		}

		// If a floating IP is found we unassign it
		if len(fips) == 1 {
			portID := ""
			updateOpts := floatingips.UpdateOpts{
				PortID: &portID,
			}
			if err = floatingips.Update(networkingClient, fips[0].ID, updateOpts).Err; err != nil {
				return err
			}
		}

		// Assign the updated floating IP
		floatingIP := d.Get("floating_ip").(string)
		if floatingIP != "" {
			lbVipV1AssignFloatingIP(floatingIP, portID, networkingClient)
		}
	}

	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}

	// Persistence has to be included, even if it hasn't changed.
	updateOpts.Persistence = resourceVipPersistenceV1(d)

	log.Printf("[DEBUG] Updating OpenStack LB VIP %s with options: %+v", d.Id(), updateOpts)

	_, err = vips.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB VIP: %s", err)
	}

	return resourceLBVipV1Read(d, meta)
}

func resourceLBVipV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "PENDING_DELETE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForLBVIPDelete(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB VIP: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceVipPersistenceV1(d *schema.ResourceData) *vips.SessionPersistence {
	rawP := d.Get("persistence").(interface{})
	rawMap := rawP.(map[string]interface{})
	if len(rawMap) != 0 {
		p := vips.SessionPersistence{}
		if t, ok := rawMap["type"]; ok {
			p.Type = t.(string)
		}
		if c, ok := rawMap["cookie_name"]; ok {
			p.CookieName = c.(string)
		}
		return &p
	}
	return nil
}

func lbVipV1AssignFloatingIP(floatingIP, portID string, networkingClient *gophercloud.ServiceClient) error {
	log.Printf("[DEBUG] Assigning floating IP %s to VIP %s", floatingIP, portID)

	listOpts := floatingips.ListOpts{
		FloatingIP: floatingIP,
	}
	page, err := floatingips.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return err
	}

	fips, err := floatingips.ExtractFloatingIPs(page)
	if err != nil {
		return err
	}
	if len(fips) != 1 {
		return fmt.Errorf("Unable to retrieve floating IP '%s'", floatingIP)
	}

	updateOpts := floatingips.UpdateOpts{
		PortID: &portID,
	}
	if err = floatingips.Update(networkingClient, fips[0].ID, updateOpts).Err; err != nil {
		return err
	}

	return nil
}

func waitForLBVIPActive(networkingClient *gophercloud.ServiceClient, vipId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		p, err := vips.Get(networkingClient, vipId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack LB VIP: %+v", p)
		if p.Status == "ACTIVE" {
			return p, "ACTIVE", nil
		}

		return p, p.Status, nil
	}
}

func waitForLBVIPDelete(networkingClient *gophercloud.ServiceClient, vipId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack LB VIP %s", vipId)

		p, err := vips.Get(networkingClient, vipId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LB VIP %s", vipId)
				return p, "DELETED", nil
			}
			return p, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack LB VIP: %+v", p)
		err = vips.Delete(networkingClient, vipId).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LB VIP %s", vipId)
				return p, "DELETED", nil
			}
			return p, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack LB VIP %s still active.", vipId)
		return p, "ACTIVE", nil
	}

}
