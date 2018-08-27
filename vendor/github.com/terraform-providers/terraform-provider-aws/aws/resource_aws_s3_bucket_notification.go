package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func resourceAwsS3BucketNotification() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsS3BucketNotificationPut,
		Read:   resourceAwsS3BucketNotificationRead,
		Update: resourceAwsS3BucketNotificationPut,
		Delete: resourceAwsS3BucketNotificationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"topic": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"filter_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_suffix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"topic_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
						"events": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"queue": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"filter_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_suffix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"queue_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
						"events": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"lambda_function": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"filter_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_suffix": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"lambda_function_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"events": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},
		},
	}
}

func resourceAwsS3BucketNotificationPut(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn
	bucket := d.Get("bucket").(string)

	// TopicNotifications
	topicNotifications := d.Get("topic").([]interface{})
	topicConfigs := make([]*s3.TopicConfiguration, 0, len(topicNotifications))
	for i, c := range topicNotifications {
		tc := &s3.TopicConfiguration{}

		c := c.(map[string]interface{})

		// Id
		if val, ok := c["id"].(string); ok && val != "" {
			tc.Id = aws.String(val)
		} else {
			tc.Id = aws.String(resource.PrefixedUniqueId("tf-s3-topic-"))
		}

		// TopicArn
		if val, ok := c["topic_arn"].(string); ok {
			tc.TopicArn = aws.String(val)
		}

		// Events
		events := d.Get(fmt.Sprintf("topic.%d.events", i)).(*schema.Set).List()
		tc.Events = make([]*string, 0, len(events))
		for _, e := range events {
			tc.Events = append(tc.Events, aws.String(e.(string)))
		}

		// Filter
		filterRules := make([]*s3.FilterRule, 0, 2)
		if val, ok := c["filter_prefix"].(string); ok && val != "" {
			filterRule := &s3.FilterRule{
				Name:  aws.String("prefix"),
				Value: aws.String(val),
			}
			filterRules = append(filterRules, filterRule)
		}
		if val, ok := c["filter_suffix"].(string); ok && val != "" {
			filterRule := &s3.FilterRule{
				Name:  aws.String("suffix"),
				Value: aws.String(val),
			}
			filterRules = append(filterRules, filterRule)
		}
		if len(filterRules) > 0 {
			tc.Filter = &s3.NotificationConfigurationFilter{
				Key: &s3.KeyFilter{
					FilterRules: filterRules,
				},
			}
		}
		topicConfigs = append(topicConfigs, tc)
	}

	// SQS
	queueNotifications := d.Get("queue").([]interface{})
	queueConfigs := make([]*s3.QueueConfiguration, 0, len(queueNotifications))
	for i, c := range queueNotifications {
		qc := &s3.QueueConfiguration{}

		c := c.(map[string]interface{})

		// Id
		if val, ok := c["id"].(string); ok && val != "" {
			qc.Id = aws.String(val)
		} else {
			qc.Id = aws.String(resource.PrefixedUniqueId("tf-s3-queue-"))
		}

		// QueueArn
		if val, ok := c["queue_arn"].(string); ok {
			qc.QueueArn = aws.String(val)
		}

		// Events
		events := d.Get(fmt.Sprintf("queue.%d.events", i)).(*schema.Set).List()
		qc.Events = make([]*string, 0, len(events))
		for _, e := range events {
			qc.Events = append(qc.Events, aws.String(e.(string)))
		}

		// Filter
		filterRules := make([]*s3.FilterRule, 0, 2)
		if val, ok := c["filter_prefix"].(string); ok && val != "" {
			filterRule := &s3.FilterRule{
				Name:  aws.String("prefix"),
				Value: aws.String(val),
			}
			filterRules = append(filterRules, filterRule)
		}
		if val, ok := c["filter_suffix"].(string); ok && val != "" {
			filterRule := &s3.FilterRule{
				Name:  aws.String("suffix"),
				Value: aws.String(val),
			}
			filterRules = append(filterRules, filterRule)
		}
		if len(filterRules) > 0 {
			qc.Filter = &s3.NotificationConfigurationFilter{
				Key: &s3.KeyFilter{
					FilterRules: filterRules,
				},
			}
		}
		queueConfigs = append(queueConfigs, qc)
	}

	// Lambda
	lambdaFunctionNotifications := d.Get("lambda_function").([]interface{})
	lambdaConfigs := make([]*s3.LambdaFunctionConfiguration, 0, len(lambdaFunctionNotifications))
	for i, c := range lambdaFunctionNotifications {
		lc := &s3.LambdaFunctionConfiguration{}

		c := c.(map[string]interface{})

		// Id
		if val, ok := c["id"].(string); ok && val != "" {
			lc.Id = aws.String(val)
		} else {
			lc.Id = aws.String(resource.PrefixedUniqueId("tf-s3-lambda-"))
		}

		// LambdaFunctionArn
		if val, ok := c["lambda_function_arn"].(string); ok {
			lc.LambdaFunctionArn = aws.String(val)
		}

		// Events
		events := d.Get(fmt.Sprintf("lambda_function.%d.events", i)).(*schema.Set).List()
		lc.Events = make([]*string, 0, len(events))
		for _, e := range events {
			lc.Events = append(lc.Events, aws.String(e.(string)))
		}

		// Filter
		filterRules := make([]*s3.FilterRule, 0, 2)
		if val, ok := c["filter_prefix"].(string); ok && val != "" {
			filterRule := &s3.FilterRule{
				Name:  aws.String("prefix"),
				Value: aws.String(val),
			}
			filterRules = append(filterRules, filterRule)
		}
		if val, ok := c["filter_suffix"].(string); ok && val != "" {
			filterRule := &s3.FilterRule{
				Name:  aws.String("suffix"),
				Value: aws.String(val),
			}
			filterRules = append(filterRules, filterRule)
		}
		if len(filterRules) > 0 {
			lc.Filter = &s3.NotificationConfigurationFilter{
				Key: &s3.KeyFilter{
					FilterRules: filterRules,
				},
			}
		}
		lambdaConfigs = append(lambdaConfigs, lc)
	}

	notificationConfiguration := &s3.NotificationConfiguration{}
	if len(lambdaConfigs) > 0 {
		notificationConfiguration.LambdaFunctionConfigurations = lambdaConfigs
	}
	if len(queueConfigs) > 0 {
		notificationConfiguration.QueueConfigurations = queueConfigs
	}
	if len(topicConfigs) > 0 {
		notificationConfiguration.TopicConfigurations = topicConfigs
	}
	i := &s3.PutBucketNotificationConfigurationInput{
		Bucket:                    aws.String(bucket),
		NotificationConfiguration: notificationConfiguration,
	}

	log.Printf("[DEBUG] S3 bucket: %s, Putting notification: %v", bucket, i)
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		if _, err := s3conn.PutBucketNotificationConfiguration(i); err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				switch awserr.Message() {
				case "Unable to validate the following destination configurations":
					return resource.RetryableError(awserr)
				}
			}
			// Didn't recognize the error, so shouldn't retry.
			return resource.NonRetryableError(err)
		}
		// Successful put configuration
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error putting S3 notification configuration: %s", err)
	}

	d.SetId(bucket)

	return resourceAwsS3BucketNotificationRead(d, meta)
}

