package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRuleGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRuleGroupCreate,
		Read:   resourceAwsWafRuleGroupRead,
		Update: resourceAwsWafRuleGroupUpdate,
		Delete: resourceAwsWafRuleGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"metric_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateWafMetricName,
			},
			"activated_rule": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Required: true,
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
						"rule_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  waf.WafRuleTypeRegular,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRuleGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	wr := newWafRetryer(conn)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateRuleGroupInput{
			ChangeToken: token,
			MetricName:  aws.String(d.Get("metric_name").(string)),
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateRuleGroup(params)
	})
	if err != nil {
		return err
	}
	resp := out.(*waf.CreateRuleGroupOutput)
	d.SetId(*resp.RuleGroup.RuleGroupId)
	return resourceAwsWafRuleGroupUpdate(d, meta)
}

func resourceAwsWafRuleGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	params := &waf.GetRuleGroupInput{
		RuleGroupId: aws.String(d.Id()),
	}

	resp, err := conn.GetRuleGroup(params)
	if err != nil {
		if isAWSErr(err, "WAFNonexistentItemException", "") {
			log.Printf("[WARN] WAF Rule Group (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	rResp, err := conn.ListActivatedRulesInRuleGroup(&waf.ListActivatedRulesInRuleGroupInput{
		RuleGroupId: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("error listing activated rules in WAF Rule Group (%s): %s", d.Id(), err)
	}

	d.Set("activated_rule", flattenWafActivatedRules(rResp.ActivatedRules))
	d.Set("name", resp.RuleGroup.Name)
	d.Set("metric_name", resp.RuleGroup.MetricName)

	return nil
}

func resourceAwsWafRuleGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	if d.HasChange("activated_rule") {
		o, n := d.GetChange("activated_rule")
		oldRules, newRules := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateWafRuleGroupResource(d.Id(), oldRules, newRules, conn)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Rule Group: %s", err)
		}
	}

	return resourceAwsWafRuleGroupRead(d, meta)
}

func resourceAwsWafRuleGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldRules := d.Get("activated_rule").(*schema.Set).List()
	err := deleteWafRuleGroup(d.Id(), oldRules, conn)

	return err
}

func deleteWafRuleGroup(id string, oldRules []interface{}, conn *waf.WAF) error {
	if len(oldRules) > 0 {
		noRules := []interface{}{}
		err := updateWafRuleGroupResource(id, oldRules, noRules, conn)
		if err != nil {
			return fmt.Errorf("Error updating WAF Rule Group Predicates: %s", err)
		}
	}

	wr := newWafRetryer(conn)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteRuleGroupInput{
			ChangeToken: token,
			RuleGroupId: aws.String(id),
		}
		log.Printf("[INFO] Deleting WAF Rule Group")
		return conn.DeleteRuleGroup(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting WAF Rule Group: %s", err)
	}
	return nil
}

func updateWafRuleGroupResource(id string, oldRules, newRules []interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateRuleGroupInput{
			ChangeToken: token,
			RuleGroupId: aws.String(id),
			Updates:     diffWafRuleGroupActivatedRules(oldRules, newRules),
		}

		return conn.UpdateRuleGroup(req)
	})
	if err != nil {
		return fmt.Errorf("Error Updating WAF Rule Group: %s", err)
	}

	return nil
}
