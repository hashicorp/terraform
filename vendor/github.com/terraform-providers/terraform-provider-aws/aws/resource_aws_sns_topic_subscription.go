package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/sns"
)

const awsSNSPendingConfirmationMessage = "pending confirmation"
const awsSNSPendingConfirmationMessageWithoutSpaces = "pendingconfirmation"
const awsSNSPasswordObfuscationPattern = "****"

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
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					// email and email-json not supported
					"application",
					"http",
					"https",
					"lambda",
					"sms",
					"sqs",
				}, true),
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
				Type:             schema.TypeString,
				Optional:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: suppressEquivalentSnsTopicSubscriptionDeliveryPolicy,
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
				ValidateFunc:     validation.ValidateJsonString,
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
		if err := snsSubscriptionAttributeUpdate(snsconn, d.Id(), "RawMessageDelivery", fmt.Sprintf("%t", d.Get("raw_message_delivery").(bool))); err != nil {
			return err
		}
	}

	if d.HasChange("filter_policy") {
		filterPolicy := d.Get("filter_policy").(string)

		// https://docs.aws.amazon.com/sns/latest/dg/message-filtering.html#message-filtering-policy-remove
		if filterPolicy == "" {
			filterPolicy = "{}"
		}

		if err := snsSubscriptionAttributeUpdate(snsconn, d.Id(), "FilterPolicy", filterPolicy); err != nil {
			return err
		}
	}

	if d.HasChange("delivery_policy") {
		if err := snsSubscriptionAttributeUpdate(snsconn, d.Id(), "DeliveryPolicy", d.Get("delivery_policy").(string)); err != nil {
			return err
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

	if isAWSErr(err, sns.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] SNS Topic Subscription (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading SNS Topic Subscription (%s) attributes: %s", d.Id(), err)
	}

	if attributeOutput == nil || len(attributeOutput.Attributes) == 0 {
		return fmt.Errorf("error reading SNS Topic Subscription (%s) attributes: no attributes found", d.Id())
	}

	d.Set("arn", attributeOutput.Attributes["SubscriptionArn"])
	d.Set("delivery_policy", attributeOutput.Attributes["DeliveryPolicy"])
	d.Set("endpoint", attributeOutput.Attributes["Endpoint"])
	d.Set("filter_policy", attributeOutput.Attributes["FilterPolicy"])
	d.Set("protocol", attributeOutput.Attributes["Protocol"])

	d.Set("raw_message_delivery", false)
	if v, ok := attributeOutput.Attributes["RawMessageDelivery"]; ok && aws.StringValue(v) == "true" {
		d.Set("raw_message_delivery", true)
	}

	d.Set("topic_arn", attributeOutput.Attributes["TopicArn"])

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

func snsSubscriptionAttributeUpdate(snsconn *sns.SNS, subscriptionArn, attributeName, attributeValue string) error {
	req := &sns.SetSubscriptionAttributesInput{
		SubscriptionArn: aws.String(subscriptionArn),
		AttributeName:   aws.String(attributeName),
		AttributeValue:  aws.String(attributeValue),
	}
	_, err := snsconn.SetSubscriptionAttributes(req)

	if err != nil {
		return fmt.Errorf("error setting subscription (%s) attribute (%s): %s", subscriptionArn, attributeName, err)
	}
	return nil
}

type snsTopicSubscriptionDeliveryPolicy struct {
	Guaranteed         bool                                                  `json:"guaranteed,omitempty"`
	HealthyRetryPolicy *snsTopicSubscriptionDeliveryPolicyHealthyRetryPolicy `json:"healthyRetryPolicy,omitempty"`
	SicklyRetryPolicy  *snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy  `json:"sicklyRetryPolicy,omitempty"`
	ThrottlePolicy     *snsTopicSubscriptionDeliveryPolicyThrottlePolicy     `json:"throttlePolicy,omitempty"`
}

func (s snsTopicSubscriptionDeliveryPolicy) String() string {
	return awsutil.Prettify(s)
}

func (s snsTopicSubscriptionDeliveryPolicy) GoString() string {
	return s.String()
}

type snsTopicSubscriptionDeliveryPolicyHealthyRetryPolicy struct {
	BackoffFunction    string `json:"backoffFunction,omitempty"`
	MaxDelayTarget     int    `json:"maxDelayTarget,omitempty"`
	MinDelayTarget     int    `json:"minDelayTarget,omitempty"`
	NumMaxDelayRetries int    `json:"numMaxDelayRetries,omitempty"`
	NumMinDelayRetries int    `json:"numMinDelayRetries,omitempty"`
	NumNoDelayRetries  int    `json:"numNoDelayRetries,omitempty"`
	NumRetries         int    `json:"numRetries,omitempty"`
}

func (s snsTopicSubscriptionDeliveryPolicyHealthyRetryPolicy) String() string {
	return awsutil.Prettify(s)
}

func (s snsTopicSubscriptionDeliveryPolicyHealthyRetryPolicy) GoString() string {
	return s.String()
}

type snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy struct {
	BackoffFunction    string `json:"backoffFunction,omitempty"`
	MaxDelayTarget     int    `json:"maxDelayTarget,omitempty"`
	MinDelayTarget     int    `json:"minDelayTarget,omitempty"`
	NumMaxDelayRetries int    `json:"numMaxDelayRetries,omitempty"`
	NumMinDelayRetries int    `json:"numMinDelayRetries,omitempty"`
	NumNoDelayRetries  int    `json:"numNoDelayRetries,omitempty"`
	NumRetries         int    `json:"numRetries,omitempty"`
}

func (s snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy) String() string {
	return awsutil.Prettify(s)
}

func (s snsTopicSubscriptionDeliveryPolicySicklyRetryPolicy) GoString() string {
	return s.String()
}

type snsTopicSubscriptionDeliveryPolicyThrottlePolicy struct {
	MaxReceivesPerSecond int `json:"maxReceivesPerSecond,omitempty"`
}

func (s snsTopicSubscriptionDeliveryPolicyThrottlePolicy) String() string {
	return awsutil.Prettify(s)
}

func (s snsTopicSubscriptionDeliveryPolicyThrottlePolicy) GoString() string {
	return s.String()
}

func suppressEquivalentSnsTopicSubscriptionDeliveryPolicy(k, old, new string, d *schema.ResourceData) bool {
	var deliveryPolicy snsTopicSubscriptionDeliveryPolicy

	if err := json.Unmarshal([]byte(old), &deliveryPolicy); err != nil {
		log.Printf("[WARN] Unable to unmarshal SNS Topic Subscription delivery policy JSON: %s", err)
		return false
	}

	normalizedDeliveryPolicy, err := json.Marshal(deliveryPolicy)

	if err != nil {
		log.Printf("[WARN] Unable to marshal SNS Topic Subscription delivery policy back to JSON: %s", err)
		return false
	}

	ob := bytes.NewBufferString("")
	if err := json.Compact(ob, normalizedDeliveryPolicy); err != nil {
		return false
	}

	nb := bytes.NewBufferString("")
	if err := json.Compact(nb, []byte(new)); err != nil {
		return false
	}

	return jsonBytesEqual(ob.Bytes(), nb.Bytes())
}
