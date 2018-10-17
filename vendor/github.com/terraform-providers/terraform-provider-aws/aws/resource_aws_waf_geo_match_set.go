package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafGeoMatchSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafGeoMatchSetCreate,
		Read:   resourceAwsWafGeoMatchSetRead,
		Update: resourceAwsWafGeoMatchSetUpdate,
		Delete: resourceAwsWafGeoMatchSetDelete,

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

func resourceAwsWafGeoMatchSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	log.Printf("[INFO] Creating GeoMatchSet: %s", d.Get("name").(string))

	wr := newWafRetryer(conn, "global")
	out, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		params := &waf.CreateGeoMatchSetInput{
			ChangeToken: token,
			Name:        aws.String(d.Get("name").(string)),
		}

		return conn.CreateGeoMatchSet(params)
	})
	if err != nil {
		return fmt.Errorf("Error creating GeoMatchSet: %s", err)
	}
	resp := out.(*waf.CreateGeoMatchSetOutput)

	d.SetId(*resp.GeoMatchSet.GeoMatchSetId)

	return resourceAwsWafGeoMatchSetUpdate(d, meta)
}

func resourceAwsWafGeoMatchSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn
	log.Printf("[INFO] Reading GeoMatchSet: %s", d.Get("name").(string))
	params := &waf.GetGeoMatchSetInput{
		GeoMatchSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetGeoMatchSet(params)
	if err != nil {
		// TODO: Replace with constant once it's available
		// See https://github.com/aws/aws-sdk-go/issues/1856
		if isAWSErr(err, "WAFNonexistentItemException", "") {
			log.Printf("[WARN] WAF GeoMatchSet (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", resp.GeoMatchSet.Name)
	d.Set("geo_match_constraint", flattenWafGeoMatchConstraint(resp.GeoMatchSet.GeoMatchConstraints))

	return nil
}

func resourceAwsWafGeoMatchSetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	if d.HasChange("geo_match_constraint") {
		o, n := d.GetChange("geo_match_constraint")
		oldT, newT := o.(*schema.Set).List(), n.(*schema.Set).List()

		err := updateGeoMatchSetResource(d.Id(), oldT, newT, conn)
		if err != nil {
			return fmt.Errorf("Error updating GeoMatchSet: %s", err)
		}
	}

	return resourceAwsWafGeoMatchSetRead(d, meta)
}

func resourceAwsWafGeoMatchSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafconn

	oldConstraints := d.Get("geo_match_constraint").(*schema.Set).List()
	if len(oldConstraints) > 0 {
		noConstraints := []interface{}{}
		err := updateGeoMatchSetResource(d.Id(), oldConstraints, noConstraints, conn)
		if err != nil {
			return fmt.Errorf("Error updating GeoMatchConstraint: %s", err)
		}
	}

	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.DeleteGeoMatchSetInput{
			ChangeToken:   token,
			GeoMatchSetId: aws.String(d.Id()),
		}

		return conn.DeleteGeoMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Error deleting GeoMatchSet: %s", err)
	}

	return nil
}

func updateGeoMatchSetResource(id string, oldT, newT []interface{}, conn *waf.WAF) error {
	wr := newWafRetryer(conn, "global")
	_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
		req := &waf.UpdateGeoMatchSetInput{
			ChangeToken:   token,
			GeoMatchSetId: aws.String(id),
			Updates:       diffWafGeoMatchSetConstraints(oldT, newT),
		}

		log.Printf("[INFO] Updating GeoMatchSet constraints: %s", req)
		return conn.UpdateGeoMatchSet(req)
	})
	if err != nil {
		return fmt.Errorf("Error updating GeoMatchSet: %s", err)
	}

	return nil
}
