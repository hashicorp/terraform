package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRateBasedRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRateBasedRuleCreate,
		Read:   resourceAwsWafRateBasedRuleRead,
		Update: resourceAwsWafRateBasedRuleUpdate,
		Delete: resourceAwsWafRateBasedRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"metric_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateWafMetricName,
			},
			"predicates": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"negated": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
						"data_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if len(value) > 128 {
									errors = append(errors, fmt.Errorf(
										"%q cannot be longer than 128 characters", k))
								}
								return
							},
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(string)
								if value != "IPMatch" && value != "ByteMatch" && value != "SqlInjectionMatch" && value != "SizeConstraint" && value != "XssMatch" {
									errors = append(errors, fmt.Errorf(
										"%q must be one of IPMatch | ByteMatch | SqlInjectionMatch | SizeConstraint | XssMatch", k))
								}
								return
							},
						},
					},
				},
			},
			"rate_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"rate_limit": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value < 2000 {
						errors = append(errors, fmt.Errorf("%q cannot be less than 2000", k))
					}
					return
				},
			},
		},
	}
}

func resourceAwsWafRateBasedRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	wr := newWafRetryer(conn, "global")
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
	return resourceAwsWafRateBasedRuleUpdate(d, meta)
}

func resourceAwsWafRateBasedRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	params := &waf.GetRateBasedRuleInput{
		RuleId: aws.String(d.Id()),
	}

	resp, err := conn.GetRateBasedRule(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF Rate Based Rule (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	var predicates []map[string]interface{}

	for _, predicateSet := range resp.Rule.MatchPredicates {
		predicate := map[string]interface{}{
			"negated": *predicateSet.Negated,
			"type":    *predicateSet.Type,
			"data_id": *predicateSet.DataId,
		}
		predicates = append(predicates, predicate)
	}

	d.Set("predicates", predicates)
	d.Set("name", resp.Rule.Name)
	d.Set("metric_name", resp.Rule.MetricName)
	d.Set("rate_key", resp.Rule.RateKey)
	d.Set("rate_limit", resp.Rule.RateLimit)

	return nil
}

func resourceAwsWafRateBasedRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	if d.HasChange("predicates") {
		o, n := d.GetChange("predicates")
		oldP, newP := o.(*schema.Set).List(), n.(*schema.Set).List()
		rateLimit := d.Get("rate_limit")

		err := updateWafRateBasedRuleResource(d.Id(), oldP, newP, rateLimit, conn)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Rule: %s", err)
		}
	}

	return resourceAwsWafRateBasedRuleRead(d, meta)
}

func resourceAwsWafRateBasedRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldPredicates := d.Get("predicates").(*schema.Set).List()
	if len(oldPredicates) > 0 {
		noPredicates := []interface{}{}
		rateLimit := d.Get("rate_limit")

		err := updateWafRateBasedRuleResource(d.Id(), oldPredicates, noPredicates, rateLimit, conn)
		if err != nil {
			return fmt.Errorf("Error updating WAF Rate Based Rule Predicates: %s", err)
		}
	}

	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteRateBasedRuleInput{
			ChangeToken: token,
			RuleId:      aws.String(d.Id()),
		}
		log.Printf("[INFO] Deleting WAF Rate Based Rule")
		return conn.DeleteRateBasedRule(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting WAF Rate Based Rule: %s", err)
	}

	return nil
}

func updateWafRateBasedRuleResource(id string, oldP, newP []interface{}, rateLimit interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn, "global")
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
		return fmt.Errorf("Error Updating WAF Rate Based Rule: %s", err)
	}

	return nil
}
