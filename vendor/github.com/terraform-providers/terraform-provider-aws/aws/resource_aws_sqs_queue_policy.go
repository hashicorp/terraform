package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/jen20/awspolicyequivalence"
)

func resourceAwsSqsQueuePolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSqsQueuePolicyUpsert,
		Read:   resourceAwsSqsQueuePolicyRead,
		Update: resourceAwsSqsQueuePolicyUpsert,
		Delete: resourceAwsSqsQueuePolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		MigrateState:  resourceAwsSqsQueuePolicyMigrateState,
		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"queue_url": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
		},
	}
}

func resourceAwsSqsQueuePolicyUpsert(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn
	policy := d.Get("policy").(string)
	url := d.Get("queue_url").(string)

	sqaInput := &sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(url),
		Attributes: aws.StringMap(map[string]string{
			sqs.QueueAttributeNamePolicy: policy,
		}),
	}
	log.Printf("[DEBUG] Updating SQS attributes: %s", sqaInput)
	_, err := conn.SetQueueAttributes(sqaInput)
	if err != nil {
		return fmt.Errorf("Error updating SQS attributes: %s", err)
	}

	// https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_SetQueueAttributes.html
	// When you change a queue's attributes, the change can take up to 60 seconds
	// for most of the attributes to propagate throughout the Amazon SQS system.
	gqaInput := &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(url),
		AttributeNames: []*string{aws.String(sqs.QueueAttributeNamePolicy)},
	}
	notUpdatedError := fmt.Errorf("SQS attribute %s not updated", sqs.QueueAttributeNamePolicy)
	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Reading SQS attributes: %s", gqaInput)
		out, err := conn.GetQueueAttributes(gqaInput)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		queuePolicy, ok := out.Attributes[sqs.QueueAttributeNamePolicy]
		if !ok {
			log.Printf("[DEBUG] SQS attribute %s not found - retrying", sqs.QueueAttributeNamePolicy)
			return resource.RetryableError(notUpdatedError)
		}
		equivalent, err := awspolicy.PoliciesAreEquivalent(*queuePolicy, policy)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if !equivalent {
			log.Printf("[DEBUG] SQS attribute %s not updated - retrying", sqs.QueueAttributeNamePolicy)
			return resource.RetryableError(notUpdatedError)
		}
		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(url)

	return resourceAwsSqsQueuePolicyRead(d, meta)
}

func resourceAwsSqsQueuePolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn

	out, err := conn.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(d.Id()),
		AttributeNames: []*string{aws.String(sqs.QueueAttributeNamePolicy)},
	})
	if err != nil {
		if isAWSErr(err, "AWS.SimpleQueueService.NonExistentQueue", "") {
			log.Printf("[WARN] SQS Queue (%s) not found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	if out == nil {
		return fmt.Errorf("Received empty response for SQS queue %s", d.Id())
	}

	policy, ok := out.Attributes[sqs.QueueAttributeNamePolicy]
	if ok {
		d.Set("policy", policy)
	} else {
		d.Set("policy", "")
	}

	d.Set("queue_url", d.Id())

	return nil
}

func resourceAwsSqsQueuePolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn

	log.Printf("[DEBUG] Deleting SQS Queue Policy of %s", d.Id())
	_, err := conn.SetQueueAttributes(&sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(d.Id()),
		Attributes: aws.StringMap(map[string]string{
			sqs.QueueAttributeNamePolicy: "",
		}),
	})
	if err != nil {
		return fmt.Errorf("Error deleting SQS Queue policy: %s", err)
	}
	return nil
}
