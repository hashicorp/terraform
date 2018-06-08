package aws

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
)

const awsSNSPendingConfirmationMessage = "pending confirmation"
const awsSNSPendingConfirmationMessageWithoutSpaces = "pendingconfirmation"
const awsSNSPasswordObfuscationPattern = "****"

var SNSSubscriptionAttributeMap = map[string]string{
	"topic_arn":            "TopicArn",
	"endpoint":             "Endpoint",
	"protocol":             "Protocol",
	"raw_message_delivery": "RawMessageDelivery",
	"filter_policy":        "FilterPolicy",
}

func resourceAwsSnsTopicSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsTopicSubscriptionCreate,
		Read:   resourceAwsSnsTopicSubscriptionRead,
		Update: resourceAwsSnsTopicSubscriptionUpdate,
		Delete: resourceAwsSnsTopicSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"protocol": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateSNSSubscriptionProtocol,
			},
			"endpoint": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"endpoint_auto_confirms": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"confirmation_timeout_in_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},
			"topic_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"delivery_policy": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"raw_message_delivery": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"filter_policy": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
		},
	}
}

func resourceAwsSnsTopicSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	output, err := subscribeToSNSTopic(d, snsconn)

	if err != nil {
		return err
	}

	if subscriptionHasPendingConfirmation(output.SubscriptionArn) {
		log.Printf("[WARN] Invalid SNS Subscription, received a \"%s\" ARN", awsSNSPendingConfirmationMessage)
		return nil
	}

	log.Printf("New subscription ARN: %s", *output.SubscriptionArn)
	d.SetId(*output.SubscriptionArn)

	// Write the ARN to the 'arn' field for export
	d.Set("arn", output.SubscriptionArn)

	return resourceAwsSnsTopicSubscriptionUpdate(d, meta)
}

func resourceAwsSnsTopicSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

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

	if d.HasChange("filter_policy") {
		_, n := d.GetChange("filter_policy")

		attrValue := n.(string)

		req := &sns.SetSubscriptionAttributesInput{
			SubscriptionArn: aws.String(d.Id()),
			AttributeName:   aws.String("FilterPolicy"),
			AttributeValue:  aws.String(attrValue),
		}
		_, err := snsconn.SetSubscriptionAttributes(req)

		if err != nil {
			return fmt.Errorf("Unable to set filter policy attribute on subscription: %s", err)
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
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			log.Printf("[WARN] SNS Topic Subscription (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		attrHash := attributeOutput.Attributes
		resource := *resourceAwsSnsTopicSubscription()

		for iKey, oKey := range SNSSubscriptionAttributeMap {
			log.Printf("[DEBUG] Reading %s => %s", iKey, oKey)

			if attrHash[oKey] != nil {
				if resource.Schema[iKey] != nil {
					var value string
					value = *attrHash[oKey]
					log.Printf("[DEBUG] Reading %s => %s -> %s", iKey, oKey, value)
					d.Set(iKey, value)
				}
			}
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
	endpoint_auto_confirms := d.Get("endpoint_auto_confirms").(bool)
	confirmation_timeout_in_minutes := d.Get("confirmation_timeout_in_minutes").(int)

	if strings.Contains(protocol, "http") && !endpoint_auto_confirms {
		return nil, fmt.Errorf("Protocol http/https is only supported for endpoints which auto confirms!")
	}

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

	log.Printf("[DEBUG] Finished subscribing to topic %s with subscription arn %s", topic_arn, *output.SubscriptionArn)

	if strings.Contains(protocol, "http") && subscriptionHasPendingConfirmation(output.SubscriptionArn) {

		log.Printf("[DEBUG] SNS create topic subscription is pending so fetching the subscription list for topic : %s (%s) @ '%s'", endpoint, protocol, topic_arn)

		err = resource.Retry(time.Duration(confirmation_timeout_in_minutes)*time.Minute, func() *resource.RetryError {

			subscription, err := findSubscriptionByNonID(d, snsconn)

			if subscription != nil {
				output.SubscriptionArn = subscription.SubscriptionArn
				return nil
			}

			if err != nil {
				return resource.RetryableError(
					fmt.Errorf("Error fetching subscriptions for SNS topic %s: %s", topic_arn, err))
			}

			return resource.RetryableError(
				fmt.Errorf("Endpoint (%s) did not autoconfirm the subscription for topic %s", endpoint, topic_arn))
		})

		if err != nil {
			return nil, err
		}
	}

	log.Printf("[DEBUG] Created new subscription! %s", *output.SubscriptionArn)
	return output, nil
}

// finds a subscription using protocol, endpoint and topic_arn (which is a key in sns subscription)
func findSubscriptionByNonID(d *schema.ResourceData, snsconn *sns.SNS) (*sns.Subscription, error) {
	protocol := d.Get("protocol").(string)
	endpoint := d.Get("endpoint").(string)
	topic_arn := d.Get("topic_arn").(string)
	obfuscatedEndpoint := obfuscateEndpoint(endpoint)

	req := &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(topic_arn),
	}

	for {
		res, err := snsconn.ListSubscriptionsByTopic(req)

		if err != nil {
			return nil, fmt.Errorf("Error fetching subscriptions for topic %s : %s", topic_arn, err)
		}

		for _, subscription := range res.Subscriptions {
			log.Printf("[DEBUG] check subscription with Subscription EndPoint %s (local: %s), Protocol %s, topicARN %s and SubscriptionARN %s", *subscription.Endpoint, obfuscatedEndpoint, *subscription.Protocol, *subscription.TopicArn, *subscription.SubscriptionArn)
			if *subscription.Endpoint == obfuscatedEndpoint && *subscription.Protocol == protocol && *subscription.TopicArn == topic_arn && !subscriptionHasPendingConfirmation(subscription.SubscriptionArn) {
				return subscription, nil
			}
		}

		// if there are more than 100 subscriptions then go to the next 100 otherwise return an error
		if res.NextToken != nil {
			req.NextToken = res.NextToken
		} else {
			return nil, fmt.Errorf("Error finding subscription for topic %s with endpoint %s and protocol %s", topic_arn, endpoint, protocol)
		}
	}
}

// returns true if arn is nil or has both pending and confirmation words in the arn
func subscriptionHasPendingConfirmation(arn *string) bool {
	if arn != nil && !strings.Contains(strings.Replace(strings.ToLower(*arn), " ", "", -1), awsSNSPendingConfirmationMessageWithoutSpaces) {
		return false
	}

	return true
}

// returns the endpoint with obfuscated password, if any
func obfuscateEndpoint(endpoint string) string {
	res, err := url.Parse(endpoint)
	if err != nil {
		fmt.Println(err)
	}

	var obfuscatedEndpoint = res.String()

	// If the user is defined, we try to get the username and password, if defined.
	// Then, we update the user with the obfuscated version.
	if res.User != nil {
		if password, ok := res.User.Password(); ok {
			obfuscatedEndpoint = strings.Replace(obfuscatedEndpoint, password, awsSNSPasswordObfuscationPattern, 1)
		}
	}
	return obfuscatedEndpoint
}
