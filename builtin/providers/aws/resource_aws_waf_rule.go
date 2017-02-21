package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRuleCreate,
		Read:   resourceAwsWafRuleRead,
		Update: resourceAwsWafRuleUpdate,
		Delete: resourceAwsWafRuleDelete,

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
							Optional: true,
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
		},
	}
}

func resourceAwsWafRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	// ChangeToken
	var ct *waf.GetChangeTokenInput

	res, err := conn.GetChangeToken(ct)
	if err != nil {
		return fmt.Errorf("Error getting change token: %s", err)
	}

	params := &waf.CreateRuleInput{
		ChangeToken: res.ChangeToken,
		MetricName:  aws.String(d.Get("metric_name").(string)),
		Name:        aws.String(d.Get("name").(string)),
	}

	resp, err := conn.CreateRule(params)
	if err != nil {
		return err
	}
	d.SetId(*resp.Rule.RuleId)
	return resourceAwsWafRuleUpdate(d, meta)
}

func resourceAwsWafRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	params := &waf.GetRuleInput{
		RuleId: aws.String(d.Id()),
	}

	resp, err := conn.GetRule(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF Rule (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	var predicates []map[string]interface{}

	for _, predicateSet := range resp.Rule.Predicates {
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

	return nil
}

func resourceAwsWafRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	err := updateWafRuleResource(d, meta, waf.ChangeActionInsert)
	if err != nil {
		return fmt.Errorf("Error Updating WAF Rule: %s", err)
	}
	return resourceAwsWafRuleRead(d, meta)
}

func resourceAwsWafRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	err := updateWafRuleResource(d, meta, waf.ChangeActionDelete)
	if err != nil {
		return fmt.Errorf("Error Removing WAF Rule Predicates: %s", err)
	}
	// ChangeToken
	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)

	req := &waf.DeleteRuleInput{
		ChangeToken: resp.ChangeToken,
		RuleId:      aws.String(d.Id()),
	}
	log.Printf("[INFO] Deleting WAF Rule")
	_, err = conn.DeleteRule(req)

	if err != nil {
		return fmt.Errorf("Error deleting WAF Rule: %s", err)
	}

	return nil
}

func updateWafRuleResource(d *schema.ResourceData, meta interface{}, ChangeAction string) error {
	conn := meta.(*AWSClient).wafconn

	// ChangeToken
	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)
	if err != nil {
		return fmt.Errorf("Error getting change token: %s", err)
	}

	req := &waf.UpdateRuleInput{
		ChangeToken: resp.ChangeToken,
		RuleId:      aws.String(d.Id()),
	}

	predicatesSet := d.Get("predicates").(*schema.Set)
	for _, predicateI := range predicatesSet.List() {
		predicate := predicateI.(map[string]interface{})
		updatePredicate := &waf.RuleUpdate{
			Action: aws.String(ChangeAction),
			Predicate: &waf.Predicate{
				Negated: aws.Bool(predicate["negated"].(bool)),
				Type:    aws.String(predicate["type"].(string)),
				DataId:  aws.String(predicate["data_id"].(string)),
			},
		}
		req.Updates = append(req.Updates, updatePredicate)
	}

	_, err = conn.UpdateRule(req)
	if err != nil {
		return fmt.Errorf("Error Updating WAF Rule: %s", err)
	}

	return nil
}