func resourceAwsS3BucketNotificationDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	i := &s3.PutBucketNotificationConfigurationInput{
		Bucket:                    aws.String(d.Id()),
		NotificationConfiguration: &s3.NotificationConfiguration{},
	}

	log.Printf("[DEBUG] S3 bucket: %s, Deleting notification: %v", d.Id(), i)
	_, err := s3conn.PutBucketNotificationConfiguration(i)
	if err != nil {
		return fmt.Errorf("Error deleting S3 notification configuration: %s", err)
	}

	return nil
}

func resourceAwsS3BucketNotificationRead(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	var err error
	_, err = s3conn.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		if awsError, ok := err.(awserr.RequestFailure); ok && awsError.StatusCode() == 404 {
			log.Printf("[WARN] S3 Bucket (%s) not found, error code (404)", d.Id())
			d.SetId("")
			return nil
		} else {
			// some of the AWS SDK's errors can be empty strings, so let's add
			// some additional context.
			return fmt.Errorf("error reading S3 bucket \"%s\": %s", d.Id(), err)
		}
	}

	// Read the notification configuration
	notificationConfigs, err := s3conn.GetBucketNotificationConfiguration(&s3.GetBucketNotificationConfigurationRequest{
		Bucket: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] S3 Bucket: %s, get notification: %v", d.Id(), notificationConfigs)

	d.Set("bucket", d.Id())

	// Topic Notification
	if err := d.Set("topic", flattenTopicConfigurations(notificationConfigs.TopicConfigurations)); err != nil {
		return fmt.Errorf("error reading S3 bucket \"%s\" topic notification: %s", d.Id(), err)
	}

	// SQS Notification
	if err := d.Set("queue", flattenQueueConfigurations(notificationConfigs.QueueConfigurations)); err != nil {
		return fmt.Errorf("error reading S3 bucket \"%s\" queue notification: %s", d.Id(), err)
	}

	// Lambda Notification
	if err := d.Set("lambda_function", flattenLambdaFunctionConfigurations(notificationConfigs.LambdaFunctionConfigurations)); err != nil {
		return fmt.Errorf("error reading S3 bucket \"%s\" lambda function notification: %s", d.Id(), err)
	}

	return nil
}

