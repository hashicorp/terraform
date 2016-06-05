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
			"security_group": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"action": &schema.Schema{
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
			"direction": &schema.Schema{
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
			"ip_range": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"protocol": &schema.Schema{
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
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func resourceScalewaySecurityGroupRuleCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	def := api.ScalewayNewSecurityGroupRule{
		Action:       d.Get("action").(string),
		Direction:    d.Get("direction").(string),
		IPRange:      d.Get("ip_range").(string),
		Protocol:     d.Get("protocol").(string),
		DestPortFrom: d.Get("port").(int),
	}

	err := scaleway.PostSecurityGroupRule(d.Get("security_group").(string), def)
	if err != nil {
		serr := err.(api.ScalewayAPIError)

		log.Printf("Error Posting Security Group Rule. Reason: %s. %#v", serr.APIMessage, serr)

		return serr
	}

	defs, e := scaleway.GetSecurityGroupRules(d.Get("security_group").(string))
	if e != nil {
		return e
	}
	for _, rule := range defs.Rules {
		if rule.Action == def.Action && rule.Direction == def.Direction && rule.IPRange == def.IPRange && rule.Protocol == def.Protocol {
			d.SetId(rule.ID)
			break
		}
	}
	if d.Id() == "" {
		return fmt.Errorf("Failed to find newly created security group rule.")
	}

	return resourceScalewaySecurityGroupRuleRead(d, m)
}

func resourceScalewaySecurityGroupRuleRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	rule, err := scaleway.GetASecurityGroupRule(d.Get("security_group").(string), d.Id())

	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("Error Reading Security Group Rule. Reason: %s. %#v", serr.APIMessage, serr)
		}

		if serr.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return serr
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

	var def = api.ScalewayNewSecurityGroupRule{
		Action:       d.Get("action").(string),
		Direction:    d.Get("direction").(string),
		IPRange:      d.Get("ip_range").(string),
		Protocol:     d.Get("protocol").(string),
		DestPortFrom: d.Get("port").(int),
	}

	if err := scaleway.PutSecurityGroupRule(def, d.Get("security_group").(string), d.Id()); err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {

			log.Printf("Error Updating Security Group Rule. Reason: %s. %#v", serr.APIMessage, serr)
		} else {
			log.Printf("Error Updating Security Group Rule. Reason: %#v", err)

		}

		return err
	}

	return nil
}

func resourceScalewaySecurityGroupRuleDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	err := scaleway.DeleteSecurityGroupRule(d.Get("security_group").(string), d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
