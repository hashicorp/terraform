package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/schema"
)

// Number of times to retry if a throttling-related exception occurs
const LAMBDA_EVENT_SOURCE_MAPPING_MAX_THROTTLE_RETRIES = 10

// How long to sleep when a throttle-event happens
const LAMBDA_EVENT_SOURCE_MAPPING_THROTTLE_SLEEP = 2 * time.Second

func resourceAwsLambdaEventSourceMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaEventSourceMappingCreate,
		Read:   resourceAwsLambdaEventSourceMappingRead,
		//		Update: resourceAwsLambdaEventSourceMappingUpdate,
		Delete: resourceAwsLambdaEventSourceMappingDelete,

		Schema: map[string]*schema.Schema{
			"event_source_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"function_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"starting_position": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"batch_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsLambdaEventSourceMappingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	params := getAwsLambdaCreateEventSourceMappingInput(d)

	log.Printf("[DEBUG] Creating EventSourceMapping %#v", params)

	var err error
	attemptCount := 1
	for attemptCount <= LAMBDA_EVENT_SOURCE_MAPPING_MAX_THROTTLE_RETRIES {
		resp, err := conn.CreateEventSourceMapping(&params)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				log.Printf("[DEBUG] AWS Error getting EventSourceMapping: [%s] %s", awsErr.Code(), awsErr.Message())
				// CreateEventSourceMapping seems to be able to cast InvalidParameterValueException in places of throttling exceptions when
				// creating many Event Source Mappings for the same stream
				if awsErr.Code() == "TooManyRequestsException" || (awsErr.Code() == "InvalidParameterValueException" && strings.Contains(awsErr.Message(), "Please ensure the role can perform the GetRecords, GetShardIterator, DescribeStream, and ListStreams Actions on your stream in IAM")) {
					log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back create request", attemptCount, DYNAMODB_MAX_THROTTLE_RETRIES)
					time.Sleep(LAMBDA_EVENT_SOURCE_MAPPING_THROTTLE_SLEEP)
					attemptCount += 1
				} else {
					return fmt.Errorf("[WARN] AWS Error creating EventSourceMapping for %s message: \"%s\", code: \"%s\"",
						d.Get("event_source_arn").(string), awsErr.Message(), awsErr.Code())
				}
			} else {
				return fmt.Errorf("Error creating EventSourceMapping: %s", err)
			}
		} else {
			log.Printf("[DEBUG] Created EventSourceMapping with uuid %s", *resp.UUID)
			d.Set("event_source_arn", *resp.EventSourceArn)
			d.Set("function_name", *resp.FunctionArn)
			d.Set("batch_size", resp.BatchSize)
			d.SetId(*resp.UUID)
			return resourceAwsLambdaEventSourceMappingRead(d, meta)
		}
	}

	return fmt.Errorf("Error creating EventSourceMapping %#v: %#v", params, err)
}

//func resourceAwsLambdaEventSourceMappingUpdate(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).lambdaconn
//
//	params := getAwsLambdaCreateEventSourceMappingInput(d)
//  tager UUID og de variable parametre:
//		BatchSize:    aws.Int64(1),
//		Enabled:      aws.Bool(true),
//		FunctionName: aws.String("FunctionName"),

//
// 	log.Printf("[DEBUG] Updating EventSourceMapping")
//	_, err := conn.UpdateEventSourceMapping(&params)
//	if err != nil {
//		if awsErr, ok := err.(awserr.Error); ok {
//			return fmt.Errorf("[WARN] Error updating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
//				d.Get("name").(string), d.Get("log_group").(string), awsErr.Message(), awsErr.Code())
//		}
//		return err
//	}
//
//	d.SetId(LambdaEventSourceMappingId(d))
//	return resourceAwsLambdaEventSourceMappingRead(d, meta)
//}

