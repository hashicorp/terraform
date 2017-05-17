package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/firewalls"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFWFirewallV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceFWFirewallV1Create,
		Read:   resourceFWFirewallV1Read,
		Update: resourceFWFirewallV1Update,
		Delete: resourceFWFirewallV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
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
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"policy_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"value_specs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceFWFirewallV1Create(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	adminStateUp := d.Get("admin_state_up").(bool)

	firewallConfiguration := FirewallCreateOpts{
		firewalls.CreateOpts{
			Name:         d.Get("name").(string),
			Description:  d.Get("description").(string),
			PolicyID:     d.Get("policy_id").(string),
			AdminStateUp: &adminStateUp,
			TenantID:     d.Get("tenant_id").(string),
		},
		MapValueSpecs(d),
	}

	log.Printf("[DEBUG] Create firewall: %#v", firewallConfiguration)

	firewall, err := firewalls.Create(networkingClient, firewallConfiguration).Extract()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Firewall created: %#v", firewall)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForFirewallActive(networkingClient, firewall.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	d.SetId(firewall.ID)

	return resourceFWFirewallV1Read(d, meta)
}

func resourceFWFirewallV1Read(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	firewall, err := firewalls.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "firewall")
	}

	log.Printf("[DEBUG] Read OpenStack Firewall %s: %#v", d.Id(), firewall)

	d.Set("name", firewall.Name)
	d.Set("description", firewall.Description)
	d.Set("policy_id", firewall.PolicyID)
	d.Set("admin_state_up", firewall.AdminStateUp)
	d.Set("tenant_id", firewall.TenantID)
	d.Set("region", GetRegion(d))

	return nil
}

func resourceFWFirewallV1Update(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := firewalls.UpdateOpts{}

	if d.HasChange("name") {
		opts.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		opts.Description = d.Get("description").(string)
	}

	if d.HasChange("policy_id") {
		opts.PolicyID = d.Get("policy_id").(string)
	}

	if d.HasChange("admin_state_up") {
		adminStateUp := d.Get("admin_state_up").(bool)
		opts.AdminStateUp = &adminStateUp
	}

	log.Printf("[DEBUG] Updating firewall with id %s: %#v", d.Id(), opts)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE", "PENDING_UPDATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForFirewallActive(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutUpdate),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	err = firewalls.Update(networkingClient, d.Id(), opts).Err
	if err != nil {
		return err
	}

	return resourceFWFirewallV1Read(d, meta)
}

func resourceFWFirewallV1Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	// Ensure the firewall was fully created/updated before being deleted.
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE", "PENDING_UPDATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForFirewallActive(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutUpdate),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	err = firewalls.Delete(networkingClient, d.Id()).Err

	if err != nil {
		return err
	}

	stateConf = &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     []string{"DELETED"},
		Refresh:    waitForFirewallDeletion(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func waitForFirewallActive(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		fw, err := firewalls.Get(networkingClient, id).Extract()
		log.Printf("[DEBUG] Get firewall %s => %#v", id, fw)

		if err != nil {
			return nil, "", err
		}
		return fw, fw.Status, nil
	}
}

func waitForFirewallDeletion(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		fw, err := firewalls.Get(networkingClient, id).Extract()
		log.Printf("[DEBUG] Get firewall %s => %#v", id, fw)

		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Firewall %s is actually deleted", id)
				return "", "DELETED", nil
			}
			return nil, "", fmt.Errorf("Unexpected error: %s", err)
		}

		log.Printf("[DEBUG] Firewall %s deletion is pending", id)
		return fw, "DELETING", nil
	}
}
