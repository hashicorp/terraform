package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafSqlInjectionMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafSqlInjectionMatchSetCreate,
		Read:   resourceAwsWafSqlInjectionMatchSetRead,
		Update: resourceAwsWafSqlInjectionMatchSetUpdate,
		Delete: resourceAwsWafSqlInjectionMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"sql_injection_match_tuples": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field_to_match": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"data": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"text_transformation": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafSqlInjectionMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Creating SqlInjectionMatchSet: %s", d.Get("name").(string))

	wr := newWafRetryer(conn)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateSqlInjectionMatchSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateSqlInjectionMatchSet(params)
	})
	if err != nil {
		return fmt.Errorf("Error creating SqlInjectionMatchSet: %s", err)
	}
	resp := out.(*waf.CreateSqlInjectionMatchSetOutput)
	d.SetId(*resp.SqlInjectionMatchSet.SqlInjectionMatchSetId)

	return resourceAwsWafSqlInjectionMatchSetUpdate(d, meta)
}

func resourceAwsWafSqlInjectionMatchSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	log.Printf("[INFO] Reading SqlInjectionMatchSet: %s", d.Get("name").(string))
	params := &waf.GetSqlInjectionMatchSetInput{
		SqlInjectionMatchSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetSqlInjectionMatchSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.SqlInjectionMatchSet.Name)
	d.Set("sql_injection_match_tuples", resp.SqlInjectionMatchSet.SqlInjectionMatchTuples)

	return nil
}

func resourceAwsWafSqlInjectionMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	if d.HasChange("sql_injection_match_tuples") {
		o, n := d.GetChange("sql_injection_match_tuples")
		oldT, newT := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateSqlInjectionMatchSetResource(d.Id(), oldT, newT, conn)
		if err != nil {
			return fmt.Errorf("Error updating SqlInjectionMatchSet: %s", err)
		}
	}

	return resourceAwsWafSqlInjectionMatchSetRead(d, meta)
}

func resourceAwsWafSqlInjectionMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldTuples := d.Get("sql_injection_match_tuples").(*schema.Set).List()

	if len(oldTuples) > 0 {
		noTuples := []interface{}{}
		err := updateSqlInjectionMatchSetResource(d.Id(), oldTuples, noTuples, conn)
		if err != nil {
			return fmt.Errorf("Error deleting SqlInjectionMatchSet: %s", err)
		}
	}

	wr := newWafRetryer(conn)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteSqlInjectionMatchSetInput{
			ChangeToken:            token,
			SqlInjectionMatchSetId: aws.String(d.Id()),
		}

		return conn.DeleteSqlInjectionMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting SqlInjectionMatchSet: %s", err)
	}

	return nil
}

func updateSqlInjectionMatchSetResource(id string, oldT, newT []interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateSqlInjectionMatchSetInput{
			ChangeToken:            token,
			SqlInjectionMatchSetId: aws.String(id),
			Updates:                diffWafSqlInjectionMatchTuples(oldT, newT),
		}

		log.Printf("[INFO] Updating SqlInjectionMatchSet: %s", req)
		return conn.UpdateSqlInjectionMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Error updating SqlInjectionMatchSet: %s", err)
	}

	return nil
}

func flattenWafSqlInjectionMatchTuples(ts []*waf.SqlInjectionMatchTuple) []interface{} {
	out := make([]interface{}, len(ts))
	for i, t := range ts {
		m := make(map[string]interface{})
		m["text_transformation"] = *t.TextTransformation
		m["field_to_match"] = flattenFieldToMatch(t.FieldToMatch)
		out[i] = m
	}

	return out
}

func diffWafSqlInjectionMatchTuples(oldT, newT []interface{}) []*waf.SqlInjectionMatchSetUpdate {
	updates := make([]*waf.SqlInjectionMatchSetUpdate, 0)

	for _, od := range oldT {
		tuple := od.(map[string]interface{})

		if idx, contains := sliceContainsMap(newT, tuple); contains {
			newT = append(newT[:idx], newT[idx+1:]...)
			continue
		}

		updates = append(updates, &waf.SqlInjectionMatchSetUpdate{
			Action: aws.String(waf.ChangeActionDelete),
			SqlInjectionMatchTuple: &waf.SqlInjectionMatchTuple{
				FieldToMatch:       expandFieldToMatch(tuple["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				TextTransformation: aws.String(tuple["text_transformation"].(string)),
			},
		})
	}

	for _, nd := range newT {
		tuple := nd.(map[string]interface{})

		updates = append(updates, &waf.SqlInjectionMatchSetUpdate{
			Action: aws.String(waf.ChangeActionInsert),
			SqlInjectionMatchTuple: &waf.SqlInjectionMatchTuple{
				FieldToMatch:       expandFieldToMatch(tuple["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				TextTransformation: aws.String(tuple["text_transformation"].(string)),
			},
		})
	}
	return updates
}
