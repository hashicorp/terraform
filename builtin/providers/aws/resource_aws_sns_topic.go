package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
)

// Mutable attributes
var SNSAttributeMap = map[string]string{
	"display_name":    "DisplayName",
	"policy":          "Policy",
	"delivery_policy": "DeliveryPolicy",
}

func resourceAwsSnsTopic() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsTopicCreate,
		Read:   resourceAwsSnsTopicRead,
		Update: resourceAwsSnsTopicUpdate,
		Delete: resourceAwsSnsTopicDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"display_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
			"delivery_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
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

	name := d.Get("name").(string)

	log.Printf("[DEBUG] SNS create topic: %s", name)

	req := &sns.CreateTopicInput{
		Name: aws.String(name),
	}

	output, err := snsconn.CreateTopic(req)
	if err != nil {
		return fmt.Errorf("Error creating SNS topic: %s", err)
	}

	d.SetId(*output.TopicArn)

	// Write the ARN to the 'arn' field for export
	d.Set("arn", *output.TopicArn)

	return resourceAwsSnsTopicUpdate(d, meta)
}

func resourceAwsSnsTopicUpdate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	resource := *resourceAwsSnsTopic()

	for k, _ := range resource.Schema {
		if attrKey, ok := SNSAttributeMap[k]; ok {
			if d.HasChange(k) {
				log.Printf("[DEBUG] Updating %s", attrKey)
				_, n := d.GetChange(k)
				// Ignore an empty policy
				if !(k == "policy" && n == "") {
					// Make API call to update attributes
					req := &sns.SetTopicAttributesInput{
						TopicArn:       aws.String(d.Id()),
						AttributeName:  aws.String(attrKey),
						AttributeValue: aws.String(n.(string)),
					}
					snsconn.SetTopicAttributes(req)
				}
			}
		}
	}

	return resourceAwsSnsTopicRead(d, meta)
}

func resourceAwsSnsTopicRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	attributeOutput, err := snsconn.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: aws.String(d.Id()),
	})

	if err != nil {
		return err
	}

	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		attrmap := attributeOutput.Attributes
		resource := *resourceAwsSnsTopic()
		// iKey = internal struct key, oKey = AWS Attribute Map key
		for iKey, oKey := range SNSAttributeMap {
			log.Printf("[DEBUG] Updating %s => %s", iKey, oKey)

			if attrmap[oKey] != nil {
				// Some of the fetched attributes are stateful properties such as
				// the number of subscriptions, the owner, etc. skip those
				if resource.Schema[iKey] != nil {
					value := *attrmap[oKey]
					log.Printf("[DEBUG] Updating %s => %s -> %s", iKey, oKey, value)
					d.Set(iKey, *attrmap[oKey])
				}
			}
		}
	}

	return nil
}

func resourceAwsSnsTopicDelete(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	log.Printf("[DEBUG] SNS Delete Topic: %s", d.Id())
	_, err := snsconn.DeleteTopic(&sns.DeleteTopicInput{
		TopicArn: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	return nil
}
