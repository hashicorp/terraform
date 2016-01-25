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

var LambdaFunctionRegexp = `^(arn:aws:lambda:)?([a-z]{2}-[a-z]+-\d{1}:)?(\d{12}:)?(function:)?([a-zA-Z0-9-_]+)(:(\$LATEST|[a-zA-Z0-9-_]+))?$`

func resourceAwsLambdaPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLambdaPermissionCreate,
		Read:   resourceAwsLambdaPermissionRead,
		Delete: resourceAwsLambdaPermissionDelete,

		Schema: map[string]*schema.Schema{
			"action": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateLambdaPermissionAction,
			},
			"function_name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateLambdaFunctionName,
			},
			"principal": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"qualifier": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateLambdaQualifier,
			},
			"source_account": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"source_arn": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateArn,
			},
			"statement_id": &schema.Schema{
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

	input := lambda.AddPermissionInput{
		Action:       aws.String(d.Get("action").(string)),
		FunctionName: aws.String(d.Get("function_name").(string)),
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
	err := resource.Retry(1*time.Minute, func() error {
		var err error
		out, err = conn.AddPermission(&input)

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				// IAM is eventually consistent :/
				if awsErr.Code() == "ResourceConflictException" {
					return fmt.Errorf("[WARN] Error creating ELB Listener with SSL Cert, retrying: %s", err)
				}
			}
			return resource.RetryError{Err: err}
		}
		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Created new Lambda permission: %s", *out.Statement)

	d.SetId(d.Get("statement_id").(string))

	return resourceAwsLambdaPermissionRead(d, meta)
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
	out, err := conn.GetPolicy(&input)
	if err != nil {
		return fmt.Errorf("Error reading Lambda policy: %s", err)
	}

	d.Set("full_policy", *out.Policy)
	policyInBytes := []byte(*out.Policy)
	policy := LambdaPolicy{}
	err = json.Unmarshal(policyInBytes, &policy)
	if err != nil {
		return fmt.Errorf("Error unmarshalling Lambda policy: %s", err)
	}

	statement, err := findLambdaPolicyStatementById(&policy, d.Id())
	if err != nil {
		return err
	}

	qualifier, err := getQualifierFromLambdaAliasOrVersionArn(statement.Resource)
	if err == nil {
		d.Set("qualifier", qualifier)
	}

	// Save Lambda function name in the same format
	if strings.HasPrefix(d.Get("function_name").(string), "arn:aws:lambda:") {
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
	d.Set("principal", statement.Principal["Service"])

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

	input := lambda.RemovePermissionInput{
		FunctionName: aws.String(d.Get("function_name").(string)),
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

	log.Printf("[DEBUG] Lambda permission with ID %q removed", d.Id())
	d.SetId("")

	return nil
}

func findLambdaPolicyStatementById(policy *LambdaPolicy, id string) (
	*LambdaPolicyStatement, error) {

	log.Printf("[DEBUG] Received %d statements in Lambda policy", len(policy.Statement))
	for _, statement := range policy.Statement {
		if statement.Sid == id {
			return &statement, nil
		}
	}

	return nil, fmt.Errorf("Failed to find statement %q in Lambda policy", id)
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
