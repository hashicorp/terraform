package cloudflare

import (
	"fmt"
	"log"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCloudFlarePageRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudFlarePageRuleCreate,
		Read:   resourceCloudFlarePageRuleRead,
		Update: resourceCloudFlarePageRuleUpdate,
		Delete: resourceCloudFlarePageRuleDelete,

		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"targets": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"url_pattern": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"actions": &schema.Schema{
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validatePageRuleActionID,
						},

						"value": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validatePageRuleActionValue,
						},
					},
				},
			},

			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Default:  1,
				Optional: true,
			},

			"status": &schema.Schema{
				Type:         schema.TypeString,
				Default:      "active",
				Optional:     true,
				ValidateFunc: validatePageRuleStatus,
			},
		},
	}
}

func resourceCloudFlarePageRuleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)

	targets := d.Get("targets").([]interface{})
	actions := d.Get("actions").([]interface{})

	newPageRuleTargets := make([]cloudflare.PageRuleTarget, 0, len(targets))
	newPageRuleActions := make([]cloudflare.PageRuleAction, 0, len(actions))

	for _, target := range targets {
		newPageRuleTarget := cloudflare.PageRuleTarget{
			Target: "url",
			Constraint: struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			}{
				Operator: "matches",
				Value:    target.(schema.Resource).Schema["url_pattern"].Elem.(string),
			},
		}
		newPageRuleTargets = append(newPageRuleTargets, newPageRuleTarget)
	}

	for _, action := range actions {
		newPageRuleActions = append(newPageRuleActions, cloudflare.PageRuleAction{
			ID:    action.(schema.Resource).Schema["action"].Elem.(string),
			Value: action.(schema.Resource).Schema["value"].Elem.(string),
		})
	}

	newPageRule := cloudflare.PageRule{
		Targets:  newPageRuleTargets,
		Actions:  newPageRuleActions,
		Priority: d.Get("priority").(int),
		Status:   d.Get("status").(string),
	}

	zoneName := d.Get("domain").(string)

	zoneId, err := client.ZoneIDByName(zoneName)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", zoneName, err)
	}

	d.Set("zone_id", zoneId)
	log.Printf("[DEBUG] CloudFlare Page Rule create configuration: %#v", newPageRule)

	err = client.CreatePageRule(zoneId, newPageRule)
	if err != nil {
		return fmt.Errorf("Failed to create page rule: %s", err)
	}

	return resourceCloudFlarePageRuleRead(d, meta)
}

func resourceCloudFlarePageRuleRead(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Page Rule Read not implemented.")
}

func resourceCloudFlarePageRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Page Rule Update not implemented.")
}

func resourceCloudFlarePageRuleDelete(d *schema.ResourceData, meta interface{}) error {
	return fmt.Errorf("Page Rule Delete not implemented.")
}
