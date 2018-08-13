package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalGeoMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalGeoMatchSetCreate,
		Read:   resourceAwsWafRegionalGeoMatchSetRead,
		Update: resourceAwsWafRegionalGeoMatchSetUpdate,
		Delete: resourceAwsWafRegionalGeoMatchSetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"geo_match_constraint": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRegionalGeoMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	log.Printf("[INFO] Creating WAF Regional Geo Match Set: %s", d.Get("name").(string))

	wr := newWafRegionalRetryer(conn, region)
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateGeoMatchSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateGeoMatchSet(params)
	})
	if err != nil {
		return fmt.Errorf("Failed creating WAF Regional Geo Match Set: %s", err)
	}
	resp := out.(*waf.CreateGeoMatchSetOutput)

	d.SetId(*resp.GeoMatchSet.GeoMatchSetId)

	return resourceAwsWafRegionalGeoMatchSetUpdate(d, meta)
}

func resourceAwsWafRegionalGeoMatchSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	log.Printf("[INFO] Reading WAF Regional Geo Match Set: %s", d.Get("name").(string))
	params := &waf.GetGeoMatchSetInput{
		GeoMatchSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetGeoMatchSet(params)
	if err != nil {
		// TODO: Replace with constant once it's available
		// See https://github.com/aws/aws-sdk-go/issues/1856
		if isAWSErr(err, "WAFNonexistentItemException", "") {
			log.Printf("[WARN] WAF WAF Regional Geo Match Set (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.GeoMatchSet.Name)
	d.Set("geo_match_constraint", flattenWafGeoMatchConstraint(resp.GeoMatchSet.GeoMatchConstraints))

	return nil
}

func resourceAwsWafRegionalGeoMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	if d.HasChange("geo_match_constraint") {
		o, n := d.GetChange("geo_match_constraint")
		oldConstraints, newConstraints := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateGeoMatchSetResourceWR(d.Id(), oldConstraints, newConstraints, conn, region)
		if err != nil {
			return fmt.Errorf("Failed updating WAF Regional Geo Match Set: %s", err)
		}
	}

	return resourceAwsWafRegionalGeoMatchSetRead(d, meta)
}

func resourceAwsWafRegionalGeoMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn
	region := meta.(*AWSClient).region

	oldConstraints := d.Get("geo_match_constraint").(*schema.Set).List()
	if len(oldConstraints) > 0 {
		noConstraints := []interface{}{}
		err := updateGeoMatchSetResourceWR(d.Id(), oldConstraints, noConstraints, conn, region)
		if err != nil {
			return fmt.Errorf("Error updating WAF Regional Geo Match Constraint: %s", err)
		}
	}

	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteGeoMatchSetInput{
			ChangeToken:   token,
			GeoMatchSetId: aws.String(d.Id()),
		}

		return conn.DeleteGeoMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed deleting WAF Regional Geo Match Set: %s", err)
	}

	return nil
}

func updateGeoMatchSetResourceWR(id string, oldConstraints, newConstraints []interface{}, conn *wafregional.WAFRegional, region string) error {
	wr := newWafRegionalRetryer(conn, region)
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateGeoMatchSetInput{
			ChangeToken:   token,
			GeoMatchSetId: aws.String(id),
			Updates:       diffWafGeoMatchSetConstraints(oldConstraints, newConstraints),
		}

		log.Printf("[INFO] Updating WAF Regional Geo Match Set constraints: %s", req)
		return conn.UpdateGeoMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Failed updating WAF Regional Geo Match Set: %s", err)
	}

	return nil
}
