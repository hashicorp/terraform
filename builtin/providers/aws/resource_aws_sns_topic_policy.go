package aws

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
)

func resourceAwsSnsTopicPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsTopicPolicyUpsert,
		Read:   resourceAwsSnsTopicPolicyRead,
		Update: resourceAwsSnsTopicPolicyUpsert,
		Delete: resourceAwsSnsTopicPolicyDelete,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
		},
	}
}

func resourceAwsSnsTopicPolicyUpsert(d *schema.ResourceData, meta interface{}) error {
	arn := d.Get("arn").(string)
	req := sns.SetTopicAttributesInput{
		TopicArn:       aws.String(arn),
		AttributeName:  aws.String("Policy"),
		AttributeValue: aws.String(d.Get("policy").(string)),
	}

	d.SetId(arn)

	// Retry the update in the event of an eventually consistent style of
	// error, where say an IAM resource is successfully created but not
	// actually available. See https://github.com/hashicorp/terraform/issues/3660
	log.Printf("[DEBUG] Updating SNS Topic Policy: %s", req)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"retrying"},
		Target:     []string{"success"},
		Refresh:    resourceAwsSNSUpdateRefreshFunc(meta, req),
		Timeout:    3 * time.Minute,
		MinTimeout: 3 * time.Second,
	}
	_, err := stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsSnsTopicPolicyRead(d, meta)
}

func resourceAwsSnsTopicPolicyRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	attributeOutput, err := snsconn.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: aws.String(d.Id()),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			log.Printf("[WARN] SNS Topic (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if attributeOutput.Attributes == nil {
		log.Printf("[WARN] SNS Topic (%q) attributes not found (nil)", d.Id())
		d.SetId("")
		return nil
	}
	attrmap := attributeOutput.Attributes

	policy, ok := attrmap["Policy"]
	if !ok {
		log.Printf("[WARN] SNS Topic (%q) policy not found in attributes", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("policy", policy)

	return nil
}

func resourceAwsSnsTopicPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	accountId, err := getAccountIdFromSnsTopicArn(d.Id(), meta.(*AWSClient).partition)
	if err != nil {
		return err
	}

	req := sns.SetTopicAttributesInput{
		TopicArn:      aws.String(d.Id()),
		AttributeName: aws.String("Policy"),
		// It is impossible to delete a policy or set to empty
		// (confirmed by AWS Support representative)
		// so we instead set it back to the default one
		AttributeValue: aws.String(buildDefaultSnsTopicPolicy(d.Id(), accountId)),
	}

	// Retry the update in the event of an eventually consistent style of
	// error, where say an IAM resource is successfully created but not
	// actually available. See https://github.com/hashicorp/terraform/issues/3660
	log.Printf("[DEBUG] Resetting SNS Topic Policy to default: %s", req)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"retrying"},
		Target:     []string{"success"},
		Refresh:    resourceAwsSNSUpdateRefreshFunc(meta, req),
		Timeout:    3 * time.Minute,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}
	return nil
}

func getAccountIdFromSnsTopicArn(arn, partition string) (string, error) {
	// arn:aws:sns:us-west-2:123456789012:test-new
	// arn:aws-us-gov:sns:us-west-2:123456789012:test-new
	re := regexp.MustCompile(fmt.Sprintf("^arn:%s:sns:[^:]+:([0-9]{12}):.+", partition))
	matches := re.FindStringSubmatch(arn)
	if len(matches) != 2 {
		return "", fmt.Errorf("Unable to get account ID from ARN (%q)", arn)
	}
	return matches[1], nil
}

func buildDefaultSnsTopicPolicy(topicArn, accountId string) string {
	return fmt.Sprintf(`{
  "Version": "2008-10-17",
  "Id": "__default_policy_ID",
  "Statement": [
    {
      "Sid": "__default_statement_ID",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": [
        "SNS:GetTopicAttributes",
        "SNS:SetTopicAttributes",
        "SNS:AddPermission",
        "SNS:RemovePermission",
        "SNS:DeleteTopic",
        "SNS:Subscribe",
        "SNS:ListSubscriptionsByTopic",
        "SNS:Publish",
        "SNS:Receive"
      ],
      "Resource": "%s",
      "Condition": {
        "StringEquals": {
          "AWS:SourceOwner": "%s"
        }
      }
    }
  ]
}`, topicArn, accountId)
}
