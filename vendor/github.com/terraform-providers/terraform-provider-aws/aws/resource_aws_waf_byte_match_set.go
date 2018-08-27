package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafByteMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafByteMatchSetCreate,
		Read:   resourceAwsWafByteMatchSetRead,
		Update: resourceAwsWafByteMatchSetUpdate,
		Delete: resourceAwsWafByteMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"byte_match_tuples": {
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
						"positional_constraint": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target_string": {
							Type:     schema.TypeString,
							Optional: true,
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

func resourceAwsWafByteMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Creating ByteMatchSet: %s", d.Get("name").(string))

	wr := newWafRetryer(conn, "global")
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateByteMatchSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}
		return conn.CreateByteMatchSet(params)
	})
	if err != nil {
		return fmt.Errorf("Error creating ByteMatchSet: %s", err)
	}
	resp := out.(*waf.CreateByteMatchSetOutput)

	d.SetId(*resp.ByteMatchSet.ByteMatchSetId)

	return resourceAwsWafByteMatchSetUpdate(d, meta)
}

func resourceAwsWafByteMatchSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	log.Printf("[INFO] Reading ByteMatchSet: %s", d.Get("name").(string))
	params := &waf.GetByteMatchSetInput{
		ByteMatchSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetByteMatchSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.ByteMatchSet.Name)
	d.Set("byte_match_tuples", flattenWafByteMatchTuples(resp.ByteMatchSet.ByteMatchTuples))

	return nil
}

func resourceAwsWafByteMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Updating ByteMatchSet: %s", d.Get("name").(string))

	if d.HasChange("byte_match_tuples") {
		o, n := d.GetChange("byte_match_tuples")
		oldT, newT := o.(*schema.Set).List(), n.(*schema.Set).List()
		err := updateByteMatchSetResource(d.Id(), oldT, newT, conn)
		if err != nil {
			return fmt.Errorf("Error updating ByteMatchSet: %s", err)
		}
	}

	return resourceAwsWafByteMatchSetRead(d, meta)
}

func resourceAwsWafByteMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldTuples := d.Get("byte_match_tuples").(*schema.Set).List()
	if len(oldTuples) > 0 {
		noTuples := []interface{}{}
		err := updateByteMatchSetResource(d.Id(), oldTuples, noTuples, conn)
		if err != nil {
			return fmt.Errorf("Error updating ByteMatchSet: %s", err)
		}
	}

	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteByteMatchSetInput{
			ChangeToken:    token,
			ByteMatchSetId: aws.String(d.Id()),
		}
		log.Printf("[INFO] Deleting WAF ByteMatchSet: %s", req)
		return conn.DeleteByteMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting ByteMatchSet: %s", err)
	}

	return nil
}

func updateByteMatchSetResource(id string, oldT, newT []interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateByteMatchSetInput{
			ChangeToken:    token,
			ByteMatchSetId: aws.String(id),
			Updates:        diffWafByteMatchSetTuples(oldT, newT),
		}

		return conn.UpdateByteMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Error updating ByteMatchSet: %s", err)
	}

	return nil
}

func flattenWafByteMatchTuples(bmt []*waf.ByteMatchTuple) []interface{} {
	out := make([]interface{}, len(bmt), len(bmt))
	for i, t := range bmt {
		m := make(map[string]interface{})

		if t.FieldToMatch != nil {
			m["field_to_match"] = flattenFieldToMatch(t.FieldToMatch)
		}
		m["positional_constraint"] = *t.PositionalConstraint
		m["target_string"] = string(t.TargetString)
		m["text_transformation"] = *t.TextTransformation

		out[i] = m
	}
	return out
}

func diffWafByteMatchSetTuples(oldT, newT []interface{}) []*waf.ByteMatchSetUpdate {
	updates := make([]*waf.ByteMatchSetUpdate, 0)

	for _, ot := range oldT {
		tuple := ot.(map[string]interface{})

		if idx, contains := sliceContainsMap(newT, tuple); contains {
			newT = append(newT[:idx], newT[idx+1:]...)
			continue
		}

		updates = append(updates, &waf.ByteMatchSetUpdate{
			Action: aws.String(waf.ChangeActionDelete),
			ByteMatchTuple: &waf.ByteMatchTuple{
				FieldToMatch:         expandFieldToMatch(tuple["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				PositionalConstraint: aws.String(tuple["positional_constraint"].(string)),
				TargetString:         []byte(tuple["target_string"].(string)),
				TextTransformation:   aws.String(tuple["text_transformation"].(string)),
			},
		})
	}

	for _, nt := range newT {
		tuple := nt.(map[string]interface{})

		updates = append(updates, &waf.ByteMatchSetUpdate{
			Action: aws.String(waf.ChangeActionInsert),
			ByteMatchTuple: &waf.ByteMatchTuple{
				FieldToMatch:         expandFieldToMatch(tuple["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				PositionalConstraint: aws.String(tuple["positional_constraint"].(string)),
				TargetString:         []byte(tuple["target_string"].(string)),
				TextTransformation:   aws.String(tuple["text_transformation"].(string)),
			},
		})
	}
	return updates
}
