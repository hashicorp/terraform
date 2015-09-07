package aws

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/mitchellh/go-homedir"

	"github.com/hashicorp/terraform/helper/schema"
)

// Number of times to retry if a throttling-related exception occurs
const LAMBDA_MAX_THROTTLE_RETRIES = 5

// How long to sleep when a throttle-event happens
const LAMBDA_THROTTLE_SLEEP = 2 * time.Second

func resourceAwsLambdaFunction() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaFunctionCreate,
		Read:   resourceAwsLambdaFunctionRead,
		Update: resourceAwsLambdaFunctionUpdate,
		Delete: resourceAwsLambdaFunctionDelete,

		Schema: map[string]*schema.Schema{
			"filename": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true, // TODO make this editable
			},
			"function_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"handler": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true, // TODO make this editable
			},
			"memory_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  128,
				ForceNew: true, // TODO make this editable
			},
			"role": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true, // TODO make this editable
			},
			"runtime": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "nodejs",
			},
			"timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3,
				ForceNew: true, // TODO make this editable
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_modified": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_code_hash": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

// resourceAwsLambdaFunction maps to:
// CreateFunction in the API / SDK
func resourceAwsLambdaFunctionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	functionName := d.Get("function_name").(string)
	iamRole := d.Get("role").(string)

	log.Printf("[DEBUG] Creating Lambda Function %s with role %s", functionName, iamRole)

	filename, err := homedir.Expand(d.Get("filename").(string))
	if err != nil {
		return err
	}
	zipfile, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	d.Set("source_code_hash", sha256.Sum256(zipfile))

	log.Printf("[DEBUG] ")

	params := &lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: zipfile,
		},
		Description:  aws.String(d.Get("description").(string)),
		FunctionName: aws.String(functionName),
		Handler:      aws.String(d.Get("handler").(string)),
		MemorySize:   aws.Int64(int64(d.Get("memory_size").(int))),
		Role:         aws.String(iamRole),
		Runtime:      aws.String(d.Get("runtime").(string)),
		Timeout:      aws.Int64(int64(d.Get("timeout").(int))),
	}

	for i := 0; i < 5; i++ {
		_, err = conn.CreateFunction(params)
		if awsErr, ok := err.(awserr.Error); ok {

			// IAM profiles can take ~10 seconds to propagate in AWS:
			//  http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
			// Error creating Lambda function: InvalidParameterValueException: The role defined for the task cannot be assumed by Lambda.
			if awsErr.Code() == "InvalidParameterValueException" && strings.Contains(awsErr.Message(), "The role defined for the task cannot be assumed by Lambda.") {
				log.Printf("[DEBUG] Invalid IAM Instance Profile referenced, retrying...")
				time.Sleep(2 * time.Second)
				continue
			} else if awsErr.Code() == "TooManyRequestsException" {
				log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back create request", i, 5)
				time.Sleep(2 * time.Second)
				continue
			}
		}
		break
	}
	if err != nil {
		return fmt.Errorf("Error creating Lambda function: %s", err)
	}

	d.SetId(d.Get("function_name").(string))

	return resourceAwsLambdaFunctionRead(d, meta)
}

// resourceAwsLambdaFunctionRead maps to:
// GetFunction in the API / SDK
func resourceAwsLambdaFunctionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[DEBUG] Fetching Lambda Function: %s", d.Id())

	params := &lambda.GetFunctionInput{
		FunctionName: aws.String(d.Get("function_name").(string)),
	}

	getFunctionOutput, err := conn.GetFunction(params)
	if err != nil {
		return err
	}

	// getFunctionOutput.Code.Location is a pre-signed URL pointing at the zip
	// file that we uploaded when we created the resource. You can use it to
	// download the code from AWS. The other part is
	// getFunctionOutput.Configuration which holds metadata.

	function := getFunctionOutput.Configuration
	// TODO error checking / handling on the Set() calls.
	d.Set("arn", function.FunctionArn)
	d.Set("description", function.Description)
	d.Set("handler", function.Handler)
	d.Set("memory_size", function.MemorySize)
	d.Set("last_modified", function.LastModified)
	d.Set("role", function.Role)
	d.Set("runtime", function.Runtime)
	d.Set("timeout", function.Timeout)

	return nil
}

// resourceAwsLambdaFunction maps to:
// DeleteFunction in the API / SDK
func resourceAwsLambdaFunctionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	log.Printf("[INFO] Deleting Lambda Function: %s", d.Id())

	attemptCount := 1
	for attemptCount <= LAMBDA_MAX_THROTTLE_RETRIES {
		_, err := conn.DeleteFunction(&lambda.DeleteFunctionInput{
			FunctionName: aws.String(d.Get("function_name").(string)),
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "TooManyRequestsException" {
					log.Printf("[DEBUG] Attempt %d/%d: Sleeping for a bit to throttle back create request", attemptCount, LAMBDA_MAX_THROTTLE_RETRIES)
					time.Sleep(LAMBDA_THROTTLE_SLEEP)
					attemptCount += 1
				} else if awsErr.Code() == "ResourceNotFoundException" {
					log.Printf("[DEBUG] Function no longer exists - that's actually OK")
					d.SetId("")
					return nil
				} else {
					// Some other non-retryable exception occurred
					return fmt.Errorf("AWS Error deleting Lambda function: %s", err)
				}
			} else {
				// Non-AWS exception occurred, give up
				return fmt.Errorf("Error deleting Lambda function: %s", err)
			}
		} else {
			d.SetId("")
			return nil
		}
	}

	// Too many throttling events occurred, give up
	return fmt.Errorf("Unable to delete Lambda function '%s' after %d attempts", d.Id(), attemptCount)
}

// resourceAwsLambdaFunctionUpdate maps to:
// UpdateFunctionCode in the API / SDK
func resourceAwsLambdaFunctionUpdate(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*AWSClient).lambdaconn

	return nil
}
