package triton

import (
	"context"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/triton-go"
)

func resourceFirewallRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceFirewallRuleCreate,
		Exists: resourceFirewallRuleExists,
		Read:   resourceFirewallRuleRead,
		Update: resourceFirewallRuleUpdate,
		Delete: resourceFirewallRuleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
			"description": {
				Description: "Human-readable description of the rule",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"global": {
				Description: "Indicates whether or not the rule is global",
				Type:        schema.TypeBool,
				Computed:    true,
			},
		},
	}
}

func resourceFirewallRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	rule, err := client.Firewall().CreateFirewallRule(context.Background(), &triton.CreateFirewallRuleInput{
		Rule:        d.Get("rule").(string),
		Enabled:     d.Get("enabled").(bool),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	d.SetId(rule.ID)

	return resourceFirewallRuleRead(d, meta)
}

func resourceFirewallRuleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*triton.Client)

	return resourceExists(client.Firewall().GetFirewallRule(context.Background(), &triton.GetFirewallRuleInput{
		ID: d.Id(),
	}))
}

func resourceFirewallRuleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	rule, err := client.Firewall().GetFirewallRule(context.Background(), &triton.GetFirewallRuleInput{
		ID: d.Id(),
	})
	if err != nil {
		return err
	}

	d.SetId(rule.ID)
	d.Set("rule", rule.Rule)
	d.Set("enabled", rule.Enabled)
	d.Set("global", rule.Global)
	d.Set("description", rule.Description)

	return nil
}

func resourceFirewallRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	_, err := client.Firewall().UpdateFirewallRule(context.Background(), &triton.UpdateFirewallRuleInput{
		ID:          d.Id(),
		Rule:        d.Get("rule").(string),
		Enabled:     d.Get("enabled").(bool),
		Description: d.Get("description").(string),
	})
	if err != nil {
		return err
	}

	return resourceFirewallRuleRead(d, meta)
}

func resourceFirewallRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	return client.Firewall().DeleteFirewallRule(context.Background(), &triton.DeleteFirewallRuleInput{
		ID: d.Id(),
	})
}
