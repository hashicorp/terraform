package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalRuleGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalRuleGroupCreate,
		Read:   resourceAwsWafRegionalRuleGroupRead,
		Update: resourceAwsWafRegionalRuleGroupUpdate,
		Delete: resourceAwsWafRegionalRuleGroupDelete,

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
							Default:  wafregional.WafRuleTypeRegular,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRegionalRuleGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	wr := newWafRegionalRetryer(conn, region)
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
	return resourceAwsWafRegionalRuleGroupUpdate(d, meta)
}

func resourceAwsWafRegionalRuleGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	params := &waf.GetRuleGroupInput{
		RuleGroupId: aws.String(d.Id()),
	}

	resp, err := conn.GetRuleGroup(params)
	if err != nil {
		if isAWSErr(err, wafregional.ErrCodeWAFNonexistentItemException, "") {
			log.Printf("[WARN] WAF Regional Rule Group (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	rResp, err := conn.ListActivatedRulesInRuleGroup(&waf.ListActivatedRulesInRuleGroupInput{
		RuleGroupId: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("error listing activated rules in WAF Regional Rule Group (%s): %s", d.Id(), err)
	}

	d.Set("activated_rule", flattenWafActivatedRules(rResp.ActivatedRules))
	d.Set("name", resp.RuleGroup.Name)
	d.Set("metric_name", resp.RuleGroup.MetricName)

	return nil
}

func resourceAwsWafRegionalRuleGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	if d.HasChange("activated_rule") {
		o, n := d.GetChange("activated_rule")
		oldRules, newRules := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateWafRuleGroupResourceWR(d.Id(), oldRules, newRules, conn, region)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Regional Rule Group: %s", err)
		}
	}

	return resourceAwsWafRegionalRuleGroupRead(d, meta)
}

func resourceAwsWafRegionalRuleGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	oldRules := d.Get("activated_rule").(*schema.Set).List()
	err := deleteWafRegionalRuleGroup(d.Id(), oldRules, conn, region)

	return err
}

func deleteWafRegionalRuleGroup(id string, oldRules []interface{}, conn *wafregional.WAFRegional, region string) error {
	if len(oldRules) > 0 {
		noRules := []interface{}{}
		err := updateWafRuleGroupResourceWR(id, oldRules, noRules, conn, region)
		if err != nil {
			return fmt.Errorf("Error updating WAF Regional Rule Group Predicates: %s", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteRuleGroupInput{
			ChangeToken: token,
			RuleGroupId: aws.String(id),
		}
		log.Printf("[INFO] Deleting WAF Regional Rule Group")
		return conn.DeleteRuleGroup(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting WAF Regional Rule Group: %s", err)
	}
	return nil
}

func updateWafRuleGroupResourceWR(id string, oldRules, newRules []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateRuleGroupInput{
			ChangeToken: token,
			RuleGroupId: aws.String(id),
			Updates:     diffWafRuleGroupActivatedRules(oldRules, newRules),
		}

		return conn.UpdateRuleGroup(req)
	})
	if err != nil {
		return fmt.Errorf("Error Updating WAF Regional Rule Group: %s", err)
	}

	return nil
}
