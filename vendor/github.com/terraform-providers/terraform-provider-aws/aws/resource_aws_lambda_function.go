package aws

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/mitchellh/go-homedir"

	"errors"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

const awsMutexLambdaKey = `aws_lambda_function`

func resourceAwsLambdaFunction() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaFunctionCreate,
		Read:   resourceAwsLambdaFunctionRead,
		Update: resourceAwsLambdaFunctionUpdate,
		Delete: resourceAwsLambdaFunctionDelete,

		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				d.Set("function_name", d.Id())
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"filename": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"s3_bucket", "s3_key", "s3_object_version"},
			},
			"s3_bucket": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"filename"},
			},
			"s3_key": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"filename"},
			},
			"s3_object_version": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"filename"},
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dead_letter_config": {
				Type:     schema.TypeList,
				Optional: true,
				MinItems: 0,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
						},
					},
				},
			},
			"function_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"handler": {
				Type:     schema.TypeString,
				Required: true,
			},
			"memory_size": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  128,
			},
			"reserved_concurrent_executions": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"role": {
				Type:     schema.TypeString,
				Required: true,
			},
			"runtime": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					// lambda.RuntimeNodejs has reached end of life since October 2016 so not included here
					lambda.RuntimeDotnetcore10,
					lambda.RuntimeDotnetcore20,
					lambda.RuntimeDotnetcore21,
					lambda.RuntimeGo1X,
					lambda.RuntimeJava8,
					lambda.RuntimeNodejs43,
					lambda.RuntimeNodejs43Edge,
					lambda.RuntimeNodejs610,
					lambda.RuntimeNodejs810,
					lambda.RuntimePython27,
					lambda.RuntimePython36,
				}, false),
			},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3,
			},
			"publish": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_config": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_ids": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"security_group_ids": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"qualified_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"invoke_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_modified": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_code_hash": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"source_code_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"environment": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"variables": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"tracing_config": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"Active", "PassThrough"}, true),
						},
					},
				},
			},
			"kms_key_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"tags": tagsSchema(),
		},

		CustomizeDiff: updateComputedAttributesOnPublish,
	}
}

func updateComputedAttributesOnPublish(d *schema.ResourceDiff, meta interface{}) error {
	if needsFunctionCodeUpdate(d) {
		d.SetNewComputed("last_modified")
		publish := d.Get("publish").(bool)
		if publish {
			d.SetNewComputed("version")
			d.SetNewComputed("qualified_arn")
		}
	}
	return nil
}

