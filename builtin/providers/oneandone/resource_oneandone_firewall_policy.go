package oneandone

import (
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"strings"
)

func resourceOneandOneFirewallPolicy() *schema.Resource {
	return &schema.Resource{

		Create: resourceOneandOneFirewallCreate,
		Read:   resourceOneandOneFirewallRead,
		Update: resourceOneandOneFirewallUpdate,
		Delete: resourceOneandOneFirewallDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"rules": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"port_from": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
						"port_to": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
						"source_ip": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Required: true,
			},
		},
	}
}

func resourceOneandOneFirewallCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	req := oneandone.FirewallPolicyRequest{
		Name: d.Get("name").(string),
	}

	if desc, ok := d.GetOk("description"); ok {
		req.Description = desc.(string)
	}

	req.Rules = getRules(d)

	fw_id, fw, err := config.API.CreateFirewallPolicy(&req)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(fw, "ACTIVE", 10, config.Retries)
	if err != nil {
		return err
	}

	d.SetId(fw_id)

	if err != nil {
		return err
	}

	return resourceOneandOneFirewallRead(d, meta)
}

func resourceOneandOneFirewallUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	if d.HasChange("name") || d.HasChange("description") {
		fw, err := config.API.UpdateFirewallPolicy(d.Id(), d.Get("name").(string), d.Get("description").(string))
		if err != nil {
			return err
		}
		err = config.API.WaitForState(fw, "ACTIVE", 10, config.Retries)
		if err != nil {
			return err
		}
	}

	if d.HasChange("rules") {
		oldR, newR := d.GetChange("rules")
		oldValues := oldR.([]interface{})
		newValues := newR.([]interface{})
		if len(oldValues) > len(newValues) {
			diff := difference(oldValues, newValues)
			for _, old := range diff {
				o := old.(map[string]interface{})
				if o["id"] != nil {
					old_id := o["id"].(string)
					fw, err := config.API.DeleteFirewallPolicyRule(d.Id(), old_id)
					if err != nil {
						return err
					}

					err = config.API.WaitForState(fw, "ACTIVE", 10, config.Retries)
					if err != nil {
						return err
					}
				}
			}
		} else {
			var rules []oneandone.FirewallPolicyRule

			for _, raw := range newValues {
				rl := raw.(map[string]interface{})

				if rl["id"].(string) == "" {
					rule := oneandone.FirewallPolicyRule{
						Protocol: rl["protocol"].(string),
					}

					if rl["port_from"] != nil {
						rule.PortFrom = oneandone.Int2Pointer(rl["port_from"].(int))
					}
					if rl["port_to"] != nil {
						rule.PortTo = oneandone.Int2Pointer(rl["port_to"].(int))
					}

					if rl["source_ip"] != nil {
						rule.SourceIp = rl["source_ip"].(string)
					}

					rules = append(rules, rule)
				}
			}

			if len(rules) > 0 {
				fw, err := config.API.AddFirewallPolicyRules(d.Id(), rules)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(fw, "ACTIVE", 10, config.Retries)
			}
		}
	}

	return resourceOneandOneFirewallRead(d, meta)
}

func resourceOneandOneFirewallRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	fw, err := config.API.GetFirewallPolicy(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("rules", readRules(d, fw.Rules))
	d.Set("description", fw.Description)

	return nil
}

func resourceOneandOneFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	fp, err := config.API.DeleteFirewallPolicy(d.Id())
	if err != nil {
		return err
	}

	err = config.API.WaitUntilDeleted(fp)
	if err != nil {
		return err
	}

	return nil
}

func readRules(d *schema.ResourceData, rules []oneandone.FirewallPolicyRule) interface{} {
	rawRules := d.Get("rules").([]interface{})
	counter := 0
	for _, rR := range rawRules {
		if len(rules) > counter {
			rawMap := rR.(map[string]interface{})
			rawMap["id"] = rules[counter].Id
			if rules[counter].SourceIp != "0.0.0.0" {
				rawMap["source_ip"] = rules[counter].SourceIp
			}
		}
		counter++
	}

	return rawRules
}

func getRules(d *schema.ResourceData) []oneandone.FirewallPolicyRule {
	var rules []oneandone.FirewallPolicyRule

	if raw, ok := d.GetOk("rules"); ok {
		rawRules := raw.([]interface{})

		for _, raw := range rawRules {
			rl := raw.(map[string]interface{})
			rule := oneandone.FirewallPolicyRule{
				Protocol: rl["protocol"].(string),
			}

			if rl["port_from"] != nil {
				rule.PortFrom = oneandone.Int2Pointer(rl["port_from"].(int))
			}
			if rl["port_to"] != nil {
				rule.PortTo = oneandone.Int2Pointer(rl["port_to"].(int))
			}

			if rl["source_ip"] != nil {
				rule.SourceIp = rl["source_ip"].(string)
			}

			rules = append(rules, rule)
		}
	}
	return rules
}

func difference(oldV, newV []interface{}) (toreturn []interface{}) {
	var (
		lenMin  int
		longest []interface{}
	)
	// Determine the shortest length and the longest slice
	if len(oldV) < len(newV) {
		lenMin = len(oldV)
		longest = newV
	} else {
		lenMin = len(newV)
		longest = oldV
	}
	// compare common indeces
	for i := 0; i < lenMin; i++ {
		if oldV[i] == nil || newV[i] == nil {
			continue
		}
		if oldV[i].(map[string]interface{})["id"] != newV[i].(map[string]interface{})["id"] {
			toreturn = append(toreturn, newV) //out += fmt.Sprintf("=>\t%s\t%s\n", oldV[i], newV[i])
		}
	}
	// add indeces not in common
	for _, v := range longest[lenMin:] {
		//out += fmt.Sprintf("=>\t%s\n", v)
		toreturn = append(toreturn, v)
	}
	return toreturn
}
