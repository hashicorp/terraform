package aws

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
)

var sqsQueueAttributeMap = map[string]string{
	"delay_seconds":                     sqs.QueueAttributeNameDelaySeconds,
	"max_message_size":                  sqs.QueueAttributeNameMaximumMessageSize,
	"message_retention_seconds":         sqs.QueueAttributeNameMessageRetentionPeriod,
	"receive_wait_time_seconds":         sqs.QueueAttributeNameReceiveMessageWaitTimeSeconds,
	"visibility_timeout_seconds":        sqs.QueueAttributeNameVisibilityTimeout,
	"policy":                            sqs.QueueAttributeNamePolicy,
	"redrive_policy":                    sqs.QueueAttributeNameRedrivePolicy,
	"arn":                               sqs.QueueAttributeNameQueueArn,
	"fifo_queue":                        sqs.QueueAttributeNameFifoQueue,
	"content_based_deduplication":       sqs.QueueAttributeNameContentBasedDeduplication,
	"kms_master_key_id":                 sqs.QueueAttributeNameKmsMasterKeyId,
	"kms_data_key_reuse_period_seconds": sqs.QueueAttributeNameKmsDataKeyReusePeriodSeconds,
}

// A number of these are marked as computed because if you don't
// provide a value, SQS will provide you with defaults (which are the
// default values specified below)
func resourceAwsSqsQueue() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSqsQueueCreate,
		Read:   resourceAwsSqsQueueRead,
		Update: resourceAwsSqsQueueUpdate,
		Delete: resourceAwsSqsQueueDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateSQSQueueName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
			},
			"delay_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"max_message_size": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  262144,
			},
			"message_retention_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  345600,
			},
			"receive_wait_time_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"visibility_timeout_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
			},
			"policy": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateFunc:     validation.ValidateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
			"redrive_policy": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.ValidateJsonString,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"fifo_queue": {
				Type:     schema.TypeBool,
				Default:  false,
				ForceNew: true,
				Optional: true,
			},
			"content_based_deduplication": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
			"kms_master_key_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"kms_data_key_reuse_period_seconds": {
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsSqsQueueCreate(d *schema.ResourceData, meta interface{}) error {
	sqsconn := meta.(*AWSClient).sqsconn

	var name string

	fq := d.Get("fifo_queue").(bool)

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
		if fq {
			name += ".fifo"
		}
	} else {
		name = resource.UniqueId()
	}

	cbd := d.Get("content_based_deduplication").(bool)

	if fq {
		if errors := validateSQSFifoQueueName(name); len(errors) > 0 {
			return fmt.Errorf("Error validating the FIFO queue name: %v", errors)
		}
	} else {
		if errors := validateSQSNonFifoQueueName(name); len(errors) > 0 {
			return fmt.Errorf("Error validating SQS queue name: %v", errors)
		}
	}

	if !fq && cbd {
		return fmt.Errorf("Content based deduplication can only be set with FIFO queues")
	}

	log.Printf("[DEBUG] SQS queue create: %s", name)

	req := &sqs.CreateQueueInput{
		QueueName: aws.String(name),
	}

	attributes := make(map[string]*string)

	queueResource := *resourceAwsSqsQueue()

	for k, s := range queueResource.Schema {
		if attrKey, ok := sqsQueueAttributeMap[k]; ok {
			if value, ok := d.GetOk(k); ok {
				switch s.Type {
				case schema.TypeInt:
					attributes[attrKey] = aws.String(strconv.Itoa(value.(int)))
				case schema.TypeBool:
					attributes[attrKey] = aws.String(strconv.FormatBool(value.(bool)))
				default:
					attributes[attrKey] = aws.String(value.(string))
				}
			}

		}
	}

	if len(attributes) > 0 {
		req.Attributes = attributes
	}

	var output *sqs.CreateQueueOutput
	err := resource.Retry(70*time.Second, func() *resource.RetryError {
		var err error
		output, err = sqsconn.CreateQueue(req)
		if err != nil {
			if isAWSErr(err, sqs.ErrCodeQueueDeletedRecently, "You must wait 60 seconds after deleting a queue before you can create another with the same name.") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error creating SQS queue: %s", err)
	}

	d.SetId(aws.StringValue(output.QueueUrl))

	return resourceAwsSqsQueueUpdate(d, meta)
}

func resourceAwsSqsQueueUpdate(d *schema.ResourceData, meta interface{}) error {
	sqsconn := meta.(*AWSClient).sqsconn

	if err := setTagsSQS(sqsconn, d); err != nil {
		return err
	}

	attributes := make(map[string]*string)

	resource := *resourceAwsSqsQueue()

	for k, s := range resource.Schema {
		if attrKey, ok := sqsQueueAttributeMap[k]; ok {
			if d.HasChange(k) {
				log.Printf("[DEBUG] Updating %s", attrKey)
				_, n := d.GetChange(k)
				switch s.Type {
				case schema.TypeInt:
					attributes[attrKey] = aws.String(strconv.Itoa(n.(int)))
				case schema.TypeBool:
					attributes[attrKey] = aws.String(strconv.FormatBool(n.(bool)))
				default:
					attributes[attrKey] = aws.String(n.(string))
				}
			}
		}
	}

	if len(attributes) > 0 {
		req := &sqs.SetQueueAttributesInput{
			QueueUrl:   aws.String(d.Id()),
			Attributes: attributes,
		}
		if _, err := sqsconn.SetQueueAttributes(req); err != nil {
			return fmt.Errorf("Error updating SQS attributes: %s", err)
		}
	}

	return resourceAwsSqsQueueRead(d, meta)
}

func resourceAwsSqsQueueRead(d *schema.ResourceData, meta interface{}) error {
	sqsconn := meta.(*AWSClient).sqsconn

	attributeOutput, err := sqsconn.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(d.Id()),
		AttributeNames: []*string{aws.String("All")},
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			log.Printf("ERROR Found %s", awsErr.Code())
			if awsErr.Code() == "AWS.SimpleQueueService.NonExistentQueue" {
				d.SetId("")
				log.Printf("[DEBUG] SQS Queue (%s) not found", d.Get("name").(string))
				return nil
			}
		}
		return err
	}

	name, err := extractNameFromSqsQueueUrl(d.Id())
	if err != nil {
		return err
	}
	d.Set("name", name)

	if attributeOutput.Attributes != nil && len(attributeOutput.Attributes) > 0 {
		attrmap := attributeOutput.Attributes
		resource := *resourceAwsSqsQueue()
		// iKey = internal struct key, oKey = AWS Attribute Map key
		for iKey, oKey := range sqsQueueAttributeMap {
			if attrmap[oKey] != nil {
				switch resource.Schema[iKey].Type {
				case schema.TypeInt:
					value, err := strconv.Atoi(*attrmap[oKey])
					if err != nil {
						return err
					}
					d.Set(iKey, value)
					log.Printf("[DEBUG] Reading %s => %s -> %d", iKey, oKey, value)
				case schema.TypeBool:
					value, err := strconv.ParseBool(*attrmap[oKey])
					if err != nil {
						return err
					}
					d.Set(iKey, value)
					log.Printf("[DEBUG] Reading %s => %s -> %t", iKey, oKey, value)
				default:
					log.Printf("[DEBUG] Reading %s => %s -> %s", iKey, oKey, *attrmap[oKey])
					d.Set(iKey, *attrmap[oKey])
				}
			}
		}
	}

	// Since AWS does not send the FifoQueue attribute back when the queue
	// is a standard one (even to false), this enforces the queue to be set
	// to the correct value.
	d.Set("fifo_queue", d.Get("fifo_queue").(bool))
	d.Set("content_based_deduplication", d.Get("content_based_deduplication").(bool))

	tags := make(map[string]string)
	listTagsOutput, err := sqsconn.ListQueueTags(&sqs.ListQueueTagsInput{
		QueueUrl: aws.String(d.Id()),
	})
	if err != nil {
		// Non-standard partitions (e.g. US Gov) and some local development
		// solutions do not yet support this API call. Depending on the
		// implementation it may return InvalidAction or AWS.SimpleQueueService.UnsupportedOperation
		if !isAWSErr(err, "InvalidAction", "") && !isAWSErr(err, sqs.ErrCodeUnsupportedOperation, "") {
			return err
		}
	} else {
		tags = tagsToMapGeneric(listTagsOutput.Tags)
	}
	d.Set("tags", tags)

	return nil
}

