package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafSizeConstraintSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafSizeConstraintSetCreate,
		Read:   resourceAwsWafSizeConstraintSetRead,
		Update: resourceAwsWafSizeConstraintSetUpdate,
		Delete: resourceAwsWafSizeConstraintSetDelete,

		Schema: wafSizeConstraintSetSchema(),
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
		return fmt.Errorf("Error creating SizeConstraintSet: %s", err)
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
			log.Printf("[WARN] WAF SizeConstraintSet (%s) not found, removing from state", d.Id())
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
		oldConstraints, newConstraints := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateSizeConstraintSetResource(d.Id(), oldConstraints, newConstraints, conn)
		if err != nil {
			return fmt.Errorf("Error updating SizeConstraintSet: %s", err)
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
			return fmt.Errorf("Error deleting SizeConstraintSet: %s", err)
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
		return fmt.Errorf("Error deleting SizeConstraintSet: %s", err)
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
		return fmt.Errorf("Error updating SizeConstraintSet: %s", err)
	}

	return nil
}
