package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
)

func resourceAwsSnsTopicSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsTopicSubscriptionCreate,
		Read:   resourceAwsSnsTopicSubscriptionRead,
		Update: resourceAwsSnsTopicSubscriptionUpdate,
		Delete: resourceAwsSnsTopicSubscriptionDelete,

		Schema: map[string]*schema.Schema{
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"topic_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"delivery_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"raw_message_delivery": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Default:  false,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSnsTopicSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	if d.Get("protocol") == "email" {
		return fmt.Errorf("Email endpoints are not supported!")
	}

	output, err := subscribeToSNSTopic(d, snsconn)

	if err != nil {
		return err
	}

	log.Printf("New subscription ARN: %s", *output.SubscriptionArn)
	d.SetId(*output.SubscriptionArn)

	// Write the ARN to the 'arn' field for export
	d.Set("arn", *output.SubscriptionArn)

	return resourceAwsSnsTopicSubscriptionUpdate(d, meta)
}

func resourceAwsSnsTopicSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	// If any changes happened, un-subscribe and re-subscribe
	if d.HasChange("protocol") || d.HasChange("endpoint") || d.HasChange("topic_arn") {
		log.Printf("[DEBUG] Updating subscription %s", d.Id())
		// Unsubscribe
		_, err := snsconn.Unsubscribe(&sns.UnsubscribeInput{
			SubscriptionArn: aws.String(d.Id()),
		})

		if err != nil {
			return fmt.Errorf("Error unsubscribing from SNS topic: %s", err)
		}

		// Re-subscribe and set id
		output, err := subscribeToSNSTopic(d, snsconn)
		d.SetId(*output.SubscriptionArn)

	}

	if d.HasChange("raw_message_delivery") {
		_, n := d.GetChange("raw_message_delivery")

		attrValue := "false"

		if n.(bool) {
			attrValue = "true"
		}

		req := &sns.SetSubscriptionAttributesInput{
			SubscriptionArn: aws.String(d.Id()),
			AttributeName:   aws.String("RawMessageDelivery"),
			AttributeValue:  aws.String(attrValue),
		}
		_, err := snsconn.SetSubscriptionAttributes(req)

		if err != nil {
			return fmt.Errorf("Unable to set raw message delivery attribute on subscription")
		}
	}

	return resourceAwsSnsTopicSubscriptionRead(d, meta)
}

func resourceAwsSnsTopicSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	log.Printf("[DEBUG] Loading subscription %s", d.Id())

	attributeOutput, err := snsconn.GetSubscriptionAttributes(&sns.GetSubscriptionAttributesInput{
		SubscriptionArn: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		attrHash := attributeOutput.Attributes
		log.Printf("[DEBUG] raw message delivery: %s", *attrHash["RawMessageDelivery"])
		if *attrHash["RawMessageDelivery"] == "true" {
			d.Set("raw_message_delivery", true)
		} else {
			d.Set("raw_message_delivery", false)
		}
	}

	return nil
}

func resourceAwsSnsTopicSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	log.Printf("[DEBUG] SNS delete topic subscription: %s", d.Id())
	_, err := snsconn.Unsubscribe(&sns.UnsubscribeInput{
		SubscriptionArn: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	return nil
}

func subscribeToSNSTopic(d *schema.ResourceData, snsconn *sns.SNS) (output *sns.SubscribeOutput, err error) {
	protocol := d.Get("protocol").(string)
	endpoint := d.Get("endpoint").(string)
	topic_arn := d.Get("topic_arn").(string)

	log.Printf("[DEBUG] SNS create topic subscription: %s (%s) @ '%s'", endpoint, protocol, topic_arn)

	req := &sns.SubscribeInput{
		Protocol: aws.String(protocol),
		Endpoint: aws.String(endpoint),
		TopicArn: aws.String(topic_arn),
	}

	output, err = snsconn.Subscribe(req)
	if err != nil {
		return nil, fmt.Errorf("Error creating SNS topic: %s", err)
	}

	log.Printf("[DEBUG] Created new subscription!")
	return output, nil
}
