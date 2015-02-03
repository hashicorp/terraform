package openstack

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/firewalls"
)

func resourceFWFirewallV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallCreate,
		Read:   resourceFirewallRead,
		Update: resourceFirewallUpdate,
		Delete: resourceFirewallDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("OS_REGION_NAME"),
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
				ForceNew: true,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceFirewallCreate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	adminStateUp := d.Get("admin_state_up").(bool)

	firewallConfiguration := firewalls.CreateOpts{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		PolicyID:     d.Get("policy_id").(string),
		AdminStateUp: &adminStateUp,
	}

	log.Printf("[DEBUG] Create firewall: %#v", firewallConfiguration)

	firewall, err := firewalls.Create(networkingClient, firewallConfiguration).Extract()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Firewall created: %#v", firewall)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     "ACTIVE",
		Refresh:    WaitForFirewallActive(networkingClient, firewall.ID),
		Timeout:    30 * time.Second,
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	d.SetId(firewall.ID)

	d.Set("name", firewall.Name)
	d.Set("description", firewall.Description)
	d.Set("policy_id", firewall.PolicyID)
	d.Set("admin_state_up", firewall.AdminStateUp)

	_, err = stateConf.WaitForState()

	return nil
}

func resourceFirewallRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	firewall, err := firewalls.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		httpError, ok := err.(*perigee.UnexpectedResponseCodeError)
		if !ok {
			return err
		}

		if httpError.Actual == 404 {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", firewall.Name)
	d.Set("description", firewall.Description)
	d.Set("policy_id", firewall.PolicyID)
	d.Set("admin_state_up", firewall.AdminStateUp)

	return nil
}

func resourceFirewallUpdate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := firewalls.UpdateOpts{}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts.Name = &name
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		opts.Description = &description
	}

	if d.HasChange("policy_id") {
		opts.PolicyID = d.Get("policy_id").(string)
	}

	log.Printf("[DEBUG] Updating firewall with id %s: %#v", d.Id(), opts)

	if err := firewalls.Update(networkingClient, d.Id(), opts).Err; err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     "ACTIVE",
		Refresh:    WaitForFirewallActive(networkingClient, d.Id()),
		Timeout:    30 * time.Second,
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func resourceFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	err = firewalls.Delete(networkingClient, d.Id()).Err

	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     "DELETED",
		Refresh:    WaitForFirewallDeletion(networkingClient, d.Id()),
		Timeout:    2 * time.Minute,
		Delay:      0,
		MinTimeout: 2 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return err
}

func WaitForFirewallActive(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		fw, err := firewalls.Get(networkingClient, id).Extract()
		log.Printf("[DEBUG] Get firewall %s => %#v", id, fw)

		if err != nil {
			return nil, "", err
		}
		return fw, fw.Status, nil
	}
}

func WaitForFirewallDeletion(networkingClient *gophercloud.ServiceClient, id string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		fw, err := firewalls.Get(networkingClient, id).Extract()
		log.Printf("[DEBUG] Get firewall %s => %#v", id, fw)

		if err != nil {
			httpStatus := err.(*perigee.UnexpectedResponseCodeError)
			log.Printf("[DEBUG] Get firewall %s status is %d", id, httpStatus.Actual)

			if httpStatus.Actual == 404 {
				log.Printf("[DEBUG] Firewall %s is actually deleted", id)
				return "", "DELETED", nil
			}
			return nil, "", errors.New(fmt.Sprintf("Unexpected status code %d", httpStatus.Actual))
		}

		log.Printf("[DEBUG] Firewall %s deletion is pending", id)
		return fw, "DELETING", nil
	}
}
