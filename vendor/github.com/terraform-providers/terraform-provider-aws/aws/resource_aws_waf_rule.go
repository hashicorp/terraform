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
		},
	}
}

func resourceAwsWafRuleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	wr := newWafRetryer(conn, "global")
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
	conn := meta.(*AWSClient).wafconn

	if d.HasChange("predicates") {
		o, n := d.GetChange("predicates")
		oldP, newP := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateWafRuleResource(d.Id(), oldP, newP, conn)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Rule: %s", err)
		}
	}

	return resourceAwsWafRuleRead(d, meta)
}

func resourceAwsWafRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldPredicates := d.Get("predicates").(*schema.Set).List()
	if len(oldPredicates) > 0 {
		noPredicates := []interface{}{}
		err := updateWafRuleResource(d.Id(), oldPredicates, noPredicates, conn)
		if err != nil {
			return fmt.Errorf("Error updating WAF Rule Predicates: %s", err)
		}
	}

	wr := newWafRetryer(conn, "global")
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

func updateWafRuleResource(id string, oldP, newP []interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn, "global")
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

func diffWafRulePredicates(oldP, newP []interface{}) []*waf.RuleUpdate {
	updates := make([]*waf.RuleUpdate, 0)

	for _, op := range oldP {
		predicate := op.(map[string]interface{})

		if idx, contains := sliceContainsMap(newP, predicate); contains {
			newP = append(newP[:idx], newP[idx+1:]...)
			continue
		}

		updates = append(updates, &waf.RuleUpdate{
			Action: aws.String(waf.ChangeActionDelete),
			Predicate: &waf.Predicate{
				Negated: aws.Bool(predicate["negated"].(bool)),
				Type:    aws.String(predicate["type"].(string)),
				DataId:  aws.String(predicate["data_id"].(string)),
			},
		})
	}

	for _, np := range newP {
		predicate := np.(map[string]interface{})

		updates = append(updates, &waf.RuleUpdate{
			Action: aws.String(waf.ChangeActionInsert),
			Predicate: &waf.Predicate{
				Negated: aws.Bool(predicate["negated"].(bool)),
				Type:    aws.String(predicate["type"].(string)),
				DataId:  aws.String(predicate["data_id"].(string)),
			},
		})
	}
	return updates
}
