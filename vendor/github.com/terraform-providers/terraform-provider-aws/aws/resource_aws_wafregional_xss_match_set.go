package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalXssMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalXssMatchSetCreate,
		Read:   resourceAwsWafRegionalXssMatchSetRead,
		Update: resourceAwsWafRegionalXssMatchSetUpdate,
		Delete: resourceAwsWafRegionalXssMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"xss_match_tuple": {
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

func resourceAwsWafRegionalXssMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Creating regional WAF XSS Match Set: %s", d.Get("name").(string))

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateXssMatchSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateXssMatchSet(params)
	})
	if err != nil {
		return fmt.Errorf("Failed creating regional WAF XSS Match Set: %s", err)
	}
	resp := out.(*waf.CreateXssMatchSetOutput)

	d.SetId(*resp.XssMatchSet.XssMatchSetId)

	return resourceAwsWafRegionalXssMatchSetUpdate(d, meta)
}

func resourceAwsWafRegionalXssMatchSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	log.Printf("[INFO] Reading regional WAF XSS Match Set: %s", d.Get("name").(string))
	params := &waf.GetXssMatchSetInput{
		XssMatchSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetXssMatchSet(params)
	if err != nil {
		if isAWSErr(err, wafregional.ErrCodeWAFNonexistentItemException, "") {
			log.Printf("[WARN] Regional WAF XSS Match Set (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	set := resp.XssMatchSet

	d.Set("xss_match_tuple", flattenWafXssMatchTuples(set.XssMatchTuples))
	d.Set("name", set.Name)

	return nil
}

func resourceAwsWafRegionalXssMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	if d.HasChange("xss_match_tuple") {
		o, n := d.GetChange("xss_match_tuple")
		oldT, newT := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateXssMatchSetResourceWR(d.Id(), oldT, newT, conn, region)
		if err != nil {
			return fmt.Errorf("Failed updating regional WAF XSS Match Set: %s", err)
		}
	}

	return resourceAwsWafRegionalXssMatchSetRead(d, meta)
}

func resourceAwsWafRegionalXssMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	if v, ok := d.GetOk("xss_match_tuple"); ok {
		oldTuples := v.(*schema.Set).List()
		if len(oldTuples) > 0 {
			noTuples := []interface{}{}
			err := updateXssMatchSetResourceWR(d.Id(), oldTuples, noTuples, conn, region)
			if err != nil {
				return fmt.Errorf("Error updating regional WAF XSS Match Set: %s", err)
			}
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteXssMatchSetInput{
			ChangeToken:   token,
			XssMatchSetId: aws.String(d.Id()),
		}

		return conn.DeleteXssMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed deleting regional WAF XSS Match Set: %s", err)
	}

	return nil
}

func updateXssMatchSetResourceWR(id string, oldT, newT []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateXssMatchSetInput{
			ChangeToken:   token,
			XssMatchSetId: aws.String(id),
			Updates:       diffWafXssMatchSetTuples(oldT, newT),
		}

		log.Printf("[INFO] Updating XSS Match Set tuples: %s", req)
		return conn.UpdateXssMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed updating regional WAF XSS Match Set: %s", err)
	}

	return nil
}
