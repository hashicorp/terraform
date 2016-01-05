package aws

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/mitchellh/go-homedir"

	"errors"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLambdaFunction() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaFunctionCreate,
		Read:   resourceAwsLambdaFunctionRead,
		Update: resourceAwsLambdaFunctionUpdate,
		Delete: resourceAwsLambdaFunctionDelete,

		Schema: map[string]*schema.Schema{
			"filename": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"s3_bucket", "s3_key", "s3_object_version"},
			},
			"s3_bucket": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"filename"},
			},
			"s3_key": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"filename"},
			},
			"s3_object_version": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"filename"},
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

	var functionCode *lambda.FunctionCode
	if v, ok := d.GetOk("filename"); ok {
		filename, err := homedir.Expand(v.(string))
		if err != nil {
			return err
		}
		zipfile, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		d.Set("source_code_hash", sha256.Sum256(zipfile))
		functionCode = &lambda.FunctionCode{
			ZipFile: zipfile,
		}
	} else {
		s3Bucket, bucketOk := d.GetOk("s3_bucket")
		s3Key, keyOk := d.GetOk("s3_key")
		s3ObjectVersion, versionOk := d.GetOk("s3_object_version")
		if !bucketOk || !keyOk || !versionOk {
			return errors.New("s3_bucket, s3_key and s3_object_version must all be set while using S3 code source")
		}
		functionCode = &lambda.FunctionCode{
			S3Bucket:        aws.String(s3Bucket.(string)),
			S3Key:           aws.String(s3Key.(string)),
			S3ObjectVersion: aws.String(s3ObjectVersion.(string)),
		}
	}

	params := &lambda.CreateFunctionInput{
		Code:         functionCode,
		Description:  aws.String(d.Get("description").(string)),
		FunctionName: aws.String(functionName),
		Handler:      aws.String(d.Get("handler").(string)),
		MemorySize:   aws.Int64(int64(d.Get("memory_size").(int))),
		Role:         aws.String(iamRole),
		Runtime:      aws.String(d.Get("runtime").(string)),
		Timeout:      aws.Int64(int64(d.Get("timeout").(int))),
	}

	// IAM profiles can take ~10 seconds to propagate in AWS:
	//  http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
	// Error creating Lambda function: InvalidParameterValueException: The role defined for the task cannot be assumed by Lambda.
	err := resource.Retry(1*time.Minute, func() error {
		_, err := conn.CreateFunction(params)
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

	params := &lambda.DeleteFunctionInput{
		FunctionName: aws.String(d.Get("function_name").(string)),
	}

	_, err := conn.DeleteFunction(params)
	if err != nil {
		return fmt.Errorf("Error deleting Lambda Function: %s", err)
	}

	d.SetId("")

	return nil
}

// resourceAwsLambdaFunctionUpdate maps to:
// UpdateFunctionCode in the API / SDK
func resourceAwsLambdaFunctionUpdate(d *schema.ResourceData, meta interface{}) error {
	// conn := meta.(*AWSClient).lambdaconn

	return nil
}