func getAwsLambdaCreateEventSourceMappingInput(d *schema.ResourceData) lambda.CreateEventSourceMappingInput {
	event_source_arn := d.Get("event_source_arn").(string)
	function_name := d.Get("function_name").(string)
	starting_position := d.Get("starting_position").(string)

	params := lambda.CreateEventSourceMappingInput{
		EventSourceArn:   aws.String(event_source_arn),
		FunctionName:     aws.String(function_name),
		StartingPosition: aws.String(starting_position),
	}

	if _, ok := d.GetOk("batch_size"); ok {
		batch_size := d.Get("batch_size").(int)
		params.BatchSize = aws.Int64(int64(batch_size))
	}

	if _, ok := d.GetOk("enabled"); ok {
		params.Enabled = aws.Bool(d.Get("enabled").(bool))
	}

	return params
}

func resourceAwsLambdaEventSourceMappingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[DEBUG] Reading EventSourceMapping")

	var uuid = d.Id()
	params := &lambda.GetEventSourceMappingInput{
		UUID: aws.String(uuid),
	}

	var err error
	attemptCount := 1
	for attemptCount <= LAMBDA_EVENT_SOURCE_MAPPING_MAX_THROTTLE_RETRIES {
		resp, err := conn.GetEventSourceMapping(params)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				log.Printf("[DEBUG] AWS Error getting EventSourceMapping: [%s] %s", awsErr.Code(), awsErr.Message())
				log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back create request", attemptCount, DYNAMODB_MAX_THROTTLE_RETRIES)
				time.Sleep(LAMBDA_EVENT_SOURCE_MAPPING_THROTTLE_SLEEP)
				attemptCount += 1
			} else {
				return fmt.Errorf("Error creating EventSourceMapping: %s", err)
			}
		} else {
			d.Set("event_source_arn", *resp.EventSourceArn)
			d.Set("function_name", *resp.FunctionArn)
			d.Set("batch_size", resp.BatchSize)
			return nil
		}
	}

	return fmt.Errorf("Error reading EventSourceMapping for uuid %s: %#v", d.Id(), err)
}

func resourceAwsLambdaEventSourceMappingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[DEBUG] Deleting EventSourceMapping")

	var uuid = d.Id()
	params := &lambda.DeleteEventSourceMappingInput{
		UUID: aws.String(uuid), // Required
	}

	var err error
	attemptCount := 1
	for attemptCount <= LAMBDA_EVENT_SOURCE_MAPPING_MAX_THROTTLE_RETRIES {
		_, err := conn.DeleteEventSourceMapping(params)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "TooManyRequestsException" {
					log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back delete request", attemptCount, DYNAMODB_MAX_THROTTLE_RETRIES)
					time.Sleep(LAMBDA_EVENT_SOURCE_MAPPING_THROTTLE_SLEEP)
					attemptCount += 1
				} else {
					return fmt.Errorf("Error deleting EventSourceMapping %s: %s", uuid, err)
				}
			} else {
				return fmt.Errorf("Error deleting EventSourceMapping %s: %s", uuid, err)
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("Error deleting EventSourceMapping %s: %s", uuid, err)
}

//func LambdaEventSourceMappingId(d *schema.ResourceData) string {
//	var buf bytes.Buffer
//
//	event_source_arn := d.Get("event_source_arn").(string)
//	function_name := d.Get("function_name").(string)
//	starting_position := d.Get("starting_position").(string)
//	batch_size := d.Get("batch_size").(int)
//	enabled := d.Get("enabled").(bool)
//
//	buf.WriteString(fmt.Sprintf("%s-", event_source_arn))
//	buf.WriteString(fmt.Sprintf("%s-", function_name))
//	buf.WriteString(fmt.Sprintf("%s-", starting_position))
//	buf.WriteString(fmt.Sprintf("%d-", batch_size))
//	buf.WriteString(fmt.Sprintf("%v-", enabled))
//
//	return fmt.Sprintf("lesm-%d", hashcode.String(buf.String()))
//}
