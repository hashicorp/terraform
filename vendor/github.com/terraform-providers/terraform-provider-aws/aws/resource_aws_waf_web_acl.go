package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafWebAcl() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafWebAclCreate,
		Read:   resourceAwsWafWebAclRead,
		Update: resourceAwsWafWebAclUpdate,
		Delete: resourceAwsWafWebAclDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"default_action": &schema.Schema{
				Type:     schema.TypeSet,
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
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateWafMetricName,
			},
			"rules": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": &schema.Schema{
							Type:     schema.TypeSet,
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
						"priority": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  waf.WafRuleTypeRegular,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != waf.WafRuleTypeRegular && value != waf.WafRuleTypeRateBased {
									errors = append(errors, fmt.Errorf(
										"%q must be one of %s | %s", k, waf.WafRuleTypeRegular, waf.WafRuleTypeRateBased))
								}
								return
							},
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

func resourceAwsWafWebAclCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	wr := newWafRetryer(conn, "global")
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateWebACLInput{
			ChangeToken:   token,
			DefaultAction: expandDefaultAction(d),
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
	return resourceAwsWafWebAclUpdate(d, meta)
}

func resourceAwsWafWebAclRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	params := &waf.GetWebACLInput{
		WebACLId: aws.String(d.Id()),
	}

	resp, err := conn.GetWebACL(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF ACL (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	defaultAction := flattenDefaultAction(resp.WebACL.DefaultAction)
	if defaultAction != nil {
		if err := d.Set("default_action", defaultAction); err != nil {
			return fmt.Errorf("error setting default_action: %s", err)
		}
	}
	d.Set("name", resp.WebACL.Name)
	d.Set("metric_name", resp.WebACL.MetricName)

	return nil
}

func resourceAwsWafWebAclUpdate(d *schema.ResourceData, meta interface{}) error {
	err := updateWebAclResource(d, meta, waf.ChangeActionInsert)
	if err != nil {
		return fmt.Errorf("Error Updating WAF ACL: %s", err)
	}
	return resourceAwsWafWebAclRead(d, meta)
}

func resourceAwsWafWebAclDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	err := updateWebAclResource(d, meta, waf.ChangeActionDelete)
	if err != nil {
		return fmt.Errorf("Error Removing WAF ACL Rules: %s", err)
	}

	wr := newWafRetryer(conn, "global")
	_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteWebACLInput{
			ChangeToken: token,
			WebACLId:    aws.String(d.Id()),
		}

		log.Printf("[INFO] Deleting WAF ACL")
		return conn.DeleteWebACL(req)
	})
	if err != nil {
		return fmt.Errorf("Error Deleting WAF ACL: %s", err)
	}
	return nil
}

func updateWebAclResource(d *schema.ResourceData, meta interface{}, ChangeAction string) error {
	conn := meta.(*AWSClient).wafconn

	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateWebACLInput{
			ChangeToken: token,
			WebACLId:    aws.String(d.Id()),
		}

		if d.HasChange("default_action") {
			req.DefaultAction = expandDefaultAction(d)
		}

		rules := d.Get("rules").(*schema.Set)
		for _, rule := range rules.List() {
			aclRule := rule.(map[string]interface{})
			action := aclRule["action"].(*schema.Set).List()[0].(map[string]interface{})
			aclRuleUpdate := &waf.WebACLUpdate{
				Action: aws.String(ChangeAction),
				ActivatedRule: &waf.ActivatedRule{
					Priority: aws.Int64(int64(aclRule["priority"].(int))),
					RuleId:   aws.String(aclRule["rule_id"].(string)),
					Type:     aws.String(aclRule["type"].(string)),
					Action:   &waf.WafAction{Type: aws.String(action["type"].(string))},
				},
			}
			req.Updates = append(req.Updates, aclRuleUpdate)
		}
		return conn.UpdateWebACL(req)
	})
	if err != nil {
		return fmt.Errorf("Error Updating WAF ACL: %s", err)
	}
	return nil
}

func expandDefaultAction(d *schema.ResourceData) *waf.WafAction {
	set, ok := d.GetOk("default_action")
	if !ok {
		return nil
	}

	s := set.(*schema.Set).List()
	if s == nil || len(s) == 0 {
		return nil
	}

	if s[0] == nil {
		log.Printf("[ERR] First element of Default Action is set to nil")
		return nil
	}

	dA := s[0].(map[string]interface{})

	return &waf.WafAction{
		Type: aws.String(dA["type"].(string)),
	}
}

func flattenDefaultAction(n *waf.WafAction) []map[string]interface{} {
	if n == nil {
		return nil
	}

	m := setMap(make(map[string]interface{}))

	m.SetString("type", n.Type)
	return m.MapList()
}
