package aws

import (
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/sqs"
)

func resourceAwsSQS() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSQSCreate,
		Read:   resourceAwsSQSRead,
		Update: resourceAwsSQSUpdate,
		Delete: resourceAwsSQSDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"visibility_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"retention_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  345600,
			},
			"max_message_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  262144,
			},
			"delivery_delay": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"receive_wait_time": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
		},
	}
}

func resourceAwsSQSCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn

	params := &sqs.CreateQueueInput{
		QueueName: aws.String(d.Get("name").(string)),
		Attributes: &map[string]*string{
			"VisibilityTimeout":             aws.String(strconv.Itoa(d.Get("visibility_timeout").(int))),
			"MessageRetentionPeriod":        aws.String(strconv.Itoa(d.Get("retention_period").(int))),
			"MaximumMessageSize":            aws.String(strconv.Itoa(d.Get("max_message_size").(int))),
			"DelaySeconds":                  aws.String(strconv.Itoa(d.Get("delivery_delay").(int))),
			"ReceiveMessageWaitTimeSeconds": aws.String(strconv.Itoa(d.Get("receive_wait_time").(int))),
		},
	}

	resp, err := conn.CreateQueue(params)
	if err != nil {
		return err
	}

	d.SetId(*resp.QueueURL)

	return nil
}


func resourceAwsSQSRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn

	params := &sqs.GetQueueAttributesInput{
	    QueueURL: aws.String(d.Id()), 
	}
	_, err := conn.GetQueueAttributes(params)

	if err != nil {
		return err
	}

	d.Set("queue_url", d.Id())

	return nil
}


func resourceAwsSQSUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn

	params := &sqs.SetQueueAttributesInput{
	    Attributes: &map[string]*string{ 
	        "VisibilityTimeout":             aws.String(strconv.Itoa(d.Get("visibility_timeout").(int))),
			"MessageRetentionPeriod":        aws.String(strconv.Itoa(d.Get("retention_period").(int))),
			"MaximumMessageSize":            aws.String(strconv.Itoa(d.Get("max_message_size").(int))),
			"DelaySeconds":                  aws.String(strconv.Itoa(d.Get("delivery_delay").(int))),
			"ReceiveMessageWaitTimeSeconds": aws.String(strconv.Itoa(d.Get("receive_wait_time").(int))),
	    },
	    QueueURL: aws.String(d.Id()),
	}
	_, err := conn.SetQueueAttributes(params)
	
	if err != nil {
		return err
	}

	return resourceAwsSQSRead(d, meta)
}


func resourceAwsSQSDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sqsconn

	params := &sqs.DeleteQueueInput{
	    QueueURL: aws.String(d.Id()),
	}
	_, err := conn.DeleteQueue(params)

	if err != nil {
		return err
	}

	return nil
}
