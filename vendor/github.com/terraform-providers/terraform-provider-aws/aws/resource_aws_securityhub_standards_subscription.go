package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSecurityHubStandardsSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityHubStandardsSubscriptionCreate,
		Read:   resourceAwsSecurityHubStandardsSubscriptionRead,
		Delete: resourceAwsSecurityHubStandardsSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"standards_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
		},
	}
}

func resourceAwsSecurityHubStandardsSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn
	log.Printf("[DEBUG] Enabling Security Hub standard %s", d.Get("standards_arn"))

	resp, err := conn.BatchEnableStandards(&securityhub.BatchEnableStandardsInput{
		StandardsSubscriptionRequests: []*securityhub.StandardsSubscriptionRequest{
			{
				StandardsArn: aws.String(d.Get("standards_arn").(string)),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("Error enabling Security Hub standard: %s", err)
	}

	standardsSubscription := resp.StandardsSubscriptions[0]

	d.SetId(*standardsSubscription.StandardsSubscriptionArn)

	return resourceAwsSecurityHubStandardsSubscriptionRead(d, meta)
}

func resourceAwsSecurityHubStandardsSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn

	log.Printf("[DEBUG] Reading Security Hub standard %s", d.Id())
	resp, err := conn.GetEnabledStandards(&securityhub.GetEnabledStandardsInput{
		StandardsSubscriptionArns: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return fmt.Errorf("Error reading Security Hub standard %s: %s", d.Id(), err)
	}

	if len(resp.StandardsSubscriptions) == 0 {
		log.Printf("[WARN] Security Hub standard (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	standardsSubscription := resp.StandardsSubscriptions[0]

	d.Set("standards_arn", standardsSubscription.StandardsArn)

	return nil
}

func resourceAwsSecurityHubStandardsSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn
	log.Printf("[DEBUG] Disabling Security Hub standard %s", d.Id())

	_, err := conn.BatchDisableStandards(&securityhub.BatchDisableStandardsInput{
		StandardsSubscriptionArns: []*string{aws.String(d.Id())},
	})

	if err != nil {
		return fmt.Errorf("Error disabling Security Hub standard %s: %s", d.Id(), err)
	}

	return nil
}
