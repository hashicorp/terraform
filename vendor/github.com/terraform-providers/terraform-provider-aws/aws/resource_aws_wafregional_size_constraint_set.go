package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalSizeConstraintSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalSizeConstraintSetCreate,
		Read:   resourceAwsWafRegionalSizeConstraintSetRead,
		Update: resourceAwsWafRegionalSizeConstraintSetUpdate,
		Delete: resourceAwsWafRegionalSizeConstraintSetDelete,

		Schema: wafSizeConstraintSetSchema(),
	}
}

func resourceAwsWafRegionalSizeConstraintSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	name := d.Get("name").(string)

	log.Printf("[INFO] Creating WAF Regional SizeConstraintSet: %s", name)

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateSizeConstraintSetInput{
			ChangeToken: token,
			Name:        aws.String(name),
		}

		return conn.CreateSizeConstraintSet(params)
	})
	if err != nil {
		return fmt.Errorf("[ERROR] Error creating WAF Regional SizeConstraintSet: %s", err)
	}
	resp := out.(*waf.CreateSizeConstraintSetOutput)

	d.SetId(*resp.SizeConstraintSet.SizeConstraintSetId)

	return resourceAwsWafRegionalSizeConstraintSetUpdate(d, meta)
}

func resourceAwsWafRegionalSizeConstraintSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	log.Printf("[INFO] Reading WAF Regional SizeConstraintSet: %s", d.Get("name").(string))
	params := &waf.GetSizeConstraintSetInput{
		SizeConstraintSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetSizeConstraintSet(params)
	if err != nil {
		if isAWSErr(err, wafregional.ErrCodeWAFNonexistentItemException, "") {
			log.Printf("[WARN] WAF Regional SizeConstraintSet (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", resp.SizeConstraintSet.Name)
	d.Set("size_constraints", flattenWafSizeConstraints(resp.SizeConstraintSet.SizeConstraints))

	return nil
}

func resourceAwsWafRegionalSizeConstraintSetUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient)

	if d.HasChange("size_constraints") {
		o, n := d.GetChange("size_constraints")
		oldConstraints, newConstraints := o.(*schema.Set).List(), n.(*schema.Set).List()

		if err := updateRegionalSizeConstraintSetResource(d.Id(), oldConstraints, newConstraints, client.wafregionalconn, client.region); err != nil {
			return fmt.Errorf("[ERROR] Error updating WAF Regional SizeConstraintSet: %s", err)
		}
	}

	return resourceAwsWafRegionalSizeConstraintSetRead(d, meta)
}

func resourceAwsWafRegionalSizeConstraintSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	oldConstraints := d.Get("size_constraints").(*schema.Set).List()

	if len(oldConstraints) > 0 {
		noConstraints := []interface{}{}
		if err := updateRegionalSizeConstraintSetResource(d.Id(), oldConstraints, noConstraints, conn, region); err != nil {
			return fmt.Errorf("[ERROR] Error deleting WAF Regional SizeConstraintSet: %s", err)
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
		return fmt.Errorf("[ERROR] Error deleting WAF Regional SizeConstraintSet: %s", err)
	}

	return nil
}

func updateRegionalSizeConstraintSetResource(id string, oldConstraints, newConstraints []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateSizeConstraintSetInput{
			ChangeToken:         token,
			SizeConstraintSetId: aws.String(id),
			Updates:             diffWafSizeConstraints(oldConstraints, newConstraints),
		}

		log.Printf("[INFO] Updating WAF Regional SizeConstraintSet: %s", req)
		return conn.UpdateSizeConstraintSet(req)
	})
	if err != nil {
		return fmt.Errorf("[ERROR] Error updating WAF Regional SizeConstraintSet: %s", err)
	}

	return nil
}
