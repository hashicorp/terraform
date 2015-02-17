package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/rules"
)

func resourceFWRuleV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallRuleCreate,
		Read:   resourceFirewallRuleRead,
		Update: resourceFirewallRuleUpdate,
		Delete: resourceFirewallRuleDelete,

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
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"action": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ip_version": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  4,
			},
			"source_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"destination_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_port": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"destination_port": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceFirewallRuleCreate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	enabled := d.Get("enabled").(bool)

	ruleConfiguration := rules.CreateOpts{
		Name:                 d.Get("name").(string),
		Description:          d.Get("description").(string),
		Protocol:             d.Get("protocol").(string),
		Action:               d.Get("action").(string),
		IPVersion:            d.Get("ip_version").(int),
		SourceIPAddress:      d.Get("source_ip_address").(string),
		DestinationIPAddress: d.Get("destination_ip_address").(string),
		SourcePort:           d.Get("source_port").(string),
		DestinationPort:      d.Get("destination_port").(string),
		Enabled:              &enabled,
		TenantID:             d.Get("tenant_id").(string),
	}

	log.Printf("[DEBUG] Create firewall rule: %#v", ruleConfiguration)

	rule, err := rules.Create(networkingClient, ruleConfiguration).Extract()

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Firewall rule with id %s : %#v", rule.ID, rule)

	d.SetId(rule.ID)

	return nil
}

func resourceFirewallRuleRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	rule, err := rules.Get(networkingClient, d.Id()).Extract()

	if err != nil {
		httpError, ok := err.(*gophercloud.UnexpectedResponseCodeError)
		if !ok {
			return err
		}
		if httpError.Actual == 404 {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", rule.Name)
	d.Set("description", rule.Description)
	d.Set("protocol", rule.Protocol)
	d.Set("action", rule.Action)
	d.Set("ip_version", rule.IPVersion)
	d.Set("source_ip_address", rule.SourceIPAddress)
	d.Set("destination_ip_address", rule.DestinationIPAddress)
	d.Set("source_port", rule.SourcePort)
	d.Set("destination_port", rule.DestinationPort)
	d.Set("enabled", rule.Enabled)

	return nil
}

func resourceFirewallRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := rules.UpdateOpts{}

	if d.HasChange("name") {
		opts.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		opts.Description = d.Get("description").(string)
	}
	if d.HasChange("protocol") {
		opts.Protocol = d.Get("protocol").(string)
	}
	if d.HasChange("action") {
		opts.Action = d.Get("action").(string)
	}
	if d.HasChange("ip_version") {
		opts.IPVersion = d.Get("ip_version").(int)
	}
	if d.HasChange("source_ip_address") {
		sourceIPAddress := d.Get("source_ip_address").(string)
		opts.SourceIPAddress = &sourceIPAddress
	}
	if d.HasChange("destination_ip_address") {
		destinationIPAddress := d.Get("destination_ip_address").(string)
		opts.DestinationIPAddress = &destinationIPAddress
	}
	if d.HasChange("source_port") {
		sourcePort := d.Get("source_port").(string)
		opts.SourcePort = &sourcePort
	}
	if d.HasChange("destination_port") {
		destinationPort := d.Get("destination_port").(string)
		opts.DestinationPort = &destinationPort
	}
	if d.HasChange("enabled") {
		enabled := d.Get("enabled").(bool)
		opts.Enabled = &enabled
	}

	log.Printf("[DEBUG] Updating firewall rules: %#v", opts)

	return rules.Update(networkingClient, d.Id(), opts).Err
}

func resourceFirewallRuleDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}
	return rules.Delete(networkingClient, d.Id()).Err
}
