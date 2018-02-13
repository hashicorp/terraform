package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
)

// Mutable attributes
var SNSAttributeMap = map[string]string{
	"arn":                                      "TopicArn",
	"display_name":                             "DisplayName",
	"policy":                                   "Policy",
	"delivery_policy":                          "DeliveryPolicy",
	"application_success_feedback_role_arn":    "ApplicationSuccessFeedbackRoleArn",
	"application_success_feedback_sample_rate": "ApplicationSuccessFeedbackSampleRate",
	"application_failure_feedback_role_arn":    "ApplicationFailureFeedbackRoleArn",
	"http_success_feedback_role_arn":           "HTTPSuccessFeedbackRoleArn",
	"http_success_feedback_sample_rate":        "HTTPSuccessFeedbackSampleRate",
	"http_failure_feedback_role_arn":           "HTTPFailureFeedbackRoleArn",
	"lambda_success_feedback_role_arn":         "LambdaSuccessFeedbackRoleArn",
	"lambda_success_feedback_sample_rate":      "LambdaSuccessFeedbackSampleRate",
	"lambda_failure_feedback_role_arn":         "LambdaFailureFeedbackRoleArn",
	"sqs_success_feedback_role_arn":            "SQSSuccessFeedbackRoleArn",
	"sqs_success_feedback_sample_rate":         "SQSSuccessFeedbackSampleRate",
	"sqs_failure_feedback_role_arn":            "SQSFailureFeedbackRoleArn"}

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
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
			},
			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"display_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"policy": &schema.Schema{
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
			"delivery_policy": &schema.Schema{
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
				ValidateFunc: validateIntegerInRange(0, 100),
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
				ValidateFunc: validateIntegerInRange(0, 100),
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
				ValidateFunc: validateIntegerInRange(0, 100),
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
				ValidateFunc: validateIntegerInRange(0, 100),
			},
			"sqs_failure_feedback_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
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

	// Write the ARN to the 'arn' field for export
	d.Set("arn", *output.TopicArn)

	return resourceAwsSnsTopicUpdate(d, meta)
}

func resourceAwsSnsTopicUpdate(d *schema.ResourceData, meta interface{}) error {
	r := *resourceAwsSnsTopic()

	for k, _ := range r.Schema {
		if attrKey, ok := SNSAttributeMap[k]; ok {
			if d.HasChange(k) {
				log.Printf("[DEBUG] Updating %s", attrKey)
				_, n := d.GetChange(k)
				// Ignore an empty policy
				if !(k == "policy" && n == "") {
					// Make API call to update attributes
					req := sns.SetTopicAttributesInput{
						TopicArn:       aws.String(d.Id()),
						AttributeName:  aws.String(attrKey),
						AttributeValue: aws.String(fmt.Sprintf("%v", n)),
					}
					conn := meta.(*AWSClient).snsconn
					// Retry the update in the event of an eventually consistent style of
					// error, where say an IAM resource is successfully created but not
					// actually available. See https://github.com/hashicorp/terraform/issues/3660
					_, err := retryOnAwsCode("InvalidParameter", func() (interface{}, error) {
						return conn.SetTopicAttributes(&req)
					})
					return err
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
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			log.Printf("[WARN] SNS Topic (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		attrmap := attributeOutput.Attributes
		resource := *resourceAwsSnsTopic()
		// iKey = internal struct key, oKey = AWS Attribute Map key
		for iKey, oKey := range SNSAttributeMap {
			log.Printf("[DEBUG] Reading %s => %s", iKey, oKey)

			if attrmap[oKey] != nil {
				// Some of the fetched attributes are stateful properties such as
				// the number of subscriptions, the owner, etc. skip those
				if resource.Schema[iKey] != nil {
					var value string
					if iKey == "policy" {
						value, err = structure.NormalizeJsonString(*attrmap[oKey])
						if err != nil {
							return errwrap.Wrapf("policy contains an invalid JSON: {{err}}", err)
						}
					} else {
						value = *attrmap[oKey]
					}
					log.Printf("[DEBUG] Reading %s => %s -> %s", iKey, oKey, value)
					d.Set(iKey, value)
				}
			}
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