func resourceAwsSqsQueueDelete(d *schema.ResourceData, meta interface{}) error {
	sqsconn := meta.(*AWSClient).sqsconn

	log.Printf("[DEBUG] SQS Delete Queue: %s", d.Id())
	_, err := sqsconn.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(d.Id()),
	})
	return err
}

func extractNameFromSqsQueueUrl(queue string) (string, error) {
	//http://sqs.us-west-2.amazonaws.com/123456789012/queueName
	u, err := url.Parse(queue)
	if err != nil {
		return "", err
	}
	segments := strings.Split(u.Path, "/")
	if len(segments) != 3 {
		return "", fmt.Errorf("SQS Url not parsed correctly")
	}

	return segments[2], nil

}

func setTagsSQS(conn *sqs.SQS, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		create, remove := diffTagsGeneric(oraw.(map[string]interface{}), nraw.(map[string]interface{}))

		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			keys := make([]*string, 0, len(remove))
			for k := range remove {
				keys = append(keys, aws.String(k))
			}

			_, err := conn.UntagQueue(&sqs.UntagQueueInput{
				QueueUrl: aws.String(d.Id()),
				TagKeys:  keys,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)

			_, err := conn.TagQueue(&sqs.TagQueueInput{
				QueueUrl: aws.String(d.Id()),
				Tags:     create,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
