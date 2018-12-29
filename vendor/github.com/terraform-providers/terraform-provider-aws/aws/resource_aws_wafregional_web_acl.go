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
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"default_action": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"metric_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"override_action": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"priority": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  waf.WafRuleTypeRegular,
							ValidateFunc: validation.StringInSlice([]string{
								waf.WafRuleTypeRegular,
								waf.WafRuleTypeRateBased,
								waf.WafRuleTypeGroup,
							}, false),
						},
						"rule_id": {
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
			DefaultAction: expandWafAction(d.Get("default_action").([]interface{})),
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
			log.Printf("[WARN] WAF Regional ACL (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if resp == nil || resp.WebACL == nil {
		log.Printf("[WARN] WAF Regional ACL (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("default_action", flattenWafAction(resp.WebACL.DefaultAction)); err != nil {
		return fmt.Errorf("error setting default_action: %s", err)
	}
	d.Set("name", resp.WebACL.Name)
	d.Set("metric_name", resp.WebACL.MetricName)
	if err := d.Set("rule", flattenWafWebAclRules(resp.WebACL.Rules)); err != nil {
		return fmt.Errorf("error setting rule: %s", err)
	}

	return nil
}

func resourceAwsWafRegionalWebAclUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	if d.HasChange("default_action") || d.HasChange("rule") {
		o, n := d.GetChange("rule")
		oldR, newR := o.(*schema.Set).List(), n.(*schema.Set).List()

		wr := newWafRegionalRetryer(conn, region)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateWebACLInput{
				ChangeToken:   token,
				DefaultAction: expandWafAction(d.Get("default_action").([]interface{})),
				Updates:       diffWafWebAclRules(oldR, newR),
				WebACLId:      aws.String(d.Id()),
			}
			return conn.UpdateWebACL(req)
		})
		if err != nil {
			return fmt.Errorf("Error Updating WAF Regional ACL: %s", err)
		}
	}
	return resourceAwsWafRegionalWebAclRead(d, meta)
}

func resourceAwsWafRegionalWebAclDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	// First, need to delete all rules
	rules := d.Get("rule").(*schema.Set).List()
	if len(rules) > 0 {
		wr := newWafRegionalRetryer(conn, region)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateWebACLInput{
				ChangeToken:   token,
				DefaultAction: expandWafAction(d.Get("default_action").([]interface{})),
				Updates:       diffWafWebAclRules(rules, []interface{}{}),
				WebACLId:      aws.String(d.Id()),
			}
			return conn.UpdateWebACL(req)
		})
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
