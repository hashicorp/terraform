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
				Required: true,
				MinItems: 1,
				// 18 total, auto_https nor forwarding can be set with others
				MaxItems: 16,
				Elem: &schema.Resource{
					SchemaVersion: 1,
					Schema: map[string]*schema.Schema{
						/* These are binary options, but in order to tell if
						 * they've been set to one or the other by a user inside
						 * a TypeSet, we need TypeString (and Cloudflare expects
						 * "on" or "off" anyway. */
						"always_online": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},

						"automatic_https_rewrites": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},

						"browser_check": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},

						"email_obfuscation": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},

						"ip_geolocation": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},

						"opportunistic_encryption": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},

						"server_side_exclude": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},

						"smart_errors": {
							Type:         schema.TypeString,
							ValidateFunc: validateOnOff,
							Optional:     true,
						},
						// End binary

						/* These are unitary options; we need TypeBools though
						 * in order to determine (inside a TypeSet) if they've
						 * been set *by the user*. */
						"always_use_https": {
							Type:         schema.TypeBool,
							Optional:     true,
							ValidateFunc: validateIsTrue,
						},

						"disable_apps": {
							Type:         schema.TypeBool,
							Optional:     true,
							ValidateFunc: validateIsTrue,
						},

						"disable_performance": {
							Type:         schema.TypeBool,
							Optional:     true,
							ValidateFunc: validateIsTrue,
						},

						"disable_security": {
							Type:         schema.TypeBool,
							Optional:     true,
							ValidateFunc: validateIsTrue,
						},
						// End unitaries

						"browser_cache_ttl": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateTTL,
						},

						"edge_cache_ttl": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validateTTL,
						},

						"cache_level": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateCacheLevel,
						},

						"forwarding_url": {
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

						"rocket_loader": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateRocketLoader,
						},

						"security_level": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateSecurityLevel,
						},

						"ssl": {
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

	actions := d.Get("actions").(*schema.Set).List()
	newPageRuleActions := make([]cloudflare.PageRuleAction, 0, len(actions))
	for _, action := range actions {
		newPageRuleAction, err := transformToCloudFlarePageRuleAction(action.(map[string]interface{}))
		if err != nil {
			return err
		}
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
		key, value, err := transformFromCloudFlarePageRuleAction(&pageRuleAction)
		if err != nil {
			return err
		}
		actions = append(actions, map[string]interface{}{key: value})
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

	if v, ok := d.GetOk("actions"); ok {
		actions := v.(*schema.Set).List()
		newPageRuleActions := make([]cloudflare.PageRuleAction, 0, len(actions))

		for _, action := range actions {
			newPageRuleAction, err := transformToCloudFlarePageRuleAction(action.(map[string]interface{}))
			if err != nil {
				return err
			}
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

func transformFromCloudFlarePageRuleAction(pageRuleAction *cloudflare.PageRuleAction) (key string, value interface{}, err error) {
	key = pageRuleAction.ID

	switch pageRuleAction.ID {
	case "always_online":
	case "automatic_https_rewrites":
	case "browser_check":
	case "email_obfuscation":
	case "ip_geolocation":
	case "opportunistic_encryption":
	case "server_side_exclude":
	case "smart_errors":
		if pageRuleAction.Value.(bool) {
			value = true
		} else {
			value = false
		}
		break

	case "always_use_https":
	case "disable_apps":
	case "disable_performance":
	case "disable_security":
		value = true
		break

	case "browser_cache_ttl":
	case "edge_cache_ttl":
		value = pageRuleAction.Value.(int)
		break

	case "cache_level":
	case "rocket_loader":
	case "security_level":
	case "ssl":
		value = pageRuleAction.Value.(string)
		break

	case "forwarding_url":
		value = pageRuleAction.Value.(map[string]interface{})
		break

	default:
		// User supplied ID is already validated, so this is always an internal error
		err = fmt.Errorf("Unimplemented action ID %q. This is always an internal error.", pageRuleAction.ID)
	}
	return
}

func transformToCloudFlarePageRuleAction(action map[string]interface{}) (pageRuleAction cloudflare.PageRuleAction, err error) {
	var id string
	var value interface{}

	for k, v := range action {
		if strValue, ok := v.(string); ok {
			if id == "" && strValue != "" {
				id, value = k, v
			} else if strValue != "" {
				err = fmt.Errorf("Must only have one action per element, found %q and %q", id, k)
			}
		} else if boolValue, ok := v.(bool); ok {
			if id == "" && boolValue {
				id, value = k, boolValue
			} else if boolValue {
				err = fmt.Errorf("Must only have one action per element, found %q and %q", id, k)
			}
		} else if intValue, ok := v.(int); ok {
			if id == "" && intValue != 0 {
				id, value = k, intValue
			} else if intValue != 0 {
				err = fmt.Errorf("Must only have one action per element, found %q and %q", id, k)
			}
		} else if k == "forwarding_url" {
			forwardActionSchema := v.(*schema.Set).List()
			if len(forwardActionSchema) != 0 {
				// We've already validated the status_code, and enforced 2
				// elements; so whichever isn't the status_code is the url.
				fst := forwardActionSchema[0].(map[string]interface{})
				snd := forwardActionSchema[1].(map[string]interface{})

				if statusCode := fst["status_code"].(int); statusCode != 0 {
					id, value = k, map[string]interface{}{
						"url":         snd["url"].(string),
						"status_code": statusCode,
					}
				} else if statusCode := snd["status_code"].(int); statusCode != 0 {
					id, value = k, map[string]interface{}{
						"url":         snd["url"].(string),
						"status_code": statusCode,
					}
				}
			}
		}
	}

	pageRuleAction.ID = id
	pageRuleAction.Value = value
	return
}
