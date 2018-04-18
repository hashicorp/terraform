package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/service/wafregional"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsWafRegionalRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalRuleCreate,
		Read:   resourceAwsWafRegionalRuleRead,
		Update: resourceAwsWafRegionalRuleUpdate,
		Delete: resourceAwsWafRegionalRuleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"metric_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"predicate": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"negated": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
						"data_id": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 128),
						},
						"type": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateWafPredicatesType(),
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRegionalRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateRuleInput{
			ChangeToken: token,
			MetricName:  aws.String(d.Get("metric_name").(string)),
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateRule(params)
	})
	if err != nil {
		return err
	}
	resp := out.(*waf.CreateRuleOutput)
	d.SetId(*resp.Rule.RuleId)
	return resourceAwsWafRegionalRuleUpdate(d, meta)
}

func resourceAwsWafRegionalRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	params := &waf.GetRuleInput{
		RuleId: aws.String(d.Id()),
	}

	resp, err := conn.GetRule(params)
	if err != nil {
		if isAWSErr(err, wafregional.ErrCodeWAFNonexistentItemException, "") {
			log.Printf("[WARN] WAF Rule (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("predicate", flattenWafPredicates(resp.Rule.Predicates))
	d.Set("name", resp.Rule.Name)
	d.Set("metric_name", resp.Rule.MetricName)

	return nil
}

func resourceAwsWafRegionalRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("predicate") {
		o, n := d.GetChange("predicate")
		oldP, newP := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateWafRegionalRuleResource(d.Id(), oldP, newP, meta)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Rule: %s", err)
		}
	}
	return resourceAwsWafRegionalRuleRead(d, meta)
}

func resourceAwsWafRegionalRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	oldPredicates := d.Get("predicate").(*schema.Set).List()
	if len(oldPredicates) > 0 {
		noPredicates := []interface{}{}
		err := updateWafRegionalRuleResource(d.Id(), oldPredicates, noPredicates, meta)
		if err != nil {
			return fmt.Errorf("Error Removing WAF Rule Predicates: %s", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteRuleInput{
			ChangeToken: token,
			RuleId:      aws.String(d.Id()),
		}
		log.Printf("[INFO] Deleting WAF Rule")
		return conn.DeleteRule(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting WAF Rule: %s", err)
	}

	return nil
}

func updateWafRegionalRuleResource(id string, oldP, newP []interface{}, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateRuleInput{
			ChangeToken: token,
			RuleId:      aws.String(id),
			Updates:     diffWafRulePredicates(oldP, newP),
		}

		return conn.UpdateRule(req)
	})

	if err != nil {
		return fmt.Errorf("Error Updating WAF Rule: %s", err)
	}

	return nil
}

func flattenWafPredicates(ts []*waf.Predicate) []interface{} {
	out := make([]interface{}, len(ts), len(ts))
	for i, p := range ts {
		m := make(map[string]interface{})
		m["negated"] = *p.Negated
		m["type"] = *p.Type
		m["data_id"] = *p.DataId
		out[i] = m
	}
	return out
}
