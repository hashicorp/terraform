package aws

import (
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var AttributeMap = map[string]string{
	"delay_seconds":               "DelaySeconds",
	"max_message_size":            "MaximumMessageSize",
	"message_retention_seconds":   "MessageRetentionPeriod",
	"receive_wait_time_seconds":   "ReceiveMessageWaitTimeSeconds",
	"visibility_timeout_seconds":  "VisibilityTimeout",
	"policy":                      "Policy",
	"redrive_policy":              "RedrivePolicy",
	"arn":                         "QueueArn",
	"fifo_queue":                  "FifoQueue",
	"content_based_deduplication": "ContentBasedDeduplication",
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
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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
				ValidateFunc:     validateJsonString,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
			"redrive_policy": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateJsonString,
				StateFunc: func(v interface{}) string {
					json, _ := normalizeJsonString(v)
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
		},
	}
}

func resourceAwsSqsQueueCreate(d *schema.ResourceData, meta interface{}) error {
	sqsconn := meta.(*AWSClient).sqsconn

	name := d.Get("name").(string)
	fq := d.Get("fifo_queue").(bool)
	cbd := d.Get("content_based_deduplication").(bool)

	if fq {
		if errors := validateSQSFifoQueueName(name, "name"); len(errors) > 0 {
			return fmt.Errorf("Error validating the FIFO queue name: %v", errors)
		}
	} else {
		if errors := validateSQSQueueName(name, "name"); len(errors) > 0 {
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

	resource := *resourceAwsSqsQueue()

	for k, s := range resource.Schema {
		if attrKey, ok := AttributeMap[k]; ok {
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

	output, err := sqsconn.CreateQueue(req)
	if err != nil {
		return fmt.Errorf("Error creating SQS queue: %s", err)
	}

	d.SetId(*output.QueueUrl)

	return resourceAwsSqsQueueUpdate(d, meta)
}

func resourceAwsSqsQueueUpdate(d *schema.ResourceData, meta interface{}) error {
	sqsconn := meta.(*AWSClient).sqsconn
	attributes := make(map[string]*string)

	resource := *resourceAwsSqsQueue()

	for k, s := range resource.Schema {
		if attrKey, ok := AttributeMap[k]; ok {
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
			return fmt.Errorf("[ERR] Error updating SQS attributes: %s", err)
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
			if "AWS.SimpleQueueService.NonExistentQueue" == awsErr.Code() {
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
		for iKey, oKey := range AttributeMap {
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

	return nil
}

func resourceAwsSqsQueueDelete(d *schema.ResourceData, meta interface{}) error {
	sqsconn := meta.(*AWSClient).sqsconn

	log.Printf("[DEBUG] SQS Delete Queue: %s", d.Id())
	_, err := sqsconn.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	return nil
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
