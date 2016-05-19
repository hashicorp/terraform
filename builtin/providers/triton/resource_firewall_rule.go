package triton

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/gocommon/errors"
	"github.com/joyent/gosdc/cloudapi"
)

func resourceFirewallRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallRuleCreate,
		Exists: resourceFirewallRuleExists,
		Read:   resourceFirewallRuleRead,
		Update: resourceFirewallRuleUpdate,
		Delete: resourceFirewallRuleDelete,

		Schema: map[string]*schema.Schema{
			"rule": {
				Description: "firewall rule text",
				Type:        schema.TypeString,
				Required:    true,
			},
			"enabled": {
				Description: "Indicates if the rule is enabled",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func resourceFirewallRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	rule, err := client.CreateFirewallRule(cloudapi.CreateFwRuleOpts{
		Rule:    d.Get("rule").(string),
		Enabled: d.Get("enabled").(bool),
	})
	if err != nil {
		return err
	}

	d.SetId(rule.Id)

	err = resourceFirewallRuleRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceFirewallRuleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*cloudapi.Client)

	rule, err := client.GetFirewallRule(d.Id())
	if errors.IsResourceNotFound(err) {
		return false, nil
	}

	return rule != nil && err == nil, err
}

func resourceFirewallRuleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	rule, err := client.GetFirewallRule(d.Id())
	if err != nil {
		return err
	}

	d.SetId(rule.Id)
	d.Set("rule", rule.Rule)
	d.Set("enabled", rule.Enabled)

	return nil
}

func resourceFirewallRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	_, err := client.UpdateFirewallRule(
		d.Id(),
		cloudapi.CreateFwRuleOpts{
			Rule:    d.Get("rule").(string),
			Enabled: d.Get("enabled").(bool),
		},
	)
	if err != nil {
		return err
	}

	return resourceFirewallRuleRead(d, meta)
}

func resourceFirewallRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	if err := client.DeleteFirewallRule(d.Id()); err != nil {
		return err
	}

	return nil
}
