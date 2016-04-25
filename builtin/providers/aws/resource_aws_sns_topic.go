package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
)

// Mutable attributes
var SNSAttributeMap = map[string]string{
	"arn":             "TopicArn",
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
				Computed: true,
				StateFunc: func(v interface{}) string {
					s, ok := v.(string)
					if !ok || s == "" {
						return ""
					}
					jsonb := []byte(s)
					buffer := new(bytes.Buffer)
					if err := json.Compact(buffer, jsonb); err != nil {
						log.Printf("[WARN] Error compacting JSON for Policy in SNS Topic")
						return ""
					}
					value := normalizeJson(buffer.String())
					log.Printf("[DEBUG] topic policy before save: %s", value)
					return value
				},
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
						AttributeValue: aws.String(n.(string)),
					}

					// Retry the update in the event of an eventually consistent style of
					// error, where say an IAM resource is successfully created but not
					// actually available. See https://github.com/hashicorp/terraform/issues/3660
					log.Printf("[DEBUG] Updating SNS Topic (%s) attributes request: %s", d.Id(), req)
					stateConf := &resource.StateChangeConf{
						Pending:    []string{"retrying"},
						Target:     []string{"success"},
						Refresh:    resourceAwsSNSUpdateRefreshFunc(meta, req),
						Timeout:    1 * time.Minute,
						MinTimeout: 3 * time.Second,
					}
					_, err := stateConf.WaitForState()
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return resourceAwsSnsTopicRead(d, meta)
}

func resourceAwsSNSUpdateRefreshFunc(
	meta interface{}, params sns.SetTopicAttributesInput) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		snsconn := meta.(*AWSClient).snsconn
		if _, err := snsconn.SetTopicAttributes(&params); err != nil {
			log.Printf("[WARN] Erroring updating topic attributes: %s", err)
			if awsErr, ok := err.(awserr.Error); ok {
				// if the error contains the PrincipalNotFound message, we can retry
				if strings.Contains(awsErr.Message(), "PrincipalNotFound") {
					log.Printf("[DEBUG] Retrying AWS SNS Topic Update: %s", params)
					return nil, "retrying", nil
				}
			}
			return nil, "failed", err
		}
		return 42, "success", nil
	}
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
						value = normalizeJson(*attrmap[oKey])
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
