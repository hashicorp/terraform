package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/policies"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/fwaas/rules"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFWRuleV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceFWRuleV1Create,
		Read:   resourceFWRuleV1Read,
		Update: resourceFWRuleV1Update,
		Delete: resourceFWRuleV1Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
			"value_specs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceFWRuleV1Create(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	enabled := d.Get("enabled").(bool)
	ipVersion := resourceFWRuleV1DetermineIPVersion(d.Get("ip_version").(int))
	protocol := resourceFWRuleV1DetermineProtocol(d.Get("protocol").(string))

	ruleConfiguration := RuleCreateOpts{
		rules.CreateOpts{
			Name:                 d.Get("name").(string),
			Description:          d.Get("description").(string),
			Protocol:             protocol,
			Action:               d.Get("action").(string),
			IPVersion:            ipVersion,
			SourceIPAddress:      d.Get("source_ip_address").(string),
			DestinationIPAddress: d.Get("destination_ip_address").(string),
			SourcePort:           d.Get("source_port").(string),
			DestinationPort:      d.Get("destination_port").(string),
			Enabled:              &enabled,
			TenantID:             d.Get("tenant_id").(string),
		},
		MapValueSpecs(d),
	}

	log.Printf("[DEBUG] Create firewall rule: %#v", ruleConfiguration)

	rule, err := rules.Create(networkingClient, ruleConfiguration).Extract()

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Firewall rule with id %s : %#v", rule.ID, rule)

	d.SetId(rule.ID)

	return resourceFWRuleV1Read(d, meta)
}

func resourceFWRuleV1Read(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Retrieve information about firewall rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	rule, err := rules.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "FW rule")
	}

	log.Printf("[DEBUG] Read OpenStack Firewall Rule %s: %#v", d.Id(), rule)

	d.Set("action", rule.Action)
	d.Set("name", rule.Name)
	d.Set("description", rule.Description)
	d.Set("ip_version", rule.IPVersion)
	d.Set("source_ip_address", rule.SourceIPAddress)
	d.Set("destination_ip_address", rule.DestinationIPAddress)
	d.Set("source_port", rule.SourcePort)
	d.Set("destination_port", rule.DestinationPort)
	d.Set("enabled", rule.Enabled)

	if rule.Protocol == "" {
		d.Set("protocol", "any")
	} else {
		d.Set("protocol", rule.Protocol)
	}

	d.Set("region", GetRegion(d))

	return nil
}

func resourceFWRuleV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	opts := rules.UpdateOpts{}

	if d.HasChange("name") {
		v := d.Get("name").(string)
		opts.Name = &v
	}

	if d.HasChange("description") {
		v := d.Get("description").(string)
		opts.Description = &v
	}

	if d.HasChange("protocol") {
		v := d.Get("protocol").(string)
		opts.Protocol = &v
	}

	if d.HasChange("action") {
		v := d.Get("action").(string)
		opts.Action = &v
	}

	if d.HasChange("ip_version") {
		v := d.Get("ip_version").(int)
		ipVersion := resourceFWRuleV1DetermineIPVersion(v)
		opts.IPVersion = &ipVersion
	}

	if d.HasChange("source_ip_address") {
		v := d.Get("source_ip_address").(string)
		opts.SourceIPAddress = &v
	}

	if d.HasChange("destination_ip_address") {
		v := d.Get("destination_ip_address").(string)
		opts.DestinationIPAddress = &v
	}

	if d.HasChange("source_port") {
		v := d.Get("source_port").(string)
		opts.SourcePort = &v
	}

	if d.HasChange("destination_port") {
		v := d.Get("destination_port").(string)
		opts.DestinationPort = &v
	}

	if d.HasChange("enabled") {
		v := d.Get("enabled").(bool)
		opts.Enabled = &v
	}

	log.Printf("[DEBUG] Updating firewall rules: %#v", opts)

	err = rules.Update(networkingClient, d.Id(), opts).Err
	if err != nil {
		return err
	}

	return resourceFWRuleV1Read(d, meta)
}

func resourceFWRuleV1Delete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] Destroy firewall rule: %s", d.Id())

	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	rule, err := rules.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return err
	}

	if rule.PolicyID != "" {
		_, err := policies.RemoveRule(networkingClient, rule.PolicyID, rule.ID).Extract()
		if err != nil {
			return err
		}
	}

	return rules.Delete(networkingClient, d.Id()).Err
}

func resourceFWRuleV1DetermineIPVersion(ipv int) gophercloud.IPVersion {
	// Determine the IP Version
	var ipVersion gophercloud.IPVersion
	switch ipv {
	case 4:
		ipVersion = gophercloud.IPv4
	case 6:
		ipVersion = gophercloud.IPv6
	}

	return ipVersion
}

func resourceFWRuleV1DetermineProtocol(p string) rules.Protocol {
	var protocol rules.Protocol
	switch p {
	case "any":
		protocol = rules.ProtocolAny
	case "icmp":
		protocol = rules.ProtocolICMP
	case "tcp":
		protocol = rules.ProtocolTCP
	case "udp":
		protocol = rules.ProtocolUDP
	}

	return protocol
}
