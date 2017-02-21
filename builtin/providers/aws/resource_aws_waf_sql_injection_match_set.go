package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafSqlInjectionMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafSqlInjectionMatchSetCreate,
		Read:   resourceAwsWafSqlInjectionMatchSetRead,
		Update: resourceAwsWafSqlInjectionMatchSetUpdate,
		Delete: resourceAwsWafSqlInjectionMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"sql_injection_match_tuples": &schema.Schema{
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

func resourceAwsWafSqlInjectionMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Creating SqlInjectionMatchSet: %s", d.Get("name").(string))

	// ChangeToken
	var ct *waf.GetChangeTokenInput

	res, err := conn.GetChangeToken(ct)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error getting change token: {{err}}", err)
	}

	params := &waf.CreateSqlInjectionMatchSetInput{
		ChangeToken: res.ChangeToken,
		Name:        aws.String(d.Get("name").(string)),
	}

	resp, err := conn.CreateSqlInjectionMatchSet(params)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error creating SqlInjectionMatchSet: {{err}}", err)
	}

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
			log.Printf("[WARN] WAF IPSet (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.SqlInjectionMatchSet.Name)

	return nil
}

func resourceAwsWafSqlInjectionMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating SqlInjectionMatchSet: %s", d.Get("name").(string))
	err := updateSqlInjectionMatchSetResource(d, meta, waf.ChangeActionInsert)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SqlInjectionMatchSet: {{err}}", err)
	}
	return resourceAwsWafSqlInjectionMatchSetRead(d, meta)
}

func resourceAwsWafSqlInjectionMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Deleting SqlInjectionMatchSet: %s", d.Get("name").(string))
	err := updateSqlInjectionMatchSetResource(d, meta, waf.ChangeActionDelete)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SqlInjectionMatchSet: {{err}}", err)
	}

	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)

	req := &waf.DeleteSqlInjectionMatchSetInput{
		ChangeToken:            resp.ChangeToken,
		SqlInjectionMatchSetId: aws.String(d.Id()),
	}

	_, err = conn.DeleteSqlInjectionMatchSet(req)

	if err != nil {
		return errwrap.Wrapf("[ERROR] Error deleting SqlInjectionMatchSet: {{err}}", err)
	}

	return nil
}

func updateSqlInjectionMatchSetResource(d *schema.ResourceData, meta interface{}, ChangeAction string) error {
	conn := meta.(*AWSClient).wafconn

	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error getting change token: {{err}}", err)
	}

	req := &waf.UpdateSqlInjectionMatchSetInput{
		ChangeToken:            resp.ChangeToken,
		SqlInjectionMatchSetId: aws.String(d.Id()),
	}

	sqlInjectionMatchTuples := d.Get("sql_injection_match_tuples").(*schema.Set)
	for _, sqlInjectionMatchTuple := range sqlInjectionMatchTuples.List() {
		simt := sqlInjectionMatchTuple.(map[string]interface{})
		sizeConstraintUpdate := &waf.SqlInjectionMatchSetUpdate{
			Action: aws.String(ChangeAction),
			SqlInjectionMatchTuple: &waf.SqlInjectionMatchTuple{
				FieldToMatch:       expandFieldToMatch(simt["field_to_match"].(*schema.Set).List()[0].(map[string]interface{})),
				TextTransformation: aws.String(simt["text_transformation"].(string)),
			},
		}
		req.Updates = append(req.Updates, sizeConstraintUpdate)
	}

	_, err = conn.UpdateSqlInjectionMatchSet(req)
	if err != nil {
		return errwrap.Wrapf("[ERROR] Error updating SqlInjectionMatchSet: {{err}}", err)
	}

	return nil
}
