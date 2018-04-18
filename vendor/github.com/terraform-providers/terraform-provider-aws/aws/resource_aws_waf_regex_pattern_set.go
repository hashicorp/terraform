package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegexPatternSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegexPatternSetCreate,
		Read:   resourceAwsWafRegexPatternSetRead,
		Update: resourceAwsWafRegexPatternSetUpdate,
		Delete: resourceAwsWafRegexPatternSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"regex_pattern_strings": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceAwsWafRegexPatternSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Creating WAF Regex Pattern Set: %s", d.Get("name").(string))

	wr := newWafRetryer(conn, "global")
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateRegexPatternSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}
		return conn.CreateRegexPatternSet(params)
	})
	if err != nil {
		return fmt.Errorf("Failed creating WAF Regex Pattern Set: %s", err)
	}
	resp := out.(*waf.CreateRegexPatternSetOutput)

	d.SetId(*resp.RegexPatternSet.RegexPatternSetId)

	return resourceAwsWafRegexPatternSetUpdate(d, meta)
}

func resourceAwsWafRegexPatternSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	log.Printf("[INFO] Reading WAF Regex Pattern Set: %s", d.Get("name").(string))
	params := &waf.GetRegexPatternSetInput{
		RegexPatternSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetRegexPatternSet(params)
	if err != nil {
		// TODO: Replace with a constant once available
		// See https://github.com/aws/aws-sdk-go/issues/1856
		if isAWSErr(err, "WAFNonexistentItemException", "") {
			log.Printf("[WARN] WAF Regex Pattern Set (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.RegexPatternSet.Name)
	d.Set("regex_pattern_strings", aws.StringValueSlice(resp.RegexPatternSet.RegexPatternStrings))

	return nil
}

func resourceAwsWafRegexPatternSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Updating WAF Regex Pattern Set: %s", d.Get("name").(string))

	if d.HasChange("regex_pattern_strings") {
		o, n := d.GetChange("regex_pattern_strings")
		oldPatterns, newPatterns := o.(*schema.Set).List(), n.(*schema.Set).List()
		err := updateWafRegexPatternSetPatternStrings(d.Id(), oldPatterns, newPatterns, conn)
		if err != nil {
			return fmt.Errorf("Failed updating WAF Regex Pattern Set: %s", err)
		}
	}

	return resourceAwsWafRegexPatternSetRead(d, meta)
}

func resourceAwsWafRegexPatternSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldPatterns := d.Get("regex_pattern_strings").(*schema.Set).List()
	if len(oldPatterns) > 0 {
		noPatterns := []interface{}{}
		err := updateWafRegexPatternSetPatternStrings(d.Id(), oldPatterns, noPatterns, conn)
		if err != nil {
			return fmt.Errorf("Error updating WAF Regex Pattern Set: %s", err)
		}
	}

	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteRegexPatternSetInput{
			ChangeToken:       token,
			RegexPatternSetId: aws.String(d.Id()),
		}
		log.Printf("[INFO] Deleting WAF Regex Pattern Set: %s", req)
		return conn.DeleteRegexPatternSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed deleting WAF Regex Pattern Set: %s", err)
	}

	return nil
}

func updateWafRegexPatternSetPatternStrings(id string, oldPatterns, newPatterns []interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateRegexPatternSetInput{
			ChangeToken:       token,
			RegexPatternSetId: aws.String(id),
			Updates:           diffWafRegexPatternSetPatternStrings(oldPatterns, newPatterns),
		}

		return conn.UpdateRegexPatternSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed updating WAF Regex Pattern Set: %s", err)
	}

	return nil
}
