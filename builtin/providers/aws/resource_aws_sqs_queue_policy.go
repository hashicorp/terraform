package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSqsQueuePolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSqsQueuePolicyUpsert,
		Read:   resourceAwsSqsQueuePolicyRead,
		Update: resourceAwsSqsQueuePolicyUpsert,
		Delete: resourceAwsSqsQueuePolicyDelete,

		Schema: map[string]*schema.Schema{
			"queue_url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
		},
	}
}

func resourceAwsSqsQueuePolicyUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn
	url := d.Get("queue_url").(string)

	_, err := conn.SetQueueAttributes(&sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(url),
		Attributes: aws.StringMap(map[string]string{
			"Policy": d.Get("policy").(string),
		}),
	})
	if err != nil {
		return fmt.Errorf("Error updating SQS attributes: %s", err)
	}

	d.SetId("sqs-policy-" + url)

	return resourceAwsSqsQueuePolicyRead(d, meta)
}

func resourceAwsSqsQueuePolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn
	url := d.Get("queue_url").(string)
	out, err := conn.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(url),
		AttributeNames: []*string{aws.String("Policy")},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "AWS.SimpleQueueService.NonExistentQueue" {
			log.Printf("[WARN] SQS Queue (%s) not found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	if out == nil {
		return fmt.Errorf("Received empty response for SQS queue %s", d.Id())
	}

	policy, ok := out.Attributes["Policy"]
	if !ok {
		return fmt.Errorf("SQS Queue policy not found for %s", d.Id())
	}

	d.Set("policy", policy)

	return nil
}

func resourceAwsSqsQueuePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn

	url := d.Get("queue_url").(string)
	log.Printf("[DEBUG] Deleting SQS Queue Policy of %s", url)
	_, err := conn.SetQueueAttributes(&sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(url),
		Attributes: aws.StringMap(map[string]string{
			"Policy": "",
		}),
	})
	if err != nil {
		return fmt.Errorf("Error deleting SQS Queue policy: %s", err)
	}
	return nil
}
