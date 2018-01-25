package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafSizeConstraintSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafSizeConstraintSetCreate,
		Read:   resourceAwsWafSizeConstraintSetRead,
		Update: resourceAwsWafSizeConstraintSetUpdate,
		Delete: resourceAwsWafSizeConstraintSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size_constraints": &schema.Schema{
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
						"comparison_operator": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
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

func resourceAwsWafSizeConstraintSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Creating SizeConstraintSet: %s", d.Get("name").(string))

	wr := newWafRetryer(conn, "global")
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateSizeConstraintSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateSizeConstraintSet(params)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SizeConstraintSet: {{err}}", err)
	}
	resp := out.(*waf.CreateSizeConstraintSetOutput)

	d.SetId(*resp.SizeConstraintSet.SizeConstraintSetId)

	return resourceAwsWafSizeConstraintSetUpdate(d, meta)
}

func resourceAwsWafSizeConstraintSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	log.Printf("[INFO] Reading SizeConstraintSet: %s", d.Get("name").(string))
	params := &waf.GetSizeConstraintSetInput{
		SizeConstraintSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetSizeConstraintSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.SizeConstraintSet.Name)
	d.Set("size_constraints", flattenWafSizeConstraints(resp.SizeConstraintSet.SizeConstraints))

	return nil
}

func resourceAwsWafSizeConstraintSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	if d.HasChange("size_constraints") {
		o, n := d.GetChange("size_constraints")
		oldS, newS := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateSizeConstraintSetResource(d.Id(), oldS, newS, conn)
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
		}
	}

	return resourceAwsWafSizeConstraintSetRead(d, meta)
}

func resourceAwsWafSizeConstraintSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldConstraints := d.Get("size_constraints").(*schema.Set).List()

	if len(oldConstraints) > 0 {
		noConstraints := []interface{}{}
		err := updateSizeConstraintSetResource(d.Id(), oldConstraints, noConstraints, conn)
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error deleting SizeConstraintSet: {{err}}", err)
		}
	}

	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteSizeConstraintSetInput{
			ChangeToken:         token,
			SizeConstraintSetId: aws.String(d.Id()),
		}
		return conn.DeleteSizeConstraintSet(req)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SizeConstraintSet: {{err}}", err)
	}

	return nil
}

func updateSizeConstraintSetResource(id string, oldS, newS []interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateSizeConstraintSetInput{
			ChangeToken:         token,
			SizeConstraintSetId: aws.String(id),
			Updates:             diffWafSizeConstraints(oldS, newS),
		}

		log.Printf("[INFO] Updating WAF Size Constraint constraints: %s", req)
		return conn.UpdateSizeConstraintSet(req)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
	}

	return nil
}

func flattenWafSizeConstraints(sc []*waf.SizeConstraint) []interface{} {
	out := make([]interface{}, len(sc), len(sc))
	for i, c := range sc {
		m := make(map[string]interface{})
		m["comparison_operator"] = *c.ComparisonOperator
		if c.FieldToMatch != nil {
			m["field_to_match"] = flattenFieldToMatch(c.FieldToMatch)
		}
		m["size"] = *c.Size
		m["text_transformation"] = *c.TextTransformation
		out[i] = m
	}
	return out
}

func diffWafSizeConstraints(oldS, newS []interface{}) []*waf.SizeConstraintSetUpdate {
	updates := make([]*waf.SizeConstraintSetUpdate, 0)

	for _, os := range oldS {
		constraint := os.(map[string]interface{})

		if idx, contains := sliceContainsMap(newS, constraint); contains {
			newS = append(newS[:idx], newS[idx+1:]...)
			continue
		}

		updates = append(updates, &waf.SizeConstraintSetUpdate{
			Action: aws.String(waf.ChangeActionDelete),
			SizeConstraint: &waf.SizeConstraint{
				FieldToMatch:       expandFieldToMatch(constraint["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				ComparisonOperator: aws.String(constraint["comparison_operator"].(string)),
				Size:               aws.Int64(int64(constraint["size"].(int))),
				TextTransformation: aws.String(constraint["text_transformation"].(string)),
			},
		})
	}

	for _, ns := range newS {
		constraint := ns.(map[string]interface{})

		updates = append(updates, &waf.SizeConstraintSetUpdate{
			Action: aws.String(waf.ChangeActionInsert),
			SizeConstraint: &waf.SizeConstraint{
				FieldToMatch:       expandFieldToMatch(constraint["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				ComparisonOperator: aws.String(constraint["comparison_operator"].(string)),
				Size:               aws.Int64(int64(constraint["size"].(int))),
				TextTransformation: aws.String(constraint["text_transformation"].(string)),
			},
		})
	}
	return updates
}
