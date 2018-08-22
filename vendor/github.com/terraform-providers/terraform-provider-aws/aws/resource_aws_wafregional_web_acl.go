package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsWafRegionalWebAcl() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalWebAclCreate,
		Read:   resourceAwsWafRegionalWebAclRead,
		Update: resourceAwsWafRegionalWebAclUpdate,
		Delete: resourceAwsWafRegionalWebAclDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"default_action": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"metric_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rule": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"override_action": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"priority": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  waf.WafRuleTypeRegular,
							ValidateFunc: validation.StringInSlice([]string{
								waf.WafRuleTypeRegular,
								waf.WafRuleTypeRateBased,
								waf.WafRuleTypeGroup,
							}, false),
						},
						"rule_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRegionalWebAclCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateWebACLInput{
			ChangeToken:   token,
			DefaultAction: expandDefaultActionWR(d.Get("default_action").([]interface{})),
			MetricName:    aws.String(d.Get("metric_name").(string)),
			Name:          aws.String(d.Get("name").(string)),
		}

		return conn.CreateWebACL(params)
	})
	if err != nil {
		return err
	}
	resp := out.(*waf.CreateWebACLOutput)
	d.SetId(*resp.WebACL.WebACLId)
	return resourceAwsWafRegionalWebAclUpdate(d, meta)
}

func resourceAwsWafRegionalWebAclRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	params := &waf.GetWebACLInput{
		WebACLId: aws.String(d.Id()),
	}

	resp, err := conn.GetWebACL(params)
	if err != nil {
		if isAWSErr(err, wafregional.ErrCodeWAFNonexistentItemException, "") {
			log.Printf("[WARN] WAF Regional ACL (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("default_action", flattenDefaultActionWR(resp.WebACL.DefaultAction))
	d.Set("name", resp.WebACL.Name)
	d.Set("metric_name", resp.WebACL.MetricName)
	d.Set("rule", flattenWafWebAclRules(resp.WebACL.Rules))

	return nil
}

func resourceAwsWafRegionalWebAclUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("default_action") || d.HasChange("rule") {
		conn := meta.(*AWSClient).wafregionalconn
		region := meta.(*AWSClient).region

		action := expandDefaultActionWR(d.Get("default_action").([]interface{}))
		o, n := d.GetChange("rule")
		oldR, newR := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateWebAclResourceWR(d.Id(), action, oldR, newR, conn, region)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Regional ACL: %s", err)
		}
	}
	return resourceAwsWafRegionalWebAclRead(d, meta)
}

func resourceAwsWafRegionalWebAclDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	action := expandDefaultActionWR(d.Get("default_action").([]interface{}))
	rules := d.Get("rule").(*schema.Set).List()
	if len(rules) > 0 {
		noRules := []interface{}{}
		err := updateWebAclResourceWR(d.Id(), action, rules, noRules, conn, region)
		if err != nil {
			return fmt.Errorf("Error Removing WAF Regional ACL Rules: %s", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteWebACLInput{
			ChangeToken: token,
			WebACLId:    aws.String(d.Id()),
		}

		log.Printf("[INFO] Deleting WAF ACL")
		return conn.DeleteWebACL(req)
	})
	if err != nil {
		return fmt.Errorf("Error Deleting WAF Regional ACL: %s", err)
	}
	return nil
}

func updateWebAclResourceWR(id string, a *waf.WafAction, oldR, newR []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateWebACLInput{
			DefaultAction: a,
			ChangeToken:   token,
			WebACLId:      aws.String(id),
			Updates:       diffWafWebAclRules(oldR, newR),
		}
		return conn.UpdateWebACL(req)
	})
	if err != nil {
		return fmt.Errorf("Error Updating WAF Regional ACL: %s", err)
	}
	return nil
}

func expandDefaultActionWR(d []interface{}) *waf.WafAction {
	if d == nil || len(d) == 0 {
		return nil
	}

	if d[0] == nil {
		log.Printf("[ERR] First element of Default Action is set to nil")
		return nil
	}

	dA := d[0].(map[string]interface{})

	return &waf.WafAction{
		Type: aws.String(dA["type"].(string)),
	}
}

func flattenDefaultActionWR(n *waf.WafAction) []map[string]interface{} {
	if n == nil {
		return nil
	}

	m := setMap(make(map[string]interface{}))

	m.SetString("type", n.Type)
	return m.MapList()
}

func flattenWafWebAclRules(ts []*waf.ActivatedRule) []interface{} {
	out := make([]interface{}, len(ts), len(ts))
	for i, r := range ts {
		m := make(map[string]interface{})

		switch *r.Type {
		case waf.WafRuleTypeGroup:
			actionMap := map[string]interface{}{
				"type": *r.OverrideAction.Type,
			}
			m["override_action"] = []interface{}{actionMap}
		default:
			actionMap := map[string]interface{}{
				"type": *r.Action.Type,
			}
			m["action"] = []interface{}{actionMap}
		}

		m["priority"] = *r.Priority
		m["rule_id"] = *r.RuleId
		m["type"] = *r.Type
		out[i] = m
	}
	return out
}

func expandWafWebAclUpdate(updateAction string, aclRule map[string]interface{}) *waf.WebACLUpdate {
	var rule *waf.ActivatedRule

	switch aclRule["type"].(string) {
	case waf.WafRuleTypeGroup:
		ruleAction := aclRule["override_action"].([]interface{})[0].(map[string]interface{})

		rule = &waf.ActivatedRule{
			OverrideAction: &waf.WafOverrideAction{Type: aws.String(ruleAction["type"].(string))},
			Priority:       aws.Int64(int64(aclRule["priority"].(int))),
			RuleId:         aws.String(aclRule["rule_id"].(string)),
			Type:           aws.String(aclRule["type"].(string)),
		}
	default:
		ruleAction := aclRule["action"].([]interface{})[0].(map[string]interface{})

		rule = &waf.ActivatedRule{
			Action:   &waf.WafAction{Type: aws.String(ruleAction["type"].(string))},
			Priority: aws.Int64(int64(aclRule["priority"].(int))),
			RuleId:   aws.String(aclRule["rule_id"].(string)),
			Type:     aws.String(aclRule["type"].(string)),
		}
	}

	update := &waf.WebACLUpdate{
		Action:        aws.String(updateAction),
		ActivatedRule: rule,
	}

	return update
}

func diffWafWebAclRules(oldR, newR []interface{}) []*waf.WebACLUpdate {
	updates := make([]*waf.WebACLUpdate, 0)

	for _, or := range oldR {
		aclRule := or.(map[string]interface{})

		if idx, contains := sliceContainsMap(newR, aclRule); contains {
			newR = append(newR[:idx], newR[idx+1:]...)
			continue
		}
		updates = append(updates, expandWafWebAclUpdate(waf.ChangeActionDelete, aclRule))
	}

	for _, nr := range newR {
		aclRule := nr.(map[string]interface{})
		updates = append(updates, expandWafWebAclUpdate(waf.ChangeActionInsert, aclRule))
	}
	return updates
}
