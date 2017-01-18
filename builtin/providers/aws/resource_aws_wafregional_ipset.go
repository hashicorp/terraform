package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsWafRegionalIPSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsWafRegionalIPSetCreate,
		Read:   resourceAwsWafRegionalIPSetRead,
		Update: resourceAwsWafRegionalIPSetUpdate,
		Delete: resourceAwsWafRegionalIPSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ip_set_descriptors": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsWafRegionalIPSetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	// ChangeToken
	var ct *waf.GetChangeTokenInput

	res, err := conn.GetChangeToken(ct)
	if err != nil {
		return fmt.Errorf("Error getting change token: %s", err)
	}

	params := &waf.CreateIPSetInput{
		ChangeToken: res.ChangeToken,
		Name:        aws.String(d.Get("name").(string)),
	}

	resp, err := conn.CreateIPSet(params)
	if err != nil {
		return err
	}
	d.SetId(*resp.IPSet.IPSetId)
	return resourceAwsWafRegionalIPSetUpdate(d, meta)
}

func resourceAwsWafRegionalIPSetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	params := &waf.GetIPSetInput{
		IPSetId: aws.String(d.Id()),
	}

	resp, err := conn.GetIPSet(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "WAFNonexistentItemException" {
			log.Printf("[WARN] WAF IPSet (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	var IPSetDescriptors []map[string]interface{}

	for _, IPSetDescriptor := range resp.IPSet.IPSetDescriptors {
		IPSet := map[string]interface{}{
			"type":  *IPSetDescriptor.Type,
			"value": *IPSetDescriptor.Value,
		}
		IPSetDescriptors = append(IPSetDescriptors, IPSet)
	}

	d.Set("ip_set_descriptors", IPSetDescriptors)

	d.Set("name", resp.IPSet.Name)

	return nil
}

func resourceAwsWafRegionalIPSetUpdate(d *schema.ResourceData, meta interface{}) error {
	err := updateIPSetResourceWR(d, meta, waf.ChangeActionInsert)
	if err != nil {
		return fmt.Errorf("Error Updating WAF IPSet: %s", err)
	}
	return resourceAwsWafRegionalIPSetRead(d, meta)
}

func resourceAwsWafRegionalIPSetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).wafregionalconn

	err := updateIPSetResourceWR(d, meta, waf.ChangeActionDelete)
	if err != nil {
		return fmt.Errorf("Error Removing IPSetDescriptors: %s", err)
	}

	// ChangeToken
	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)

	req := &waf.DeleteIPSetInput{
		ChangeToken: resp.ChangeToken,
		IPSetId:     aws.String(d.Id()),
	}
	log.Printf("[INFO] Deleting WAF IPSet")
	_, err = conn.DeleteIPSet(req)

	if err != nil {
		return fmt.Errorf("Error Deleting WAF IPSet: %s", err)
	}

	return nil
}

func updateIPSetResourceWR(d *schema.ResourceData, meta interface{}, ChangeAction string) error {
	conn := meta.(*AWSClient).wafregionalconn

	// ChangeToken
	var ct *waf.GetChangeTokenInput

	resp, err := conn.GetChangeToken(ct)
	if err != nil {
		return fmt.Errorf("Error getting change token: %s", err)
	}

	req := &waf.UpdateIPSetInput{
		ChangeToken: resp.ChangeToken,
		IPSetId:     aws.String(d.Id()),
	}

	IPSetDescriptors := d.Get("ip_set_descriptors").(*schema.Set)
	for _, IPSetDescriptor := range IPSetDescriptors.List() {
		IPSet := IPSetDescriptor.(map[string]interface{})
		IPSetUpdate := &waf.IPSetUpdate{
			Action: aws.String(ChangeAction),
			IPSetDescriptor: &waf.IPSetDescriptor{
				Type:  aws.String(IPSet["type"].(string)),
				Value: aws.String(IPSet["value"].(string)),
			},
		}
		req.Updates = append(req.Updates, IPSetUpdate)
	}

	_, err = conn.UpdateIPSet(req)
	if err != nil {
		return fmt.Errorf("Error Updating WAF IPSet: %s", err)
	}

	return nil
}
