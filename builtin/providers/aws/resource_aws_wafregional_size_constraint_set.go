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

func resourceAwsWafRegionalSizeConstraintSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalSizeConstraintSetCreate,
		Read:   resourceAwsWafRegionalSizeConstraintSetRead,
		Update: resourceAwsWafRegionalSizeConstraintSetUpdate,
		Delete: resourceAwsWafRegionalSizeConstraintSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size_constraint": &schema.Schema{
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

func resourceAwsWafRegionalSizeConstraintSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Creating SizeConstraintSet: %s", d.Get("name").(string))

	wr := newWafRegionalRetryer(conn, region)
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

	return resourceAwsWafRegionalSizeConstraintSetUpdate(d, meta)
}

func resourceAwsWafRegionalSizeConstraintSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	log.Printf("[INFO] Reading SizeConstraintSet: %s", d.Get("name").(string))
	params := &waf.GetSizeConstraintSetInput{
		SizeConstraintSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetSizeConstraintSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	var constraints []map[string]interface{}

	for _, constraint := range resp.SizeConstraintSet.SizeConstraints {
		field_to_match := constraint.FieldToMatch
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

		constraint := map[string]interface{}{
			"comparison_operator": *constraint.ComparisonOperator,
			"field_to_match":      ms,
			"size":                *constraint.Size,
			"text_transformation": *constraint.TextTransformation,
		}
		constraints = append(constraints, constraint)
	}

	d.Set("size_constraint", constraints)
	d.Set("name", resp.SizeConstraintSet.Name)

	return nil
}

func resourceAwsWafRegionalSizeConstraintSetUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating SizeConstraintSet: %s", d.Get("name").(string))
	if d.HasChange("size_constraint") {
		conn := meta.(*AWSClient).wafregionalconn
		region := meta.(*AWSClient).region

		o, n := d.GetChange("size_constraint")
		oldD, newD := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateSizeConstraintSetResourceWR(d, oldD, newD, conn, region)

		if err != nil {
			return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
		}
	}
	return resourceAwsWafRegionalSizeConstraintSetRead(d, meta)
}

func resourceAwsWafRegionalSizeConstraintSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Deleting SizeConstraintSet: %s", d.Get("name").(string))
	oldD := d.Get("size_constraint").(*schema.Set).List()

	if len(oldD) > 0 {
		var newD []interface{}
		err := updateSizeConstraintSetResourceWR(d, oldD, newD, conn, region)

		if err != nil {
			return errwrap.Wrapf("[ERROR] Error deleting SizeConstraintSet: {{err}}", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
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

func updateSizeConstraintSetResourceWR(d *schema.ResourceData, oldD, newD []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateSizeConstraintSetInput{
			ChangeToken:         token,
			SizeConstraintSetId: aws.String(d.Id()),
			Updates:             diffWafRegionalSizeConstraint(oldD, newD),
		}

		return conn.UpdateSizeConstraintSet(req)
	})
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
	}

	return nil
}

func diffWafRegionalSizeConstraint(oldD, newD []interface{}) []*waf.SizeConstraintSetUpdate {
	var updates []*waf.SizeConstraintSetUpdate

	for _, od := range oldD {
		sizeConstraint := od.(map[string]interface{})

		if idx, contains := sliceContainsMap(newD, sizeConstraint); contains {
			newD = append(newD[:idx], newD[idx+1:]...)
			continue
		}

		updates = append(updates, &waf.SizeConstraintSetUpdate{
			Action: aws.String(waf.ChangeActionDelete),
			SizeConstraint: &waf.SizeConstraint{
				FieldToMatch:       expandFieldToMatch(sizeConstraint["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				ComparisonOperator: aws.String(sizeConstraint["comparison_operator"].(string)),
				Size:               aws.Int64(int64(sizeConstraint["size"].(int))),
				TextTransformation: aws.String(sizeConstraint["text_transformation"].(string)),
			},
		})
	}

	for _, nd := range newD {
		sizeConstraint := nd.(map[string]interface{})

		updates = append(updates, &waf.SizeConstraintSetUpdate{
			Action: aws.String(waf.ChangeActionInsert),
			SizeConstraint: &waf.SizeConstraint{
				FieldToMatch:       expandFieldToMatch(sizeConstraint["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				ComparisonOperator: aws.String(sizeConstraint["comparison_operator"].(string)),
				Size:               aws.Int64(int64(sizeConstraint["size"].(int))),
				TextTransformation: aws.String(sizeConstraint["text_transformation"].(string)),
			},
		})
	}

	return updates
}
