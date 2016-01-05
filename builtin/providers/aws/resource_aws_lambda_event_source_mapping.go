package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLambdaEventSourceMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaEventSourceMappingCreate,
		Read:   resourceAwsLambdaEventSourceMappingRead,
		Update: resourceAwsLambdaEventSourceMappingUpdate,
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
			},
			"starting_position": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"batch_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"function_arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_modified": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_processing_result": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state_transition_reason": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// resourceAwsLambdaEventSourceMappingCreate maps to:
// CreateEventSourceMapping in the API / SDK
func resourceAwsLambdaEventSourceMappingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	functionName := d.Get("function_name").(string)
	eventSourceArn := d.Get("event_source_arn").(string)

	log.Printf("[DEBUG] Creating Lambda event source mapping: source %s to function %s", eventSourceArn, functionName)

	params := &lambda.CreateEventSourceMappingInput{
		EventSourceArn:   aws.String(eventSourceArn),
		FunctionName:     aws.String(functionName),
		StartingPosition: aws.String(d.Get("starting_position").(string)),
		BatchSize:        aws.Int64(int64(d.Get("batch_size").(int))),
		Enabled:          aws.Bool(d.Get("enabled").(bool)),
	}

	// IAM profiles and roles can take some time to propagate in AWS:
	//  http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
	// Error creating Lambda function: InvalidParameterValueException: The
	// function defined for the task cannot be assumed by Lambda.
	//
	// The role may exist, but the permissions may not have propagated, so we
	// retry
	err := resource.Retry(1*time.Minute, func() error {
		eventSourceMappingConfiguration, err := conn.CreateEventSourceMapping(params)
		if err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				if awserr.Code() == "InvalidParameterValueException" {
					// Retryable
					return awserr
				}
			}
			// Not retryable
			return resource.RetryError{Err: err}
		}
		// No error
		d.Set("uuid", eventSourceMappingConfiguration.UUID)
		d.SetId(*eventSourceMappingConfiguration.UUID)
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating Lambda event source mapping: %s", err)
	}

	return resourceAwsLambdaEventSourceMappingRead(d, meta)
}

// resourceAwsLambdaEventSourceMappingRead maps to:
// GetEventSourceMapping in the API / SDK
func resourceAwsLambdaEventSourceMappingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[DEBUG] Fetching Lambda event source mapping: %s", d.Id())

	params := &lambda.GetEventSourceMappingInput{
		UUID: aws.String(d.Id()),
	}

	eventSourceMappingConfiguration, err := conn.GetEventSourceMapping(params)
	if err != nil {
		return err
	}

	d.Set("batch_size", eventSourceMappingConfiguration.BatchSize)
	d.Set("event_source_arn", eventSourceMappingConfiguration.EventSourceArn)
	d.Set("function_arn", eventSourceMappingConfiguration.FunctionArn)
	d.Set("last_modified", eventSourceMappingConfiguration.LastModified)
	d.Set("last_processing_result", eventSourceMappingConfiguration.LastProcessingResult)
	d.Set("state", eventSourceMappingConfiguration.State)
	d.Set("state_transition_reason", eventSourceMappingConfiguration.StateTransitionReason)
	d.Set("uuid", eventSourceMappingConfiguration.UUID)

	return nil
}

// resourceAwsLambdaEventSourceMappingDelete maps to:
// DeleteEventSourceMapping in the API / SDK
func resourceAwsLambdaEventSourceMappingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[INFO] Deleting Lambda event source mapping: %s", d.Id())

	params := &lambda.DeleteEventSourceMappingInput{
		UUID: aws.String(d.Id()),
	}

	_, err := conn.DeleteEventSourceMapping(params)
	if err != nil {
		return fmt.Errorf("Error deleting Lambda event source mapping: %s", err)
	}

	d.SetId("")

	return nil
}

// resourceAwsLambdaEventSourceMappingUpdate maps to:
// UpdateEventSourceMapping in the API / SDK
func resourceAwsLambdaEventSourceMappingUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[DEBUG] Updating Lambda event source mapping: %s", d.Id())

	params := &lambda.UpdateEventSourceMappingInput{
		UUID:         aws.String(d.Id()),
		BatchSize:    aws.Int64(int64(d.Get("batch_size").(int))),
		FunctionName: aws.String(d.Get("function_name").(string)),
		Enabled:      aws.Bool(d.Get("enabled").(bool)),
	}

	err := resource.Retry(1*time.Minute, func() error {
		_, err := conn.UpdateEventSourceMapping(params)
		if err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				if awserr.Code() == "InvalidParameterValueException" {
					// Retryable
					return awserr
				}
			}
			// Not retryable
			return resource.RetryError{Err: err}
		}
		// No error
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error updating Lambda event source mapping: %s", err)
	}

	return resourceAwsLambdaEventSourceMappingRead(d, meta)
}
