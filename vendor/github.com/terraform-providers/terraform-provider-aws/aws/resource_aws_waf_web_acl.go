package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsWafWebAcl() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafWebAclCreate,
		Read:   resourceAwsWafWebAclRead,
		Update: resourceAwsWafWebAclUpdate,
		Delete: resourceAwsWafWebAclDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"default_action": {
				Type:     schema.TypeSet,
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
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateWafMetricName,
			},
			"rules": {
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

func resourceAwsWafWebAclCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	wr := newWafRetryer(conn)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateWebACLInput{
			ChangeToken:   token,
			DefaultAction: expandWafAction(d.Get("default_action").(*schema.Set).List()),
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
		if isAWSErr(err, waf.ErrCodeNonexistentItemException, "") {
			log.Printf("[WARN] WAF ACL (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if resp == nil || resp.WebACL == nil {
		log.Printf("[WARN] WAF ACL (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("default_action", flattenWafAction(resp.WebACL.DefaultAction)); err != nil {
		return fmt.Errorf("error setting default_action: %s", err)
	}
	d.Set("name", resp.WebACL.Name)
	d.Set("metric_name", resp.WebACL.MetricName)
	if err := d.Set("rules", flattenWafWebAclRules(resp.WebACL.Rules)); err != nil {
		return fmt.Errorf("error setting rules: %s", err)
	}

	return nil
}

func resourceAwsWafWebAclUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	if d.HasChange("default_action") || d.HasChange("rules") {
		o, n := d.GetChange("rules")
		oldR, newR := o.(*schema.Set).List(), n.(*schema.Set).List()

		wr := newWafRetryer(conn)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateWebACLInput{
				ChangeToken:   token,
				DefaultAction: expandWafAction(d.Get("default_action").(*schema.Set).List()),
				Updates:       diffWafWebAclRules(oldR, newR),
				WebACLId:      aws.String(d.Id()),
			}
			return conn.UpdateWebACL(req)
		})
		if err != nil {
			return fmt.Errorf("Error Updating WAF ACL: %s", err)
		}
	}

	return resourceAwsWafWebAclRead(d, meta)
}

func resourceAwsWafWebAclDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	// First, need to delete all rules
	rules := d.Get("rules").(*schema.Set).List()
	if len(rules) > 0 {
		wr := newWafRetryer(conn)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateWebACLInput{
				ChangeToken:   token,
				DefaultAction: expandWafAction(d.Get("default_action").(*schema.Set).List()),
				Updates:       diffWafWebAclRules(rules, []interface{}{}),
				WebACLId:      aws.String(d.Id()),
			}
			return conn.UpdateWebACL(req)
		})
		if err != nil {
			return fmt.Errorf("Error Removing WAF Regional ACL Rules: %s", err)
		}
	}

	wr := newWafRetryer(conn)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
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
