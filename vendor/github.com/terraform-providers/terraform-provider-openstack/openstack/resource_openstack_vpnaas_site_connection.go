package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/vpnaas/siteconnections"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceSiteConnectionV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceSiteConnectionV2Create,
		Read:   resourceSiteConnectionV2Read,
		Update: resourceSiteConnectionV2Update,
		Delete: resourceSiteConnectionV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
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
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ikepolicy_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"peer_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"peer_address": {
				Type:     schema.TypeString,
				Required: true,
			},
			"peer_ep_group_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"local_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"vpnservice_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"local_ep_group_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ipsecpolicy_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"admin_state_up": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"psk": {
				Type:     schema.TypeString,
				Required: true,
			},
			"initiator": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"mtu": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"peer_cidrs": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"dpd": {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeString,
							Computed: true,
							Optional: true,
						},
						"timeout": {
							Type:     schema.TypeInt,
							Computed: true,
							Optional: true,
						},
						"interval": {
							Type:     schema.TypeInt,
							Computed: true,
							Optional: true,
						},
					},
				},
			},
			"value_specs": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSiteConnectionV2Create(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var createOpts siteconnections.CreateOptsBuilder

	dpd := resourceSiteConnectionV2DPDCreateOpts(d.Get("dpd").(*schema.Set))

	v := d.Get("peer_cidrs").([]interface{})
	peerCidrs := make([]string, len(v))
	for i, v := range v {
		peerCidrs[i] = v.(string)
	}

	adminStateUp := d.Get("admin_state_up").(bool)
	initiator := resourceSiteConnectionV2Initiator(d.Get("initiator").(string))

	createOpts = SiteConnectionCreateOpts{
		siteconnections.CreateOpts{
			Name:           d.Get("name").(string),
			Description:    d.Get("description").(string),
			AdminStateUp:   &adminStateUp,
			Initiator:      initiator,
			IKEPolicyID:    d.Get("ikepolicy_id").(string),
			TenantID:       d.Get("tenant_id").(string),
			PeerID:         d.Get("peer_id").(string),
			PeerAddress:    d.Get("peer_address").(string),
			PeerEPGroupID:  d.Get("peer_ep_group_id").(string),
			LocalID:        d.Get("local_id").(string),
			VPNServiceID:   d.Get("vpnservice_id").(string),
			LocalEPGroupID: d.Get("local_ep_group_id").(string),
			IPSecPolicyID:  d.Get("ipsecpolicy_id").(string),
			PSK:            d.Get("psk").(string),
			MTU:            d.Get("mtu").(int),
			PeerCIDRs:      peerCidrs,
			DPD:            &dpd,
		},
		MapValueSpecs(d),
	}

	log.Printf("[DEBUG] Create site connection: %#v", createOpts)

	conn, err := siteconnections.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"NOT_CREATED"},
		Target:     []string{"PENDING_CREATE"},
		Refresh:    waitForSiteConnectionCreation(networkingClient, conn.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}
	_, err = stateConf.WaitForState()

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] SiteConnection created: %#v", conn)

	d.SetId(conn.ID)

	return resourceSiteConnectionV2Read(d, meta)
}

func resourceSiteConnectionV2Read(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about site connection: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	conn, err := siteconnections.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "site_connection")
	}

	log.Printf("[DEBUG] Read OpenStack SiteConnection %s: %#v", d.Id(), conn)

	d.Set("name", conn.Name)
	d.Set("description", conn.Description)
	d.Set("admin_state_up", conn.AdminStateUp)
	d.Set("tenant_id", conn.TenantID)
	d.Set("initiator", conn.Initiator)
	d.Set("ikepolicy_id", conn.IKEPolicyID)
	d.Set("peer_id", conn.PeerID)
	d.Set("peer_address", conn.PeerAddress)
	d.Set("local_id", conn.LocalID)
	d.Set("peer_ep_group_id", conn.PeerEPGroupID)
	d.Set("vpnservice_id", conn.VPNServiceID)
	d.Set("local_ep_group_id", conn.LocalEPGroupID)
	d.Set("ipsecpolicy_id", conn.IPSecPolicyID)
	d.Set("psk", conn.PSK)
	d.Set("mtu", conn.MTU)
	d.Set("peer_cidrs", conn.PeerCIDRs)

	// Set the dpd
	var dpdMap map[string]interface{}
	dpdMap = make(map[string]interface{})
	dpdMap["action"] = conn.DPD.Action
	dpdMap["interval"] = conn.DPD.Interval
	dpdMap["timeout"] = conn.DPD.Timeout

	var dpd []map[string]interface{}
	dpd = append(dpd, dpdMap)
	if err := d.Set("dpd", &dpd); err != nil {
		log.Printf("[WARN] unable to set Site connection DPD")
	}

	return nil
}

