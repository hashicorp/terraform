package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

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
				Computed: true,
			},
		},
	}
}

func resourceAwsLambdaEventSourceMappingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	params := getAwsLambdaCreateEventSourceMappingInput(d)

 	log.Printf("[DEBUG] Creating EventSourceMapping %#v", params)
	resp, err := conn.CreateEventSourceMapping(&params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error creating EventSourceMapping for arn %s message: \"%s\", code: \"%s\"",
				d.Get("event_source_arn").(string), awsErr.Message(), awsErr.Code())
		}
		return err
	}

 	log.Printf("[DEBUG] Created EventSourceMapping %#v", params)

	d.SetId(LambdaEventSourceMappingId(d))
	d.Set("uuid",&resp.UUID)

 	log.Printf("[DEBUG] Created EventSourceMapping with uuid %s", &resp.UUID)

	return resourceAwsLambdaEventSourceMappingRead(d, meta)
}

//func resourceAwsLambdaEventSourceMappingUpdate(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).lambdaconn
//
//	params := getAwsLambdaCreateEventSourceMappingInput(d)
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
		EventSourceArn: aws.String(event_source_arn),
		FunctionName: aws.String(function_name),
		StartingPosition:  aws.String(starting_position),
	}

//	if _, ok := d.GetOk("batch_size"); ok {
//		params.BatchSize = aws.Int64(d.Get("batch_size").(int64))
//	}

	if _, ok := d.GetOk("enabled"); ok {
		params.Enabled = aws.Bool(d.Get("enabled").(bool))
	}

	return params
}

func resourceAwsLambdaEventSourceMappingRead(d *schema.ResourceData, meta interface{}) error {
  	conn := meta.(*AWSClient).lambdaconn

 	log.Printf("[DEBUG] Reading EventSourceMapping")

	uuid := d.Get("uuid").(string)
	params := &lambda.GetEventSourceMappingInput{
		UUID: &uuid,
	}
	_, err := conn.GetEventSourceMapping(params)

	if err != nil {
		return fmt.Errorf("Error reading EventSourceMapping for uuid %s: %#v", uuid, err)
	}

	// TODO: Might need setting of d-values?
	return nil
}

func resourceAwsLambdaEventSourceMappingDelete(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).cloudwatchlogsconn
//
 	log.Printf("[DEBUG] Deleting EventSourceMapping")
//	log_group := d.Get("log_group").(string)
//	name := d.Get("name").(string)
//
//	params := &cloudwatchlogs.DeleteSubscriptionFilterInput{
//		FilterName:   aws.String(name),      // Required
//		LogGroupName: aws.String(log_group), // Required
//	}
//	_, err := conn.DeleteSubscriptionFilter(params)
//
//	if err != nil {
//		return fmt.Errorf(
//			"Error deleting Subscription Filter from log group: %s with name filter name %s", log_group, name)
//	}
//	d.SetId("")
	return nil
}

func LambdaEventSourceMappingId(d *schema.ResourceData) string {
	var buf bytes.Buffer

	event_source_arn := d.Get("event_source_arn").(string)
	function_name := d.Get("function_name").(string)
	starting_position := d.Get("starting_position").(string)
	batch_size := d.Get("batch_size").(string)
	enabled := d.Get("enabled").(string)


	buf.WriteString(fmt.Sprintf("%s-", event_source_arn))
	buf.WriteString(fmt.Sprintf("%s-", function_name))
	buf.WriteString(fmt.Sprintf("%s-", starting_position))
	buf.WriteString(fmt.Sprintf("%d-", batch_size))
	buf.WriteString(fmt.Sprintf("%v-", enabled))

	return fmt.Sprintf("lesm-%d", hashcode.String(buf.String()))
}
