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

func resourceAwsWafRegionalRateBasedRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalRateBasedRuleCreate,
		Read:   resourceAwsWafRegionalRateBasedRuleRead,
		Update: resourceAwsWafRegionalRateBasedRuleUpdate,
		Delete: resourceAwsWafRegionalRateBasedRuleDelete,

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
			"predicate": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"negated": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"data_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(0, 128),
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateWafPredicatesType(),
						},
					},
				},
			},
			"rate_key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"rate_limit": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(2000),
			},
		},
	}
}

func resourceAwsWafRegionalRateBasedRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateRateBasedRuleInput{
			ChangeToken: token,
			MetricName:  aws.String(d.Get("metric_name").(string)),
			Name:        aws.String(d.Get("name").(string)),
			RateKey:     aws.String(d.Get("rate_key").(string)),
			RateLimit:   aws.Int64(int64(d.Get("rate_limit").(int))),
		}

		return conn.CreateRateBasedRule(params)
	})
	if err != nil {
		return err
	}
	resp := out.(*waf.CreateRateBasedRuleOutput)
	d.SetId(*resp.Rule.RuleId)
	return resourceAwsWafRegionalRateBasedRuleUpdate(d, meta)
}

func resourceAwsWafRegionalRateBasedRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	params := &waf.GetRateBasedRuleInput{
		RuleId: aws.String(d.Id()),
	}

	resp, err := conn.GetRateBasedRule(params)
	if err != nil {
		if isAWSErr(err, wafregional.ErrCodeWAFNonexistentItemException, "") {
			log.Printf("[WARN] WAF Regional Rate Based Rule (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	var predicates []map[string]interface{}

	for _, predicateSet := range resp.Rule.MatchPredicates {
		predicates = append(predicates, map[string]interface{}{
			"negated": *predicateSet.Negated,
			"type":    *predicateSet.Type,
			"data_id": *predicateSet.DataId,
		})
	}

	d.Set("predicate", predicates)
	d.Set("name", resp.Rule.Name)
	d.Set("metric_name", resp.Rule.MetricName)
	d.Set("rate_key", resp.Rule.RateKey)
	d.Set("rate_limit", resp.Rule.RateLimit)

	return nil
}

func resourceAwsWafRegionalRateBasedRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	if d.HasChange("predicate") || d.HasChange("rate_limit") {
		o, n := d.GetChange("predicate")
		oldP, newP := o.(*schema.Set).List(), n.(*schema.Set).List()
		rateLimit := d.Get("rate_limit")

		err := updateWafRateBasedRuleResourceWR(d.Id(), oldP, newP, rateLimit, conn, region)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Rule: %s", err)
		}
	}

	return resourceAwsWafRegionalRateBasedRuleRead(d, meta)
}

func resourceAwsWafRegionalRateBasedRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	oldPredicates := d.Get("predicate").(*schema.Set).List()
	if len(oldPredicates) > 0 {
		noPredicates := []interface{}{}
		rateLimit := d.Get("rate_limit")

		err := updateWafRateBasedRuleResourceWR(d.Id(), oldPredicates, noPredicates, rateLimit, conn, region)
		if err != nil {
			return fmt.Errorf("Error updating WAF Regional Rate Based Rule Predicates: %s", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteRateBasedRuleInput{
			ChangeToken: token,
			RuleId:      aws.String(d.Id()),
		}
		log.Printf("[INFO] Deleting WAF Regional Rate Based Rule")
		return conn.DeleteRateBasedRule(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting WAF Regional Rate Based Rule: %s", err)
	}

	return nil
}

func updateWafRateBasedRuleResourceWR(id string, oldP, newP []interface{}, rateLimit interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateRateBasedRuleInput{
			ChangeToken: token,
			RuleId:      aws.String(id),
			Updates:     diffWafRulePredicates(oldP, newP),
			RateLimit:   aws.Int64(int64(rateLimit.(int))),
		}

		return conn.UpdateRateBasedRule(req)
	})
	if err != nil {
		return fmt.Errorf("Error Updating WAF Regional Rate Based Rule: %s", err)
	}

	return nil
}
