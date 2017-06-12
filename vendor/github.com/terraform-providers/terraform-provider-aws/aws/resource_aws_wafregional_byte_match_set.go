package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalByteMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalByteMatchSetCreate,
		Read:   resourceAwsWafRegionalByteMatchSetRead,
		Update: resourceAwsWafRegionalByteMatchSetUpdate,
		Delete: resourceAwsWafRegionalByteMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"byte_match_tuple": &schema.Schema{
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
						"positional_constraint": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"target_string": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"text_transformation": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRegionalByteMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Creating ByteMatchSet: %s", d.Get("name").(string))

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateByteMatchSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}
		return conn.CreateByteMatchSet(params)
	})

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating ByteMatchSet: {{err}}", err)
	}
	resp := out.(*waf.CreateByteMatchSetOutput)

	d.SetId(*resp.ByteMatchSet.ByteMatchSetId)

	return resourceAwsWafRegionalByteMatchSetUpdate(d, meta)
}

func resourceAwsWafRegionalByteMatchSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	log.Printf("[INFO] Reading ByteMatchSet: %s", d.Get("name").(string))

	params := &waf.GetByteMatchSetInput{
		ByteMatchSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetByteMatchSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("byte_match_tuple", flattenWafByteMatchTuplesWR(resp.ByteMatchSet.ByteMatchTuples))
	d.Set("name", resp.ByteMatchSet.Name)

	return nil
}

func flattenWafByteMatchTuplesWR(in []*waf.ByteMatchTuple) []interface{} {
	tuples := make([]interface{}, len(in), len(in))

	for i, tuple := range in {
		field_to_match := tuple.FieldToMatch
		m := map[string]interface{}{
			"type": *field_to_match.Type,
		}

		if field_to_match.Data == nil {
			m["data"] = ""
		} else {
			m["data"] = *field_to_match.Data
		}

		var ms []map[string]interface{}
		ms = append(ms, m)

		tuple := map[string]interface{}{
			"field_to_match":        ms,
			"positional_constraint": *tuple.PositionalConstraint,
			"target_string":         tuple.TargetString,
			"text_transformation":   *tuple.TextTransformation,
		}
		tuples[i] = tuple
	}

	return tuples
}

func resourceAwsWafRegionalByteMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region
	log.Printf("[INFO] Updating ByteMatchSet: %s", d.Get("name").(string))

	if d.HasChange("byte_match_tuple") {
		o, n := d.GetChange("byte_match_tuple")
		oldT, newT := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateByteMatchSetResourceWR(d, oldT, newT, conn, region)
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error updating ByteMatchSet: {{err}}", err)
		}
	}
	return resourceAwsWafRegionalByteMatchSetRead(d, meta)
}

func resourceAwsWafRegionalByteMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Deleting ByteMatchSet: %s", d.Get("name").(string))

	oldT := d.Get("byte_match_tuple").(*schema.Set).List()

	if len(oldT) > 0 {
		var newT []interface{}

		err := updateByteMatchSetResourceWR(d, oldT, newT, conn, region)
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error deleting ByteMatchSet: {{err}}", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteByteMatchSetInput{
			ChangeToken:    token,
			ByteMatchSetId: aws.String(d.Id()),
		}
		return conn.DeleteByteMatchSet(req)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting ByteMatchSet: {{err}}", err)
	}

	return nil
}

func updateByteMatchSetResourceWR(d *schema.ResourceData, oldT, newT []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateByteMatchSetInput{
			ChangeToken:    token,
			ByteMatchSetId: aws.String(d.Id()),
			Updates:        diffByteMatchSetTuple(oldT, newT),
		}

		return conn.UpdateByteMatchSet(req)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating ByteMatchSet: {{err}}", err)
	}

	return nil
}

func expandFieldToMatchWR(d map[string]interface{}) *waf.FieldToMatch {
	return &waf.FieldToMatch{
		Type: aws.String(d["type"].(string)),
		Data: aws.String(d["data"].(string)),
	}
}

func flattenFieldToMatchWR(fm *waf.FieldToMatch) map[string]interface{} {
	m := make(map[string]interface{})
	m["data"] = *fm.Data
	m["type"] = *fm.Type
	return m
}

func diffByteMatchSetTuple(oldT, newT []interface{}) []*waf.ByteMatchSetUpdate {
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
