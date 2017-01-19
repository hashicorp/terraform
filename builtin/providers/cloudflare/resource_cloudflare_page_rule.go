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

			"target": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"actions": &schema.Schema{
				Type:     schema.TypeSet,
				MinItems: 1,
				MaxItems: 2,
				Required: true,
				Elem: &schema.Resource{
					SchemaVersion: 1,
					Schema: map[string]*schema.Schema{
						"action": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validatePageRuleActionID,
						},

						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},

						"seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateTTL,
						},

						"cache_mode": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateCacheLevel,
						},

						"forward_target": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 2,
							MaxItems: 2,
							Elem: &schema.Resource{
								SchemaVersion: 1,
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										Required: true,
									},

									"status_code": {
										Type:         schema.TypeInt,
										Required:     true,
										ValidateFunc: validateForwardStatusCode,
									},
								},
							},
						},

						"rocket_mode": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRocketLoader,
						},

						"security_mode": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateSecurityLevel,
						},

						"ssl_mode": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateSSL,
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
	domain := d.Get("domain").(string)

	newPageRuleTargets := []cloudflare.PageRuleTarget{
		cloudflare.PageRuleTarget{
			Target: "url",
			Constraint: struct {
				Operator string `json:"operator"`
				Value    string `json:"value"`
			}{
				Operator: "matches",
				Value:    d.Get("target").(string),
			},
		},
	}

	actions := d.Get("actions").([]map[string]interface{})
	newPageRuleActions := make([]cloudflare.PageRuleAction, 0, len(actions))

	for _, v := range actions {
		newPageRuleAction := cloudflare.PageRuleAction{
			ID: v["action"].(string),
		}

		setPageRuleActionValue(&newPageRuleAction, v)
		newPageRuleActions = append(newPageRuleActions, newPageRuleAction)
	}

	newPageRule := cloudflare.PageRule{
		Targets:  newPageRuleTargets,
		Actions:  newPageRuleActions,
		Priority: d.Get("priority").(int),
		Status:   d.Get("status").(string),
	}

	zoneId, err := client.ZoneIDByName(domain)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", domain, err)
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
	client := meta.(*cloudflare.API)
	domain := d.Get("domain").(string)

	zoneId, err := client.ZoneIDByName(domain)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", domain, err)
	}

	pageRule, err := client.PageRule(zoneId, d.Id())
	if err != nil {
		return err
	}

	d.SetId(pageRule.ID)

	// Cloudflare presently only has one target type, and its Operator is always
	// "matches"; so we can just read the first element's Value.
	d.Set("target", pageRule.Targets[0].Constraint.Value)

	actions := make([]map[string]interface{}, 0, len(pageRule.Actions))
	for _, pageRuleAction := range pageRule.Actions {

		// Initialise with the defaults/'empty' values
		action := map[string]interface{}{
			"enabled":        true,
			"cache_mode":     "",
			"forward_target": "",
			"rocket_mode":    "",
			"seconds":        0,
			"security_mode":  "",
			"ssl_mode":       "",
		}

		if subsettingName, err := getPageRuleActionSubsetting(pageRuleAction.ID); err != nil {
			return err
		} else {
			switch subsettingName {
			case "enabled":
				action[subsettingName] = pageRuleAction.Value.(string) == "on"
				break

			case "none":
				break

			case "seconds":
				action[subsettingName] = pageRuleAction.Value.(int)
				break

			case "cache_mode":
			case "rocket_mode":
			case "security_mode":
			case "ssl_mode":
				action[subsettingName] = pageRuleAction.Value.(string)
				break

			case "forward_target":
				action[subsettingName] = map[string]interface{}{
					"url":         pageRuleAction.Value.(string),
					"status_code": pageRuleAction.Value.(int),
				}
				break

			default:
				return fmt.Errorf("Unimplemented action ID. This is always an internal error.")
			}
		}

		action["action"] = pageRuleAction.ID
		actions = append(actions, action)
	}
	d.Set("actions", actions)

	d.Set("priority", pageRule.Priority)
	d.Set("status", pageRule.Status)

	return nil
}

func resourceCloudFlarePageRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	domain := d.Get("domain").(string)

	updatePageRule := cloudflare.PageRule{
		ID: d.Id(),
	}

	if target, ok := d.GetOk("target"); ok {
		updatePageRule.Targets = []cloudflare.PageRuleTarget{
			cloudflare.PageRuleTarget{
				Target: "url",
				Constraint: struct {
					Operator string `json:"operator"`
					Value    string `json:"value"`
				}{
					Operator: "matches",
					Value:    target.(string),
				},
			},
		}
	}

	if actions, ok := d.GetOk("actions"); ok {
		vs := actions.([]map[string]interface{})
		newPageRuleActions := make([]cloudflare.PageRuleAction, 0, len(vs))

		for _, v := range vs {
			newPageRuleAction := cloudflare.PageRuleAction{
				ID: v["action"].(string),
			}

			setPageRuleActionValue(&newPageRuleAction, v)
			newPageRuleActions = append(newPageRuleActions, newPageRuleAction)
		}

		updatePageRule.Actions = newPageRuleActions
	}

	if priority, ok := d.GetOk("priority"); ok {
		updatePageRule.Priority = priority.(int)
	}

	if status, ok := d.GetOk("status"); ok {
		updatePageRule.Status = status.(string)
	}

	zoneId, err := client.ZoneIDByName(domain)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", domain, err)
	}

	log.Printf("[DEBUG] Cloudflare Page Rule update configuration: %#v", updatePageRule)

	if err := client.UpdatePageRule(zoneId, d.Id(), updatePageRule); err != nil {
		return fmt.Errorf("Failed to update Cloudflare Page Rule: %s", err)
	}

	return resourceCloudFlarePageRuleRead(d, meta)
}

func resourceCloudFlarePageRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.API)
	domain := d.Get("domain").(string)

	zoneId, err := client.ZoneIDByName(domain)
	if err != nil {
		return fmt.Errorf("Error finding zone %q: %s", domain, err)
	}

	log.Printf("[INFO] Deleting Cloudflare Page Rule: %s, %s", domain, d.Id())

	if err := client.DeletePageRule(zoneId, d.Id()); err != nil {
		return fmt.Errorf("Error deleting Cloudflare Page Rule: %s", err)
	}

	return nil
}

func getPageRuleActionSubsetting(actionID string) (subsettingName string, err error) {
	switch actionID {
	case "always_online":
	case "automatic_https_rewrites":
	case "browser_check":
	case "email_obfuscation":
	case "ip_geolocation":
	case "opportunistic_encryption":
	case "server_side_exclude":
	case "smart_errors":
		subsettingName = "enabled"
		break

	case "always_use_https":
	case "disable_apps":
	case "disable_performance":
	case "disable_security":
		subsettingName = "none"
		break

	case "browser_cache_ttl":
	case "edge_cache_ttl":
		subsettingName = "seconds"
		break

	case "cache_level":
		subsettingName = "cache_mode"
		break

	case "forwarding_url":
		subsettingName = "forward_target"
		break

	case "rocket_loader":
		subsettingName = "rocket_mode"
		break

	case "security_level":
		subsettingName = "security_mode"
		break

	case "ssl":
		subsettingName = "ssl_mode"
		break
	}
	return
}

func setPageRuleActionValue(pageRuleAction *cloudflare.PageRuleAction, v map[string]interface{}) (err error) {
	subsettingName, err := getPageRuleActionSubsetting(pageRuleAction.ID)
	if err != nil {
		return
	}

	switch subsettingName {
	case "enabled":
		if v[subsettingName].(bool) {
			pageRuleAction.Value = "on"
		} else {
			pageRuleAction.Value = "off"
		}
		break

	case "none":
		break

	case "seconds":
		subsetting := v[subsettingName].(int)
		if subsetting == 0 {
			err = fmt.Errorf("Action value missing for %q, expected to find %q", pageRuleAction.ID, subsettingName)
		} else {
			pageRuleAction.Value = subsetting
		}
		break

	case "cache_mode":
		subsetting := v[subsettingName].(string)
		if subsetting == "" {
			err = fmt.Errorf("Action value missing for %q, expected to find %q", pageRuleAction.ID, subsettingName)
		} else {
			pageRuleAction.Value = subsetting
		}
		break

	case "forward_target":
		forwardAction := v[subsettingName].(map[string]interface{})
		pageRuleAction.Value = struct {
			URL        string
			StatusCode int
		}{forwardAction["url"].(string), forwardAction["status_code"].(int)}
		break

	case "rocket_mode":
		subsetting := v[subsettingName].(string)
		if subsetting == "" {
			err = fmt.Errorf("Action value missing for %q, expected to find %q", pageRuleAction.ID, subsettingName)
		} else {
			pageRuleAction.Value = subsetting
		}
		break

	case "security_mode":
		subsetting := v[subsettingName].(string)
		if subsetting == "" {
			err = fmt.Errorf("Action value missing for %q, expected to find %q", pageRuleAction.ID, subsettingName)
		} else {
			pageRuleAction.Value = subsetting
		}
		break

	case "ssl_mode":
		subsetting := v[subsettingName].(string)
		if subsetting == "" {
			err = fmt.Errorf("Action value missing for %q, expected to find %q", pageRuleAction.ID, subsettingName)
		} else {
			pageRuleAction.Value = subsetting
		}
		break

	default:
		// User supplied ID is already validated, so this is always an internal error
		err = fmt.Errorf("Unimplemented action ID. This is always an internal error.")
	}
	return
}