func flattenNotificationConfigurationFilter(filter *s3.NotificationConfigurationFilter) map[string]interface{} {
	filterRules := map[string]interface{}{}
	for _, f := range filter.Key.FilterRules {
		if strings.ToLower(*f.Name) == s3.FilterRuleNamePrefix {
			filterRules["filter_prefix"] = *f.Value
		}
		if strings.ToLower(*f.Name) == s3.FilterRuleNameSuffix {
			filterRules["filter_suffix"] = *f.Value
		}
	}
	return filterRules
}

func flattenTopicConfigurations(configs []*s3.TopicConfiguration) []map[string]interface{} {
	topicNotifications := make([]map[string]interface{}, 0, len(configs))
	for _, notification := range configs {
		var conf map[string]interface{}
		if filter := notification.Filter; filter != nil {
			conf = flattenNotificationConfigurationFilter(filter)
		} else {
			conf = map[string]interface{}{}
		}

		conf["id"] = *notification.Id
		conf["events"] = schema.NewSet(schema.HashString, flattenStringList(notification.Events))
		conf["topic_arn"] = *notification.TopicArn
		topicNotifications = append(topicNotifications, conf)
	}

	return topicNotifications
}

func flattenQueueConfigurations(configs []*s3.QueueConfiguration) []map[string]interface{} {
	queueNotifications := make([]map[string]interface{}, 0, len(configs))
	for _, notification := range configs {
		var conf map[string]interface{}
		if filter := notification.Filter; filter != nil {
			conf = flattenNotificationConfigurationFilter(filter)
		} else {
			conf = map[string]interface{}{}
		}

		conf["id"] = *notification.Id
		conf["events"] = schema.NewSet(schema.HashString, flattenStringList(notification.Events))
		conf["queue_arn"] = *notification.QueueArn
		queueNotifications = append(queueNotifications, conf)
	}

	return queueNotifications
}

func flattenLambdaFunctionConfigurations(configs []*s3.LambdaFunctionConfiguration) []map[string]interface{} {
	lambdaFunctionNotifications := make([]map[string]interface{}, 0, len(configs))
	for _, notification := range configs {
		var conf map[string]interface{}
		if filter := notification.Filter; filter != nil {
			conf = flattenNotificationConfigurationFilter(filter)
		} else {
			conf = map[string]interface{}{}
		}

		conf["id"] = *notification.Id
		conf["events"] = schema.NewSet(schema.HashString, flattenStringList(notification.Events))
		conf["lambda_function_arn"] = *notification.LambdaFunctionArn
		lambdaFunctionNotifications = append(lambdaFunctionNotifications, conf)
	}

	return lambdaFunctionNotifications
}
