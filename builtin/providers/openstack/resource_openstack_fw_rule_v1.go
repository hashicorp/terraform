package openstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/policies"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/fwaas/rules"
)

func resourceFWRuleV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceFWRuleV1Create,
		Read:   resourceFWRuleV1Read,
		Update: resourceFWRuleV1Update,
		Delete: resourceFWRuleV1Delete,

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

func resourceFWRuleV1Create(d *schema.ResourceData, meta interface{}) error {

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

func resourceFWRuleV1Read(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	rule, err := rules.Get(networkingClient, d.Id()).Extract()

	if err != nil {
		return CheckDeleted(d, err, "LB pool")
	}

	d.Set("protocol", rule.Protocol)
	d.Set("action", rule.Action)

	if t, exists := d.GetOk("name"); exists && t != "" {
		d.Set("name", rule.Name)
	} else {
		d.Set("name", "")
	}

	if t, exists := d.GetOk("description"); exists && t != "" {
		d.Set("description", rule.Description)
	} else {
		d.Set("description", "")
	}

	if t, exists := d.GetOk("ip_version"); exists && t != "" {
		d.Set("ip_version", rule.IPVersion)
	} else {
		d.Set("ip_version", "")
	}

	if t, exists := d.GetOk("source_ip_address"); exists && t != "" {
		d.Set("source_ip_address", rule.SourceIPAddress)
	} else {
		d.Set("source_ip_address", "")
	}

	if t, exists := d.GetOk("destination_ip_address"); exists && t != "" {
		d.Set("destination_ip_address", rule.DestinationIPAddress)
	} else {
		d.Set("destination_ip_address", "")
	}

	if t, exists := d.GetOk("source_port"); exists && t != "" {
		d.Set("source_port", rule.SourcePort)
	} else {
		d.Set("source_port", "")
	}

	if t, exists := d.GetOk("destination_port"); exists && t != "" {
		d.Set("destination_port", rule.DestinationPort)
	} else {
		d.Set("destination_port", "")
	}

	if t, exists := d.GetOk("enabled"); exists && t != "" {
		d.Set("enabled", rule.Enabled)
	} else {
		d.Set("enabled", "")
	}

	return nil
}

func resourceFWRuleV1Update(d *schema.ResourceData, meta interface{}) error {
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

func resourceFWRuleV1Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	rule, err := rules.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return err
	}

	if rule.PolicyID != "" {
		err := policies.RemoveRule(networkingClient, rule.PolicyID, rule.ID)
		if err != nil {
			return err
		}
	}

	return rules.Delete(networkingClient, d.Id()).Err
}
