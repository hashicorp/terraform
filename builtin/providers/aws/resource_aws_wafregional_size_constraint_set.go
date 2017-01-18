package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
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
			"size_constraints": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
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

	log.Printf("[INFO] Creating SizeConstraintSet: %s", d.Get("name").(string))

	// ChangeToken
	var ct *waf.GetChangeTokenInput

	res, err := conn.GetChangeToken(ct)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error getting change token: {{err}}", err)
	}

	params := &waf.CreateSizeConstraintSetInput{
		ChangeToken: res.ChangeToken,
		Name:        aws.String(d.Get("name").(string)),
	}

	resp, err := conn.CreateSizeConstraintSet(params)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SizeConstraintSet: {{err}}", err)
	}

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

	d.Set("name", resp.SizeConstraintSet.Name)

	return nil
}

func resourceAwsWafRegionalSizeConstraintSetUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating SizeConstraintSet: %s", d.Get("name").(string))
	err := updateSizeConstraintSetResourceWR(d, meta, waf.ChangeActionInsert)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
	}
	return resourceAwsWafRegionalSizeConstraintSetRead(d, meta)
}

func resourceAwsWafRegionalSizeConstraintSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	log.Printf("[INFO] Deleting SizeConstraintSet: %s", d.Get("name").(string))
	err := updateSizeConstraintSetResourceWR(d, meta, waf.ChangeActionDelete)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SizeConstraintSet: {{err}}", err)
	}

	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)

	req := &waf.DeleteSizeConstraintSetInput{
		ChangeToken:         resp.ChangeToken,
		SizeConstraintSetId: aws.String(d.Id()),
	}

	_, err = conn.DeleteSizeConstraintSet(req)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SizeConstraintSet: {{err}}", err)
	}

	return nil
}

func updateSizeConstraintSetResourceWR(d *schema.ResourceData, meta interface{}, ChangeAction string) error {
	conn := meta.(*AWSClient).wafregionalconn

	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error getting change token: {{err}}", err)
	}

	req := &waf.UpdateSizeConstraintSetInput{
		ChangeToken:         resp.ChangeToken,
		SizeConstraintSetId: aws.String(d.Id()),
	}

	sizeConstraints := d.Get("size_constraints").(*schema.Set)
	for _, sizeConstraint := range sizeConstraints.List() {
		sc := sizeConstraint.(map[string]interface{})
		sizeConstraintUpdate := &waf.SizeConstraintSetUpdate{
			Action: aws.String(ChangeAction),
			SizeConstraint: &waf.SizeConstraint{
				FieldToMatch:       expandFieldToMatch(sc["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				ComparisonOperator: aws.String(sc["comparison_operator"].(string)),
				Size:               aws.Int64(int64(sc["size"].(int))),
				TextTransformation: aws.String(sc["text_transformation"].(string)),
			},
		}
		req.Updates = append(req.Updates, sizeConstraintUpdate)
	}

	_, err = conn.UpdateSizeConstraintSet(req)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
	}

	return nil
}
