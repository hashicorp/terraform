package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
)

// Mutable attributes
var SNSAttributeMap = map[string]string{
	"application_failure_feedback_role_arn":    "ApplicationFailureFeedbackRoleArn",
	"application_success_feedback_role_arn":    "ApplicationSuccessFeedbackRoleArn",
	"application_success_feedback_sample_rate": "ApplicationSuccessFeedbackSampleRate",
	"arn":                                 "TopicArn",
	"delivery_policy":                     "DeliveryPolicy",
	"display_name":                        "DisplayName",
	"http_failure_feedback_role_arn":      "HTTPFailureFeedbackRoleArn",
	"http_success_feedback_role_arn":      "HTTPSuccessFeedbackRoleArn",
	"http_success_feedback_sample_rate":   "HTTPSuccessFeedbackSampleRate",
	"lambda_failure_feedback_role_arn":    "LambdaFailureFeedbackRoleArn",
	"lambda_success_feedback_role_arn":    "LambdaSuccessFeedbackRoleArn",
	"lambda_success_feedback_sample_rate": "LambdaSuccessFeedbackSampleRate",
	"policy":                              "Policy",
	"sqs_failure_feedback_role_arn":       "SQSFailureFeedbackRoleArn",
	"sqs_success_feedback_role_arn":       "SQSSuccessFeedbackRoleArn",
	"sqs_success_feedback_sample_rate":    "SQSSuccessFeedbackSampleRate",
}

func resourceAwsSnsTopic() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsTopicCreate,
		Read:   resourceAwsSnsTopicRead,
		Update: resourceAwsSnsTopicUpdate,
		Delete: resourceAwsSnsTopicDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
			},
			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"policy": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"delivery_policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         false,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"application_success_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"application_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"application_failure_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"http_success_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"http_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"http_failure_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"lambda_success_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"lambda_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"lambda_failure_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"sqs_success_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"sqs_success_feedback_sample_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 100),
			},
			"sqs_failure_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsSnsTopicCreate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.UniqueId()
	}

	log.Printf("[DEBUG] SNS create topic: %s", name)

	req := &sns.CreateTopicInput{
		Name: aws.String(name),
	}

	output, err := snsconn.CreateTopic(req)
	if err != nil {
		return fmt.Errorf("Error creating SNS topic: %s", err)
	}

	d.SetId(*output.TopicArn)

	return resourceAwsSnsTopicUpdate(d, meta)
}

func resourceAwsSnsTopicUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).snsconn

	for terraformAttrName, snsAttrName := range SNSAttributeMap {
		if d.HasChange(terraformAttrName) {
			_, terraformAttrValue := d.GetChange(terraformAttrName)
			err := updateAwsSnsTopicAttribute(d.Id(), snsAttrName, terraformAttrValue, conn)
			if err != nil {
				return err
			}
		}
	}

	return resourceAwsSnsTopicRead(d, meta)
}

func resourceAwsSnsTopicRead(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	log.Printf("[DEBUG] Reading SNS Topic Attributes for %s", d.Id())
	attributeOutput, err := snsconn.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: aws.String(d.Id()),
	})
	if err != nil {
		if isAWSErr(err, sns.ErrCodeNotFoundException, "") {
			log.Printf("[WARN] SNS Topic (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		attrmap := attributeOutput.Attributes
		for terraformAttrName, snsAttrName := range SNSAttributeMap {
			d.Set(terraformAttrName, attrmap[snsAttrName])
		}
	} else {
		for terraformAttrName := range SNSAttributeMap {
			d.Set(terraformAttrName, "")
		}
	}

	// If we have no name set (import) then determine it from the ARN.
	// This is a bit of a heuristic for now since AWS provides no other
	// way to get it.
	if _, ok := d.GetOk("name"); !ok {
		arn := d.Get("arn").(string)
		idx := strings.LastIndex(arn, ":")
		if idx > -1 {
			d.Set("name", arn[idx+1:])
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

func updateAwsSnsTopicAttribute(topicArn, name string, value interface{}, conn *sns.SNS) error {
	// Ignore an empty policy
	if name == "Policy" && value == "" {
		return nil
	}
	log.Printf("[DEBUG] Updating SNS Topic Attribute: %s", name)

	// Make API call to update attributes
	req := sns.SetTopicAttributesInput{
		TopicArn:       aws.String(topicArn),
		AttributeName:  aws.String(name),
		AttributeValue: aws.String(fmt.Sprintf("%v", value)),
	}

	// Retry the update in the event of an eventually consistent style of
	// error, where say an IAM resource is successfully created but not
	// actually available. See https://github.com/hashicorp/terraform/issues/3660
	_, err := retryOnAwsCode(sns.ErrCodeInvalidParameterException, func() (interface{}, error) {
		return conn.SetTopicAttributes(&req)
	})
	if err != nil {
		return err
	}
	return nil
}