func resourceSiteConnectionV2Update(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := siteconnections.UpdateOpts{}

	var hasChange bool

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts.Name = &name
		hasChange = true
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		opts.Description = &description
		hasChange = true
	}

	if d.HasChange("admin_state_up") {
		adminStateUp := d.Get("admin_state_up").(bool)
		opts.AdminStateUp = &adminStateUp
		hasChange = true
	}

	if d.HasChange("local_id") {
		opts.LocalID = d.Get("local_id").(string)
		hasChange = true
	}

	if d.HasChange("peer_address") {
		opts.PeerAddress = d.Get("peer_address").(string)
		hasChange = true
	}

	if d.HasChange("peer_id") {
		opts.PeerID = d.Get("peer_id").(string)
		hasChange = true
	}

	if d.HasChange("local_ep_group_id") {
		opts.LocalEPGroupID = d.Get("local_ep_group_id").(string)
		hasChange = true
	}

	if d.HasChange("peer_ep_group_id") {
		opts.PeerEPGroupID = d.Get("peer_ep_group_id").(string)
		hasChange = true
	}

	if d.HasChange("psk") {
		opts.PSK = d.Get("psk").(string)
		hasChange = true
	}

	if d.HasChange("mtu") {
		opts.MTU = d.Get("mtu").(int)
		hasChange = true
	}

	if d.HasChange("initiator") {
		initiator := resourceSiteConnectionV2Initiator(d.Get("initiator").(string))
		opts.Initiator = initiator
		hasChange = true
	}

	if d.HasChange("peer_cidrs") {
		opts.PeerCIDRs = d.Get("peer_cidrs").([]string)
		hasChange = true
	}

	if d.HasChange("dpd") {
		dpdUpdateOpts := resourceSiteConnectionV2DPDUpdateOpts(d.Get("dpd").(*schema.Set))
		opts.DPD = &dpdUpdateOpts
		hasChange = true
	}

	var updateOpts siteconnections.UpdateOptsBuilder
	updateOpts = opts

	log.Printf("[DEBUG] Updating site connection with id %s: %#v", d.Id(), updateOpts)

	if hasChange {
		conn, err := siteconnections.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return err
		}
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"PENDING_UPDATE"},
			Target:     []string{"UPDATED"},
			Refresh:    waitForSiteConnectionUpdate(networkingClient, conn.ID),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      0,
			MinTimeout: 2 * time.Second,
		}
		_, err = stateConf.WaitForState()

		if err != nil {
			return err
		}

		log.Printf("[DEBUG] Updated connection with id %s", d.Id())
	}

	return resourceSiteConnectionV2Read(d, meta)
}

func resourceSiteConnectionV2Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy service: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	err = siteconnections.Delete(networkingClient, d.Id()).Err

	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     []string{"DELETED"},
		Refresh:    waitForSiteConnectionDeletion(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func waitForSiteConnectionDeletion(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		conn, err := siteconnections.Get(networkingClient, id).Extract()
		log.Printf("[DEBUG] Got site connection %s => %#v", id, conn)

		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] SiteConnection %s is actually deleted", id)
				return "", "DELETED", nil
			}
			return nil, "", fmt.Errorf("Unexpected error: %s", err)
		}

		log.Printf("[DEBUG] SiteConnection %s deletion is pending", id)
		return conn, "DELETING", nil
	}
}

func waitForSiteConnectionCreation(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		service, err := siteconnections.Get(networkingClient, id).Extract()
		if err != nil {
			return "", "NOT_CREATED", nil
		}
		return service, "PENDING_CREATE", nil
	}
}

func waitForSiteConnectionUpdate(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn, err := siteconnections.Get(networkingClient, id).Extract()
		if err != nil {
			return "", "PENDING_UPDATE", nil
		}
		return conn, "UPDATED", nil
	}
}

func resourceSiteConnectionV2Initiator(initatorString string) siteconnections.Initiator {
	var ini siteconnections.Initiator
	switch initatorString {
	case "bi-directional":
		ini = siteconnections.InitiatorBiDirectional
	case "response-only":
		ini = siteconnections.InitiatorResponseOnly
	}
	return ini
}

func resourceSiteConnectionV2DPDCreateOpts(d *schema.Set) siteconnections.DPDCreateOpts {
	dpd := siteconnections.DPDCreateOpts{}

	rawPairs := d.List()
	for _, raw := range rawPairs {
		rawMap := raw.(map[string]interface{})
		dpd.Action = resourceSiteConnectionV2Action(rawMap["action"].(string))

		timeout := rawMap["timeout"].(int)
		dpd.Timeout = timeout

		interval := rawMap["interval"].(int)
		dpd.Interval = interval
	}
	return dpd
}
func resourceSiteConnectionV2Action(actionString string) siteconnections.Action {
	var act siteconnections.Action
	switch actionString {
	case "hold":
		act = siteconnections.ActionHold
	case "restart":
		act = siteconnections.ActionRestart
	case "disabled":
		act = siteconnections.ActionDisabled
	case "restart-by-peer":
		act = siteconnections.ActionRestartByPeer
	case "clear":
		act = siteconnections.ActionClear
	}
	return act
}

func resourceSiteConnectionV2DPDUpdateOpts(d *schema.Set) siteconnections.DPDUpdateOpts {
	dpd := siteconnections.DPDUpdateOpts{}

	rawPairs := d.List()
	for _, raw := range rawPairs {
		rawMap := raw.(map[string]interface{})
		dpd.Action = resourceSiteConnectionV2Action(rawMap["action"].(string))

		timeout := rawMap["timeout"].(int)
		dpd.Timeout = timeout

		interval := rawMap["interval"].(int)
		dpd.Interval = interval
	}
	return dpd
}
