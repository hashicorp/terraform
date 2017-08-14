package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

var LambdaFunctionRegexp = `^(arn:[\w-]+:lambda:)?([a-z]{2}-(?:[a-z]+-){1,2}\d{1}:)?(\d{12}:)?(function:)?([a-zA-Z0-9-_]+)(:(\$LATEST|[a-zA-Z0-9-_]+))?$`

func resourceAwsLambdaPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaPermissionCreate,
		Read:   resourceAwsLambdaPermissionRead,
		Delete: resourceAwsLambdaPermissionDelete,

		Schema: map[string]*schema.Schema{
			"action": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateLambdaPermissionAction,
			},
			"function_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateLambdaFunctionName,
			},
			"principal": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"qualifier": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateLambdaQualifier,
			},
			"source_account": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"source_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"statement_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validatePolicyStatementId,
			},
		},
	}
}

func resourceAwsLambdaPermissionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	functionName := d.Get("function_name").(string)

	// There is a bug in the API (reported and acknowledged by AWS)
	// which causes some permissions to be ignored when API calls are sent in parallel
	// We work around this bug via mutex
	awsMutexKV.Lock(functionName)
	defer awsMutexKV.Unlock(functionName)

	input := lambda.AddPermissionInput{
		Action:       aws.String(d.Get("action").(string)),
		FunctionName: aws.String(functionName),
		Principal:    aws.String(d.Get("principal").(string)),
		StatementId:  aws.String(d.Get("statement_id").(string)),
	}

	if v, ok := d.GetOk("qualifier"); ok {
		input.Qualifier = aws.String(v.(string))
	}
	if v, ok := d.GetOk("source_account"); ok {
		input.SourceAccount = aws.String(v.(string))
	}
	if v, ok := d.GetOk("source_arn"); ok {
		input.SourceArn = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Adding new Lambda permission: %s", input)
	var out *lambda.AddPermissionOutput
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error
		out, err = conn.AddPermission(&input)

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				// IAM is eventually consistent :/
				if awsErr.Code() == "ResourceConflictException" {
					return resource.RetryableError(
						fmt.Errorf("[WARN] Error adding new Lambda Permission for %s, retrying: %s",
							*input.FunctionName, err))
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if out != nil && out.Statement != nil {
		log.Printf("[DEBUG] Created new Lambda permission: %s", *out.Statement)
	} else {
		log.Printf("[DEBUG] Created new Lambda permission, but no Statement was included")
	}

	d.SetId(d.Get("statement_id").(string))

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		// IAM is eventually cosistent :/
		err := resourceAwsLambdaPermissionRead(d, meta)
		if err != nil {
			if strings.HasPrefix(err.Error(), "Error reading Lambda policy: ResourceNotFoundException") {
				return resource.RetryableError(
					fmt.Errorf("[WARN] Error reading newly created Lambda Permission for %s, retrying: %s",
						*input.FunctionName, err))
			}
			if strings.HasPrefix(err.Error(), "Failed to find statement \""+d.Id()) {
				return resource.RetryableError(
					fmt.Errorf("[WARN] Error reading newly created Lambda Permission statement for %s, retrying: %s",
						*input.FunctionName, err))
			}

			log.Printf("[ERROR] An actual error occurred when expecting Lambda policy to be there: %s", err)
			return resource.NonRetryableError(err)
		}
		return nil
	})

	return err
}

func resourceAwsLambdaPermissionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	input := lambda.GetPolicyInput{
		FunctionName: aws.String(d.Get("function_name").(string)),
	}
	if v, ok := d.GetOk("qualifier"); ok {
		input.Qualifier = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Looking for Lambda permission: %s", input)
	var out *lambda.GetPolicyOutput
	var statement *LambdaPolicyStatement
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		// IAM is eventually cosistent :/
		var err error
		out, err = conn.GetPolicy(&input)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceNotFoundException" {
					return resource.RetryableError(err)
				}
			}
			return resource.NonRetryableError(err)
		}

		policyInBytes := []byte(*out.Policy)
		policy := LambdaPolicy{}
		err = json.Unmarshal(policyInBytes, &policy)
		if err != nil {
			return resource.NonRetryableError(err)
		}

		statement, err = findLambdaPolicyStatementById(&policy, d.Id())
		return resource.RetryableError(err)
	})

	if err != nil {
		// Missing whole policy or Lambda function (API error)
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ResourceNotFoundException" {
				log.Printf("[WARN] No Lambda Permission Policy found: %v", input)
				d.SetId("")
				return nil
			}
		}

		// Missing permission inside valid policy
		if nfErr, ok := err.(*resource.NotFoundError); ok {
			log.Printf("[WARN] %s", nfErr)
			d.SetId("")
			return nil
		}

		return err
	}

	qualifier, err := getQualifierFromLambdaAliasOrVersionArn(statement.Resource)
	if err != nil {
		log.Printf("[ERR] Error getting Lambda Qualifier: %s", err)
	}
	d.Set("qualifier", qualifier)

	// Save Lambda function name in the same format
	if strings.HasPrefix(d.Get("function_name").(string), "arn:"+meta.(*AWSClient).partition+":lambda:") {
		// Strip qualifier off
		trimmedArn := strings.TrimSuffix(statement.Resource, ":"+qualifier)
		d.Set("function_name", trimmedArn)
	} else {
		functionName, err := getFunctionNameFromLambdaArn(statement.Resource)
		if err != nil {
			return err
		}
		d.Set("function_name", functionName)
	}

	d.Set("action", statement.Action)
	// Check if the pricipal is a cross-account IAM role
	if _, ok := statement.Principal["AWS"]; ok {
		d.Set("principal", statement.Principal["AWS"])
	} else {
		d.Set("principal", statement.Principal["Service"])
	}

	if stringEquals, ok := statement.Condition["StringEquals"]; ok {
		d.Set("source_account", stringEquals["AWS:SourceAccount"])
	}

	if arnLike, ok := statement.Condition["ArnLike"]; ok {
		d.Set("source_arn", arnLike["AWS:SourceArn"])
	}

	return nil
}

func resourceAwsLambdaPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).lambdaconn

	functionName := d.Get("function_name").(string)

	// There is a bug in the API (reported and acknowledged by AWS)
	// which causes some permissions to be ignored when API calls are sent in parallel
	// We work around this bug via mutex
	awsMutexKV.Lock(functionName)
	defer awsMutexKV.Unlock(functionName)

	input := lambda.RemovePermissionInput{
		FunctionName: aws.String(functionName),
		StatementId:  aws.String(d.Id()),
	}

	if v, ok := d.GetOk("qualifier"); ok {
		input.Qualifier = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Removing Lambda permission: %s", input)
	_, err := conn.RemovePermission(&input)
	if err != nil {
		return err
	}

	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Checking if Lambda permission %q is deleted", d.Id())

		params := &lambda.GetPolicyInput{
			FunctionName: aws.String(d.Get("function_name").(string)),
		}
		if v, ok := d.GetOk("qualifier"); ok {
			params.Qualifier = aws.String(v.(string))
		}

		log.Printf("[DEBUG] Looking for Lambda permission: %s", *params)
		resp, err := conn.GetPolicy(params)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceNotFoundException" {
					return nil
				}
			}
			return resource.NonRetryableError(err)
		}

		if resp.Policy == nil {
			return nil
		}

		policyInBytes := []byte(*resp.Policy)
		policy := LambdaPolicy{}
		err = json.Unmarshal(policyInBytes, &policy)
		if err != nil {
			return resource.RetryableError(
				fmt.Errorf("Error unmarshalling Lambda policy: %s", err))
		}

		_, err = findLambdaPolicyStatementById(&policy, d.Id())
		if err != nil {
			return nil
		}

		log.Printf("[DEBUG] No error when checking if Lambda permission %s is deleted", d.Id())
		return nil
	})

	if err != nil {
		return fmt.Errorf("Failed removing Lambda permission: %s", err)
	}

	log.Printf("[DEBUG] Lambda permission with ID %q removed", d.Id())
	d.SetId("")

	return nil
}

func findLambdaPolicyStatementById(policy *LambdaPolicy, id string) (
	*LambdaPolicyStatement, error) {

	log.Printf("[DEBUG] Received %d statements in Lambda policy: %s", len(policy.Statement), policy.Statement)
	for _, statement := range policy.Statement {
		if statement.Sid == id {
			return &statement, nil
		}
	}

	return nil, &resource.NotFoundError{
		LastRequest:  id,
		LastResponse: policy,
		Message:      fmt.Sprintf("Failed to find statement %q in Lambda policy:\n%s", id, policy.Statement),
	}
}

func getQualifierFromLambdaAliasOrVersionArn(arn string) (string, error) {
	matches := regexp.MustCompile(LambdaFunctionRegexp).FindStringSubmatch(arn)
	if len(matches) < 8 || matches[7] == "" {
		return "", fmt.Errorf("Invalid ARN or otherwise unable to get qualifier from ARN (%q)",
			arn)
	}

	return matches[7], nil
}

func getFunctionNameFromLambdaArn(arn string) (string, error) {
	matches := regexp.MustCompile(LambdaFunctionRegexp).FindStringSubmatch(arn)
	if len(matches) < 6 || matches[5] == "" {
		return "", fmt.Errorf("Invalid ARN or otherwise unable to get qualifier from ARN (%q)",
			arn)
	}
	return matches[5], nil
}

type LambdaPolicy struct {
	Version   string
	Statement []LambdaPolicyStatement
	Id        string
}

type LambdaPolicyStatement struct {
	Condition map[string]map[string]string
	Action    string
	Resource  string
	Effect    string
	Principal map[string]string
	Sid       string
}
