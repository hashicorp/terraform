package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalByteMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalByteMatchSetCreate,
		Read:   resourceAwsWafRegionalByteMatchSetRead,
		Update: resourceAwsWafRegionalByteMatchSetUpdate,
		Delete: resourceAwsWafRegionalByteMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"byte_match_tuple": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"byte_match_tuples"},
				Deprecated:    "use `byte_match_tuples` instead",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field_to_match": {
							Type:     schema.TypeList,
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
			"byte_match_tuples": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field_to_match": {
							Type:     schema.TypeList,
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
		return fmt.Errorf("Error creating ByteMatchSet: %s", err)
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

	if isAWSErr(err, waf.ErrCodeNonexistentItemException, "") {
		log.Printf("[WARN] WAF Regional Byte Set Match (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error getting WAF Regional Byte Match Set (%s): %s", d.Id(), err)
	}

	if resp == nil || resp.ByteMatchSet == nil {
		log.Printf("[WARN] WAF Regional Byte Set Match (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if _, ok := d.GetOk("byte_match_tuple"); ok {
		if err := d.Set("byte_match_tuple", flattenWafByteMatchTuplesWR(resp.ByteMatchSet.ByteMatchTuples)); err != nil {
			return fmt.Errorf("error setting byte_match_tuple: %s", err)
		}
	} else {
		if err := d.Set("byte_match_tuples", flattenWafByteMatchTuplesWR(resp.ByteMatchSet.ByteMatchTuples)); err != nil {
			return fmt.Errorf("error setting byte_match_tuples: %s", err)
		}
	}
	d.Set("name", resp.ByteMatchSet.Name)

	return nil
}

func flattenWafByteMatchTuplesWR(in []*waf.ByteMatchTuple) []interface{} {
	tuples := make([]interface{}, len(in))

	for i, tuple := range in {
		fieldToMatchMap := map[string]interface{}{
			"data": aws.StringValue(tuple.FieldToMatch.Data),
			"type": aws.StringValue(tuple.FieldToMatch.Type),
		}

		m := map[string]interface{}{
			"field_to_match":        []map[string]interface{}{fieldToMatchMap},
			"positional_constraint": aws.StringValue(tuple.PositionalConstraint),
			"target_string":         string(tuple.TargetString),
			"text_transformation":   aws.StringValue(tuple.TextTransformation),
		}
		tuples[i] = m
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
			return fmt.Errorf("Error updating ByteMatchSet: %s", err)
		}
	} else if d.HasChange("byte_match_tuples") {
		o, n := d.GetChange("byte_match_tuples")
		oldT, newT := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateByteMatchSetResourceWR(d, oldT, newT, conn, region)
		if err != nil {
			return fmt.Errorf("Error updating ByteMatchSet: %s", err)
		}
	}
	return resourceAwsWafRegionalByteMatchSetRead(d, meta)
}

func resourceAwsWafRegionalByteMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Deleting ByteMatchSet: %s", d.Get("name").(string))

	var oldT []interface{}
	if _, ok := d.GetOk("byte_match_tuple"); ok {
		oldT = d.Get("byte_match_tuple").(*schema.Set).List()
	} else {
		oldT = d.Get("byte_match_tuples").(*schema.Set).List()
	}

	if len(oldT) > 0 {
		var newT []interface{}

		err := updateByteMatchSetResourceWR(d, oldT, newT, conn, region)
		if err != nil {
			return fmt.Errorf("Error deleting ByteMatchSet: %s", err)
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
		return fmt.Errorf("Error deleting ByteMatchSet: %s", err)
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
		return fmt.Errorf("Error updating ByteMatchSet: %s", err)
	}

	return nil
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
				FieldToMatch:         expandFieldToMatch(tuple["field_to_match"].([]interface{})[0].(map[string]interface{})),
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
				FieldToMatch:         expandFieldToMatch(tuple["field_to_match"].([]interface{})[0].(map[string]interface{})),
				PositionalConstraint: aws.String(tuple["positional_constraint"].(string)),
				TargetString:         []byte(tuple["target_string"].(string)),
				TextTransformation:   aws.String(tuple["text_transformation"].(string)),
			},
		})
	}
	return updates
}
