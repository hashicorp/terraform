package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/sns"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSnsTopic() *schema.Resource {
	return &schema.Resource{
		// Topic updates are idempotent.
		Create: resourceAwsSnsTopicCreate,
		Update: resourceAwsSnsTopicCreate,

		Read:   resourceAwsSnsTopicRead,
		Delete: resourceAwsSnsTopicDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSnsTopicCreate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	createOpts := &sns.CreateTopicInput{
		Name: aws.String(d.Get("name").(string)),
	}

	log.Printf("[DEBUG] Creating SNS topic")
	resp, err := snsconn.CreateTopic(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating SNS topic: %s", err)
	}

	// Store the ID, in this case the ARN.
	topicArn := resp.TopicARN
	d.SetId(*topicArn)
	log.Printf("[INFO] SNS topic ID: %s", *topicArn)

	return resourceAwsSnsTopicRead(d, meta)
}

func resourceAwsSnsTopicRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	match, err := seekSnsTopic(d.Id(), snsconn)
	if err != nil {
		return err
	}

	if match == "" {
		d.SetId("")
	} else {
		d.Set("arn", match)
		d.Set("name", parseSnsTopicArn(match))
	}

	return nil
}

func resourceAwsSnsTopicDelete(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	_, err := snsconn.DeleteTopic(&sns.DeleteTopicInput{
		TopicARN: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error deleting SNS topic: %#v", err)
	}
	return nil
}

// parseSnsTopicArn extracts the topic's name from its amazon resource number.
func parseSnsTopicArn(arn string) string {
	parts := strings.Split(arn, ":")
	return parts[len(parts)-1]
}
