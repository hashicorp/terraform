package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalRegexMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalRegexMatchSetCreate,
		Read:   resourceAwsWafRegionalRegexMatchSetRead,
		Update: resourceAwsWafRegionalRegexMatchSetUpdate,
		Delete: resourceAwsWafRegionalRegexMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"regex_match_tuple": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      resourceAwsWafRegexMatchSetTupleHash,
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
										StateFunc: func(v interface{}) string {
											return strings.ToLower(v.(string))
										},
									},
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"regex_pattern_set_id": {
							Type:     schema.TypeString,
							Required: true,
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

func resourceAwsWafRegionalRegexMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Creating WAF Regional Regex Match Set: %s", d.Get("name").(string))

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateRegexMatchSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}
		return conn.CreateRegexMatchSet(params)
	})
	if err != nil {
		return fmt.Errorf("Failed creating WAF Regional Regex Match Set: %s", err)
	}
	resp := out.(*waf.CreateRegexMatchSetOutput)

	d.SetId(*resp.RegexMatchSet.RegexMatchSetId)

	return resourceAwsWafRegionalRegexMatchSetUpdate(d, meta)
}

func resourceAwsWafRegionalRegexMatchSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	log.Printf("[INFO] Reading WAF Regional Regex Match Set: %s", d.Get("name").(string))
	params := &waf.GetRegexMatchSetInput{
		RegexMatchSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetRegexMatchSet(params)
	if err != nil {
		if isAWSErr(err, wafregional.ErrCodeWAFNonexistentItemException, "") {
			log.Printf("[WARN] WAF Regional Regex Match Set (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.RegexMatchSet.Name)
	d.Set("regex_match_tuple", flattenWafRegexMatchTuples(resp.RegexMatchSet.RegexMatchTuples))

	return nil
}

func resourceAwsWafRegionalRegexMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Updating WAF Regional Regex Match Set: %s", d.Get("name").(string))

	if d.HasChange("regex_match_tuple") {
		o, n := d.GetChange("regex_match_tuple")
		oldT, newT := o.(*schema.Set).List(), n.(*schema.Set).List()
		err := updateRegexMatchSetResourceWR(d.Id(), oldT, newT, conn, region)
		if err != nil {
			return fmt.Errorf("Failed updating WAF Regional Regex Match Set: %s", err)
		}
	}

	return resourceAwsWafRegionalRegexMatchSetRead(d, meta)
}

func resourceAwsWafRegionalRegexMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	oldTuples := d.Get("regex_match_tuple").(*schema.Set).List()
	if len(oldTuples) > 0 {
		noTuples := []interface{}{}
		err := updateRegexMatchSetResourceWR(d.Id(), oldTuples, noTuples, conn, region)
		if err != nil {
			return fmt.Errorf("Error updating WAF Regional Regex Match Set: %s", err)
		}
	}

	wr := newWafRegionalRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteRegexMatchSetInput{
			ChangeToken:     token,
			RegexMatchSetId: aws.String(d.Id()),
		}
		log.Printf("[INFO] Deleting WAF Regional Regex Match Set: %s", req)
		return conn.DeleteRegexMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed deleting WAF Regional Regex Match Set: %s", err)
	}

	return nil
}

func updateRegexMatchSetResourceWR(id string, oldT, newT []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateRegexMatchSetInput{
			ChangeToken:     token,
			RegexMatchSetId: aws.String(id),
			Updates:         diffWafRegexMatchSetTuples(oldT, newT),
		}

		return conn.UpdateRegexMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed updating WAF Regional Regex Match Set: %s", err)
	}

	return nil
}
