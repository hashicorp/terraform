package scaleway

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func resourceScalewaySecurityGroupRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewaySecurityGroupRuleCreate,
		Read:   resourceScalewaySecurityGroupRuleRead,
		Update: resourceScalewaySecurityGroupRuleUpdate,
		Delete: resourceScalewaySecurityGroupRuleDelete,
		Schema: map[string]*schema.Schema{
			"security_group": {
				Type:     schema.TypeString,
				Required: true,
			},
			"action": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "accept" && value != "drop" {
						errors = append(errors, fmt.Errorf("%q must be one of 'accept', 'drop'", k))
					}
					return
				},
			},
			"direction": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "inbound" && value != "outbound" {
						errors = append(errors, fmt.Errorf("%q must be one of 'inbound', 'outbound'", k))
					}
					return
				},
			},
			"ip_range": {
				Type:     schema.TypeString,
				Required: true,
			},
			"protocol": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "ICMP" && value != "TCP" && value != "UDP" {
						errors = append(errors, fmt.Errorf("%q must be one of 'ICMP', 'TCP', 'UDP", k))
					}
					return
				},
			},
			"port": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func resourceScalewaySecurityGroupRuleCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	req := api.ScalewayNewSecurityGroupRule{
		Action:       d.Get("action").(string),
		Direction:    d.Get("direction").(string),
		IPRange:      d.Get("ip_range").(string),
		Protocol:     d.Get("protocol").(string),
		DestPortFrom: d.Get("port").(int),
	}

	err := scaleway.PostSecurityGroupRule(d.Get("security_group").(string), req)
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error creating Security Group Rule: %q\n", serr.APIMessage)
		}

		return err
	}

	resp, err := scaleway.GetSecurityGroupRules(d.Get("security_group").(string))
	if err != nil {
		return err
	}

	matches := func(rule api.ScalewaySecurityGroupRule) bool {
		return rule.Action == req.Action &&
			rule.Direction == req.Direction &&
			rule.IPRange == req.IPRange &&
			rule.Protocol == req.Protocol &&
			rule.DestPortFrom == req.DestPortFrom
	}

	for _, rule := range resp.Rules {
		if matches(rule) {
			d.SetId(rule.ID)
			break
		}
	}

	if d.Id() == "" {
		return fmt.Errorf("Failed to find created security group rule")
	}

	return resourceScalewaySecurityGroupRuleRead(d, m)
}

func resourceScalewaySecurityGroupRuleRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	rule, err := scaleway.GetASecurityGroupRule(d.Get("security_group").(string), d.Id())

	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] error reading Security Group Rule: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}

		return err
	}

	d.Set("action", rule.Rules.Action)
	d.Set("direction", rule.Rules.Direction)
	d.Set("ip_range", rule.Rules.IPRange)
	d.Set("protocol", rule.Rules.Protocol)
	d.Set("port", rule.Rules.DestPortFrom)

	return nil
}

func resourceScalewaySecurityGroupRuleUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	var req = api.ScalewayNewSecurityGroupRule{
		Action:       d.Get("action").(string),
		Direction:    d.Get("direction").(string),
		IPRange:      d.Get("ip_range").(string),
		Protocol:     d.Get("protocol").(string),
		DestPortFrom: d.Get("port").(int),
	}

	if err := scaleway.PutSecurityGroupRule(req, d.Get("security_group").(string), d.Id()); err != nil {
		log.Printf("[DEBUG] error updating Security Group Rule: %q", err)

		return err
	}

	return resourceScalewaySecurityGroupRuleRead(d, m)
}

func resourceScalewaySecurityGroupRuleDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	err := scaleway.DeleteSecurityGroupRule(d.Get("security_group").(string), d.Id())
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] error reading Security Group Rule: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}

		return err
	}

	d.SetId("")
	return nil
}
