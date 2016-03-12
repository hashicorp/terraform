package aws

import (
	"fmt"
	"log"
	"strings"

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

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"topic": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_suffix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"topic": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"events": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"queue": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_suffix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"queue": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"events": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
			},

			"lambda_function": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"filter_suffix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"lambda_function": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"events": &schema.Schema{
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
	topicNotifications := d.Get("topic").(*schema.Set).List()
	topicConfigs := make([]*s3.TopicConfiguration, 0, len(topicNotifications))
	for _, c := range topicNotifications {
		tc := &s3.TopicConfiguration{}

		c := c.(map[string]interface{})

		// Id
		if val, ok := c["id"].(string); ok {
			tc.Id = aws.String(val)
		}

		// TopicArn
		if val, ok := c["topic"].(string); ok {
			tc.TopicArn = aws.String(val)
		}

		// Events
		if v := c["events"].(*schema.Set); v.Len() > 0 {
			tc.Events = make([]*string, 0, v.Len())
			for _, val := range v.List() {
				tc.Events = append(tc.Events, aws.String(val.(string)))
			}
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
		tc.Filter = &s3.NotificationConfigurationFilter{
			Key: &s3.KeyFilter{
				FilterRules: filterRules,
			},
		}
		topicConfigs = append(topicConfigs, tc)
	}

	// Lambda
	lambdaFunctionNotifications := d.Get("lambda_function").(*schema.Set).List()
	lambdaConfigs := make([]*s3.LambdaFunctionConfiguration, 0, len(lambdaFunctionNotifications))
	for _, c := range lambdaFunctionNotifications {
		lc := &s3.LambdaFunctionConfiguration{}

		c := c.(map[string]interface{})

		// Id
		if val, ok := c["id"].(string); ok {
			lc.Id = aws.String(val)
		}

		// LambdaFunctionArn
		if val, ok := c["lambda_function"].(string); ok {
			lc.LambdaFunctionArn = aws.String(val)
		}

		// Events
		if v := c["events"].(*schema.Set); v.Len() > 0 {
			lc.Events = make([]*string, 0, v.Len())
			for _, val := range v.List() {
				lc.Events = append(lc.Events, aws.String(val.(string)))
			}
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
		lc.Filter = &s3.NotificationConfigurationFilter{
			Key: &s3.KeyFilter{
				FilterRules: filterRules,
			},
		}
		lambdaConfigs = append(lambdaConfigs, lc)
	}

	// SQS
	queueNotifications := d.Get("queue").(*schema.Set).List()
	queueConfigs := make([]*s3.QueueConfiguration, 0, len(queueNotifications))
	for _, c := range queueNotifications {
		qc := &s3.QueueConfiguration{}

		c := c.(map[string]interface{})

		// Id
		if val, ok := c["id"].(string); ok {
			qc.Id = aws.String(val)
		}

		// QueueArn
		if val, ok := c["queue"].(string); ok {
			qc.QueueArn = aws.String(val)
		}

		// Events
		if v := c["events"].(*schema.Set); v.Len() > 0 {
			qc.Events = make([]*string, 0, v.Len())
			for _, val := range v.List() {
				qc.Events = append(qc.Events, aws.String(val.(string)))
			}
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
		qc.Filter = &s3.NotificationConfigurationFilter{
			Key: &s3.KeyFilter{
				FilterRules: filterRules,
			},
		}
		queueConfigs = append(queueConfigs, qc)
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
		Bucket: aws.String(bucket),
		NotificationConfiguration: notificationConfiguration,
	}

	log.Printf("[DEBUG] S3 bucket: %s, Putting notification: %v", bucket, i)
	_, err := s3conn.PutBucketNotificationConfiguration(i)
	if err != nil {
		return fmt.Errorf("Error putting S3 notification configuration: %s", err)
	}

	d.SetId(bucket)

	return resourceAwsS3BucketNotificationRead(d, meta)
}

func resourceAwsS3BucketNotificationDelete(d *schema.ResourceData, meta interface{}) error {
	s3conn := meta.(*AWSClient).s3conn

	i := &s3.PutBucketNotificationConfigurationInput{
		Bucket: aws.String(d.Id()),
		NotificationConfiguration: &s3.NotificationConfiguration{},
	}

	log.Printf("[DEBUG] S3 bucket: %s, Deleting notification: %v", d.Id(), i)
	_, err := s3conn.PutBucketNotificationConfiguration(i)
	if err != nil {
		return fmt.Errorf("Error deleting S3 notification configuration: %s", err)
	}

	d.SetId("")

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
	// Topic Notification
	topicNotifications := make([]map[string]interface{}, 0, len(notificationConfigs.TopicConfigurations))
	for _, notification := range notificationConfigs.TopicConfigurations {
		conf := map[string]interface{}{}

		if notification.Id != nil {
			conf["id"] = *notification.Id
		}

		for _, f := range notification.Filter.Key.FilterRules {
			if strings.ToLower(*f.Name) == "prefix" {
				conf["filter_prefix"] = *f.Value
			}
			if strings.ToLower(*f.Name) == "suffix" {
				conf["filter_suffix"] = *f.Value
			}
		}

		conf["events"] = schema.NewSet(schema.HashString, flattenStringList(notification.Events))
		conf["topic"] = *notification.TopicArn
		topicNotifications = append(topicNotifications, conf)
	}
	if err := d.Set("topic", topicNotifications); err != nil {
		return fmt.Errorf("error reading S3 bucket \"%s\" topic notification: %s", d.Id(), err)
	}

	// Lambda Notification
	lambdaFunctionNotifications := make([]map[string]interface{}, 0, len(notificationConfigs.LambdaFunctionConfigurations))
	for _, notification := range notificationConfigs.LambdaFunctionConfigurations {
		conf := map[string]interface{}{}

		if notification.Id != nil {
			conf["id"] = *notification.Id
		}

		for _, f := range notification.Filter.Key.FilterRules {
			if strings.ToLower(*f.Name) == "prefix" {
				conf["filter_prefix"] = *f.Value
			}
			if strings.ToLower(*f.Name) == "suffix" {
				conf["filter_suffix"] = *f.Value
			}
		}

		conf["events"] = schema.NewSet(schema.HashString, flattenStringList(notification.Events))
		conf["lambda_function"] = *notification.LambdaFunctionArn
		lambdaFunctionNotifications = append(lambdaFunctionNotifications, conf)
	}
	if err := d.Set("lambda_function", lambdaFunctionNotifications); err != nil {
		return fmt.Errorf("error reading S3 bucket \"%s\" lambda function notification: %s", d.Id(), err)
	}

	// SQS Notification
	queueNotifications := make([]map[string]interface{}, 0, len(notificationConfigs.QueueConfigurations))
	for _, notification := range notificationConfigs.QueueConfigurations {
		conf := map[string]interface{}{}

		if notification.Id != nil {
			conf["id"] = *notification.Id
		}

		for _, f := range notification.Filter.Key.FilterRules {
			if strings.ToLower(*f.Name) == "prefix" {
				conf["filter_prefix"] = *f.Value
			}
			if strings.ToLower(*f.Name) == "suffix" {
				conf["filter_suffix"] = *f.Value
			}
		}

		conf["events"] = schema.NewSet(schema.HashString, flattenStringList(notification.Events))
		conf["queue"] = *notification.QueueArn
		queueNotifications = append(queueNotifications, conf)
	}
	if err := d.Set("queue", queueNotifications); err != nil {
		return fmt.Errorf("error reading S3 bucket \"%s\" queue notification: %s", d.Id(), err)
	}

	return nil
}