// resourceAwsLambdaFunction maps to:
// CreateFunction in the API / SDK
func resourceAwsLambdaFunctionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	functionName := d.Get("function_name").(string)
	reservedConcurrentExecutions := d.Get("reserved_concurrent_executions").(int)
	iamRole := d.Get("role").(string)

	log.Printf("[DEBUG] Creating Lambda Function %s with role %s", functionName, iamRole)

	filename, hasFilename := d.GetOk("filename")
	s3Bucket, bucketOk := d.GetOk("s3_bucket")
	s3Key, keyOk := d.GetOk("s3_key")
	s3ObjectVersion, versionOk := d.GetOk("s3_object_version")

	if !hasFilename && !bucketOk && !keyOk && !versionOk {
		return errors.New("filename or s3_* attributes must be set")
	}

	var functionCode *lambda.FunctionCode
	if hasFilename {
		// Grab an exclusive lock so that we're only reading one function into
		// memory at a time.
		// See https://github.com/hashicorp/terraform/issues/9364
		awsMutexKV.Lock(awsMutexLambdaKey)
		defer awsMutexKV.Unlock(awsMutexLambdaKey)
		file, err := loadFileContent(filename.(string))
		if err != nil {
			return fmt.Errorf("Unable to load %q: %s", filename.(string), err)
		}
		functionCode = &lambda.FunctionCode{
			ZipFile: file,
		}
	} else {
		if !bucketOk || !keyOk {
			return errors.New("s3_bucket and s3_key must all be set while using S3 code source")
		}
		functionCode = &lambda.FunctionCode{
			S3Bucket: aws.String(s3Bucket.(string)),
			S3Key:    aws.String(s3Key.(string)),
		}
		if versionOk {
			functionCode.S3ObjectVersion = aws.String(s3ObjectVersion.(string))
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
		Publish:      aws.Bool(d.Get("publish").(bool)),
	}

	if v, ok := d.GetOk("dead_letter_config"); ok {
		dlcMaps := v.([]interface{})
		if len(dlcMaps) == 1 { // Schema guarantees either 0 or 1
			// Prevent panic on nil dead_letter_config. See GH-14961
			if dlcMaps[0] == nil {
				return fmt.Errorf("Nil dead_letter_config supplied for function: %s", functionName)
			}
			dlcMap := dlcMaps[0].(map[string]interface{})
			params.DeadLetterConfig = &lambda.DeadLetterConfig{
				TargetArn: aws.String(dlcMap["target_arn"].(string)),
			}
		}
	}

	if v, ok := d.GetOk("vpc_config"); ok {

		configs := v.([]interface{})
		config, ok := configs[0].(map[string]interface{})

		if !ok {
			return errors.New("vpc_config is <nil>")
		}

		if config != nil {
			var subnetIds []*string
			for _, id := range config["subnet_ids"].(*schema.Set).List() {
				subnetIds = append(subnetIds, aws.String(id.(string)))
			}

			var securityGroupIds []*string
			for _, id := range config["security_group_ids"].(*schema.Set).List() {
				securityGroupIds = append(securityGroupIds, aws.String(id.(string)))
			}

			params.VpcConfig = &lambda.VpcConfig{
				SubnetIds:        subnetIds,
				SecurityGroupIds: securityGroupIds,
			}
		}
	}

	if v, ok := d.GetOk("tracing_config"); ok {
		tracingConfig := v.([]interface{})
		tracing := tracingConfig[0].(map[string]interface{})
		params.TracingConfig = &lambda.TracingConfig{
			Mode: aws.String(tracing["mode"].(string)),
		}
	}

	if v, ok := d.GetOk("environment"); ok {
		environments := v.([]interface{})
		environment, ok := environments[0].(map[string]interface{})
		if !ok {
			return errors.New("At least one field is expected inside environment")
		}

		if environmentVariables, ok := environment["variables"]; ok {
			variables := readEnvironmentVariables(environmentVariables.(map[string]interface{}))

			params.Environment = &lambda.Environment{
				Variables: aws.StringMap(variables),
			}
		}
	}

	if v, ok := d.GetOk("kms_key_arn"); ok {
		params.KMSKeyArn = aws.String(v.(string))
	}

	if v, exists := d.GetOk("tags"); exists {
		params.Tags = tagsFromMapGeneric(v.(map[string]interface{}))
	}

	// IAM changes can take 1 minute to propagate in AWS
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.CreateFunction(params)
		if err != nil {
			log.Printf("[DEBUG] Error creating Lambda Function: %s", err)

			if isAWSErr(err, "InvalidParameterValueException", "The role defined for the function cannot be assumed by Lambda") {
				log.Printf("[DEBUG] Received %s, retrying CreateFunction", err)
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "InvalidParameterValueException", "The provided execution role does not have permissions") {
				log.Printf("[DEBUG] Received %s, retrying CreateFunction", err)
				return resource.RetryableError(err)
			}
			if isAWSErr(err, "InvalidParameterValueException", "Your request has been throttled by EC2") {
				log.Printf("[DEBUG] Received %s, retrying CreateFunction", err)
				return resource.RetryableError(err)
			}

			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		if !isAWSErr(err, "InvalidParameterValueException", "Your request has been throttled by EC2") {
			return fmt.Errorf("Error creating Lambda function: %s", err)
		}
		// Allow 9 more minutes for EC2 throttling
		err := resource.Retry(9*time.Minute, func() *resource.RetryError {
			_, err := conn.CreateFunction(params)
			if err != nil {
				log.Printf("[DEBUG] Error creating Lambda Function: %s", err)

				if isAWSErr(err, "InvalidParameterValueException", "Your request has been throttled by EC2") {
					log.Printf("[DEBUG] Received %s, retrying CreateFunction", err)
					return resource.RetryableError(err)
				}

				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error creating Lambda function: %s", err)
		}
	}

	d.SetId(d.Get("function_name").(string))

	if reservedConcurrentExecutions > 0 {

		log.Printf("[DEBUG] Setting Concurrency to %d for the Lambda Function %s", reservedConcurrentExecutions, functionName)

		concurrencyParams := &lambda.PutFunctionConcurrencyInput{
			FunctionName:                 aws.String(functionName),
			ReservedConcurrentExecutions: aws.Int64(int64(reservedConcurrentExecutions)),
		}

		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			_, err := conn.PutFunctionConcurrency(concurrencyParams)
			if err != nil {
				if isAWSErr(err, lambda.ErrCodeResourceNotFoundException, "") {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error setting concurrency for Lambda %s: %s", functionName, err)
		}
	}

	return resourceAwsLambdaFunctionRead(d, meta)
}

// resourceAwsLambdaFunctionRead maps to:
// GetFunction in the API / SDK
func resourceAwsLambdaFunctionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	params := &lambda.GetFunctionInput{
		FunctionName: aws.String(d.Get("function_name").(string)),
	}

	// qualifier for lambda function data source
	qualifier, qualifierExistance := d.GetOk("qualifier")
	if qualifierExistance {
		params.Qualifier = aws.String(qualifier.(string))
		log.Printf("[DEBUG] Fetching Lambda Function: %s:%s", d.Id(), qualifier.(string))
	} else {
		log.Printf("[DEBUG] Fetching Lambda Function: %s", d.Id())
	}

	getFunctionOutput, err := conn.GetFunction(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" && !d.IsNewResource() {
			d.SetId("")
			return nil
		}
		return err
	}

	if getFunctionOutput.Concurrency != nil {
		d.Set("reserved_concurrent_executions", getFunctionOutput.Concurrency.ReservedConcurrentExecutions)
	} else {
		d.Set("reserved_concurrent_executions", nil)
	}

	// Tagging operations are permitted on Lambda functions only.
	// Tags on aliases and versions are not supported.
	if !qualifierExistance {
		d.Set("tags", tagsToMapGeneric(getFunctionOutput.Tags))
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
	d.Set("kms_key_arn", function.KMSKeyArn)
	d.Set("source_code_hash", function.CodeSha256)
	d.Set("source_code_size", function.CodeSize)

	config := flattenLambdaVpcConfigResponse(function.VpcConfig)
	log.Printf("[INFO] Setting Lambda %s VPC config %#v from API", d.Id(), config)
	if err := d.Set("vpc_config", config); err != nil {
		return fmt.Errorf("[ERR] Error setting vpc_config for Lambda Function (%s): %s", d.Id(), err)
	}

	environment := flattenLambdaEnvironment(function.Environment)
	log.Printf("[INFO] Setting Lambda %s environment %#v from API", d.Id(), environment)
	if err := d.Set("environment", environment); err != nil {
		log.Printf("[ERR] Error setting environment for Lambda Function (%s): %s", d.Id(), err)
	}

	if function.DeadLetterConfig != nil && function.DeadLetterConfig.TargetArn != nil {
		d.Set("dead_letter_config", []interface{}{
			map[string]interface{}{
				"target_arn": *function.DeadLetterConfig.TargetArn,
			},
		})
	} else {
		d.Set("dead_letter_config", []interface{}{})
	}

	// Assume `PassThrough` on partitions that don't support tracing config
	tracingConfigMode := "PassThrough"
	if function.TracingConfig != nil {
		tracingConfigMode = *function.TracingConfig.Mode
	}
	d.Set("tracing_config", []interface{}{
		map[string]interface{}{
			"mode": tracingConfigMode,
		},
	})

	// Get latest version and ARN unless qualifier is specified via data source
	if qualifierExistance {
		d.Set("version", function.Version)
		d.Set("qualified_arn", function.FunctionArn)
	} else {
		// List is sorted from oldest to latest
		// so this may get costly over time :'(
		var lastVersion, lastQualifiedArn string
		err = listVersionsByFunctionPages(conn, &lambda.ListVersionsByFunctionInput{
			FunctionName: function.FunctionName,
			MaxItems:     aws.Int64(10000),
		}, func(p *lambda.ListVersionsByFunctionOutput, lastPage bool) bool {
			if lastPage {
				last := p.Versions[len(p.Versions)-1]
				lastVersion = *last.Version
				lastQualifiedArn = *last.FunctionArn
				return false
			}
			return true
		})
		if err != nil {
			return err
		}

		d.Set("version", lastVersion)
		d.Set("qualified_arn", lastQualifiedArn)
	}

	invokeArn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "apigateway",
		Region:    meta.(*AWSClient).region,
		AccountID: "lambda",
		Resource:  fmt.Sprintf("path/2015-03-31/functions/%s/invocations", *function.FunctionArn),
	}.String()
	d.Set("invoke_arn", invokeArn)

	return nil
}

func listVersionsByFunctionPages(c *lambda.Lambda, input *lambda.ListVersionsByFunctionInput,
	fn func(p *lambda.ListVersionsByFunctionOutput, lastPage bool) bool) error {
	for {
		page, err := c.ListVersionsByFunction(input)
		if err != nil {
			return err
		}
		lastPage := page.NextMarker == nil

		shouldContinue := fn(page, lastPage)
		if !shouldContinue || lastPage {
			break
		}
		input.Marker = page.NextMarker
	}
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

	return nil
}

type resourceDiffer interface {
	HasChange(string) bool
}

func needsFunctionCodeUpdate(d resourceDiffer) bool {
	return d.HasChange("filename") || d.HasChange("source_code_hash") || d.HasChange("s3_bucket") || d.HasChange("s3_key") || d.HasChange("s3_object_version")
}

// resourceAwsLambdaFunctionUpdate maps to:
// UpdateFunctionCode in the API / SDK
func resourceAwsLambdaFunctionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	d.Partial(true)

	arn := d.Get("arn").(string)
	if tagErr := setTagsLambda(conn, d, arn); tagErr != nil {
		return tagErr
	}
	d.SetPartial("tags")

	configReq := &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(d.Id()),
	}

	configUpdate := false
	if d.HasChange("description") {
		configReq.Description = aws.String(d.Get("description").(string))
		configUpdate = true
	}
	if d.HasChange("handler") {
		configReq.Handler = aws.String(d.Get("handler").(string))
		configUpdate = true
	}
	if d.HasChange("memory_size") {
		configReq.MemorySize = aws.Int64(int64(d.Get("memory_size").(int)))
		configUpdate = true
	}
	if d.HasChange("role") {
		configReq.Role = aws.String(d.Get("role").(string))
		configUpdate = true
	}
	if d.HasChange("timeout") {
		configReq.Timeout = aws.Int64(int64(d.Get("timeout").(int)))
		configUpdate = true
	}
	if d.HasChange("kms_key_arn") {
		configReq.KMSKeyArn = aws.String(d.Get("kms_key_arn").(string))
		configUpdate = true
	}
	if d.HasChange("dead_letter_config") {
		dlcMaps := d.Get("dead_letter_config").([]interface{})
		configReq.DeadLetterConfig = &lambda.DeadLetterConfig{
			TargetArn: aws.String(""),
		}
		if len(dlcMaps) == 1 { // Schema guarantees either 0 or 1
			dlcMap := dlcMaps[0].(map[string]interface{})
			configReq.DeadLetterConfig.TargetArn = aws.String(dlcMap["target_arn"].(string))
		}
		configUpdate = true
	}
	if d.HasChange("tracing_config") {
		tracingConfig := d.Get("tracing_config").([]interface{})
		if len(tracingConfig) == 1 { // Schema guarantees either 0 or 1
			config := tracingConfig[0].(map[string]interface{})
			configReq.TracingConfig = &lambda.TracingConfig{
				Mode: aws.String(config["mode"].(string)),
			}
			configUpdate = true
		}
	}
	if d.HasChange("vpc_config") {
		vpcConfigRaw := d.Get("vpc_config").([]interface{})
		vpcConfig, ok := vpcConfigRaw[0].(map[string]interface{})
		if !ok {
			return errors.New("vpc_config is <nil>")
		}

		if vpcConfig != nil {
			var subnetIds []*string
			for _, id := range vpcConfig["subnet_ids"].(*schema.Set).List() {
				subnetIds = append(subnetIds, aws.String(id.(string)))
			}

			var securityGroupIds []*string
			for _, id := range vpcConfig["security_group_ids"].(*schema.Set).List() {
				securityGroupIds = append(securityGroupIds, aws.String(id.(string)))
			}

			configReq.VpcConfig = &lambda.VpcConfig{
				SubnetIds:        subnetIds,
				SecurityGroupIds: securityGroupIds,
			}
			configUpdate = true
		}
	}

	if d.HasChange("runtime") {
		configReq.Runtime = aws.String(d.Get("runtime").(string))
		configUpdate = true
	}
	if d.HasChange("environment") {
		if v, ok := d.GetOk("environment"); ok {
			environments := v.([]interface{})
			environment, ok := environments[0].(map[string]interface{})
			if !ok {
				return errors.New("At least one field is expected inside environment")
			}

			if environmentVariables, ok := environment["variables"]; ok {
				variables := readEnvironmentVariables(environmentVariables.(map[string]interface{}))

				configReq.Environment = &lambda.Environment{
					Variables: aws.StringMap(variables),
				}
				configUpdate = true
			}
		} else {
			configReq.Environment = &lambda.Environment{
				Variables: aws.StringMap(map[string]string{}),
			}
			configUpdate = true
		}
	}

	if configUpdate {
		log.Printf("[DEBUG] Send Update Lambda Function Configuration request: %#v", configReq)

		// IAM changes can take 1 minute to propagate in AWS
		err := resource.Retry(1*time.Minute, func() *resource.RetryError {
			_, err := conn.UpdateFunctionConfiguration(configReq)
			if err != nil {
				log.Printf("[DEBUG] Received error modifying Lambda Function Configuration %s: %s", d.Id(), err)

				if isAWSErr(err, "InvalidParameterValueException", "The role defined for the function cannot be assumed by Lambda") {
					log.Printf("[DEBUG] Received %s, retrying UpdateFunctionConfiguration", err)
					return resource.RetryableError(err)
				}
				if isAWSErr(err, "InvalidParameterValueException", "The provided execution role does not have permissions") {
					log.Printf("[DEBUG] Received %s, retrying UpdateFunctionConfiguration", err)
					return resource.RetryableError(err)
				}
				if isAWSErr(err, "InvalidParameterValueException", "Your request has been throttled by EC2, please make sure you have enough API rate limit.") {
					log.Printf("[DEBUG] Received %s, retrying UpdateFunctionConfiguration", err)
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			if !isAWSErr(err, "InvalidParameterValueException", "Your request has been throttled by EC2, please make sure you have enough API rate limit.") {
				return fmt.Errorf("Error modifying Lambda Function Configuration %s: %s", d.Id(), err)
			}
			// Allow 9 more minutes for EC2 throttling
			err := resource.Retry(9*time.Minute, func() *resource.RetryError {
				_, err := conn.UpdateFunctionConfiguration(configReq)
				if err != nil {
					log.Printf("[DEBUG] Received error modifying Lambda Function Configuration %s: %s", d.Id(), err)

					if isAWSErr(err, "InvalidParameterValueException", "Your request has been throttled by EC2, please make sure you have enough API rate limit.") {
						log.Printf("[DEBUG] Received %s, retrying UpdateFunctionConfiguration", err)
						return resource.RetryableError(err)
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("Error modifying Lambda Function Configuration %s: %s", d.Id(), err)
			}
		}

		d.SetPartial("description")
		d.SetPartial("handler")
		d.SetPartial("memory_size")
		d.SetPartial("role")
		d.SetPartial("timeout")
	}

	if needsFunctionCodeUpdate(d) {
		codeReq := &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(d.Id()),
			Publish:      aws.Bool(d.Get("publish").(bool)),
		}

		if v, ok := d.GetOk("filename"); ok {
			// Grab an exclusive lock so that we're only reading one function into
			// memory at a time.
			// See https://github.com/hashicorp/terraform/issues/9364
			awsMutexKV.Lock(awsMutexLambdaKey)
			defer awsMutexKV.Unlock(awsMutexLambdaKey)
			file, err := loadFileContent(v.(string))
			if err != nil {
				return fmt.Errorf("Unable to load %q: %s", v.(string), err)
			}
			codeReq.ZipFile = file
		} else {
			s3Bucket, _ := d.GetOk("s3_bucket")
			s3Key, _ := d.GetOk("s3_key")
			s3ObjectVersion, versionOk := d.GetOk("s3_object_version")

			codeReq.S3Bucket = aws.String(s3Bucket.(string))
			codeReq.S3Key = aws.String(s3Key.(string))
			if versionOk {
				codeReq.S3ObjectVersion = aws.String(s3ObjectVersion.(string))
			}
		}

		log.Printf("[DEBUG] Send Update Lambda Function Code request: %#v", codeReq)

		_, err := conn.UpdateFunctionCode(codeReq)
		if err != nil {
			return fmt.Errorf("Error modifying Lambda Function Code %s: %s", d.Id(), err)
		}

		d.SetPartial("filename")
		d.SetPartial("source_code_hash")
		d.SetPartial("s3_bucket")
		d.SetPartial("s3_key")
		d.SetPartial("s3_object_version")
	}

	if d.HasChange("reserved_concurrent_executions") {
		nc := d.Get("reserved_concurrent_executions")

		if nc.(int) > 0 {
			log.Printf("[DEBUG] Updating Concurrency to %d for the Lambda Function %s", nc.(int), d.Id())

			concurrencyParams := &lambda.PutFunctionConcurrencyInput{
				FunctionName:                 aws.String(d.Id()),
				ReservedConcurrentExecutions: aws.Int64(int64(d.Get("reserved_concurrent_executions").(int))),
			}

			_, err := conn.PutFunctionConcurrency(concurrencyParams)
			if err != nil {
				return fmt.Errorf("Error setting concurrency for Lambda %s: %s", d.Id(), err)
			}
		} else {
			log.Printf("[DEBUG] Removing Concurrency for the Lambda Function %s", d.Id())

			deleteConcurrencyParams := &lambda.DeleteFunctionConcurrencyInput{
				FunctionName: aws.String(d.Id()),
			}
			_, err := conn.DeleteFunctionConcurrency(deleteConcurrencyParams)
			if err != nil {
				return fmt.Errorf("Error setting concurrency for Lambda %s: %s", d.Id(), err)
			}
		}
	}

	d.Partial(false)

	return resourceAwsLambdaFunctionRead(d, meta)
}

// loadFileContent returns contents of a file in a given path
func loadFileContent(v string) ([]byte, error) {
	filename, err := homedir.Expand(v)
	if err != nil {
		return nil, err
	}
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

func readEnvironmentVariables(ev map[string]interface{}) map[string]string {
	variables := make(map[string]string)
	for k, v := range ev {
		variables[k] = v.(string)
	}

	return variables
}
