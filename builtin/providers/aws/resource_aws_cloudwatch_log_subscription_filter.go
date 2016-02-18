package aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsCloudwatchLogSubscriptionFilter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudwatchLogSubscriptionFilterCreate,
		Read:   resourceAwsCloudwatchLogSubscriptionFilterRead,
		Update: resourceAwsCloudwatchLogSubscriptionFilterUpdate,
		Delete: resourceAwsCloudwatchLogSubscriptionFilterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"filter_pattern": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"log_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsCloudwatchLogSubscriptionFilterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)

	log_group_name := d.Get("log_group_name").(string)

	destination_arn := d.Get("destination_arn").(string)
	if strings.HasPrefix(destination_arn, "arn:aws:lambda") {
		lambda_conn := meta.(*AWSClient).lambdaconn

		lambda_arn_sliced := strings.Split(destination_arn, ":")
		function_name := lambda_arn_sliced[len(lambda_arn_sliced)-1]
		statement_id := lambdaPermissionStatementId(log_group_name, destination_arn)

		if !permissionExists(function_name, statement_id, lambda_conn) {
			region := lambda_arn_sliced[3]
			accountid := lambda_arn_sliced[4]
			principal := fmt.Sprintf("logs.%s.amazonaws.com", region)
			source_arn := fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s:*", region, accountid, log_group_name)

			params := &lambda.AddPermissionInput{
				Action:        aws.String("lambda:InvokeFunction"),
				FunctionName:  aws.String(function_name),
				Principal:     aws.String(principal),
				StatementId:   aws.String(statement_id),
				SourceArn:     aws.String(source_arn),
				SourceAccount: aws.String(accountid),
			}

			log.Printf("[DEBUG] Attempting: to do add-access with params \"%#v\"", params)
			_, err := lambda_conn.AddPermission(params)
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					return fmt.Errorf("[WARN] Error doing add-access for LogGroup (%s) to lambda (%s), message: \"%s\", code: \"%s\"",
						log_group_name, destination_arn, awsErr.Message(), awsErr.Code())
				} else {
					return fmt.Errorf("Error creating Cloudwatch logs subscription filter %s: %#v", name, err)
				}
			}
		}
	}

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)

	log.Printf("[DEBUG] Creating SubscriptionFilter %#v", params)

	err := resource.Retry(10*time.Minute, func() error {
		out, err := conn.PutSubscriptionFilter(&params)

		if err == nil {
			d.SetId(cloudwatchLogsSubscriptionFilterId(d.Get("log_group_name").(string)))
			log.Printf("[DEBUG] Cloudwatch logs subscription %q created: %q", d.Id(), out)
			return resourceAwsCloudwatchLogSubscriptionFilterRead(d, meta)
		}

		awsErr, ok := err.(awserr.Error)
		if !ok {
			return resource.RetryError{Err: err}
		}

		if awsErr.Code() == "InvalidParameterException" {
			log.Printf("[DEBUG] Caught message: %q, code: %q: Retrying", awsErr.Message(), awsErr.Code())
			return err
		}

		return resource.RetryError{Err: err}
	})
	if err != nil {
		return err
	}
	return nil
}

func resourceAwsCloudwatchLogSubscriptionFilterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)

	log.Printf("[DEBUG] Update SubscriptionFilter %#v", params)
	_, err := conn.PutSubscriptionFilter(&params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error updating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
				d.Get("name").(string), d.Get("log_group_name").(string), awsErr.Message(), awsErr.Code())
		}
		return err
	}

	d.SetId(cloudwatchLogsSubscriptionFilterId(d.Get("log_group_name").(string)))
	return resourceAwsCloudwatchLogSubscriptionFilterRead(d, meta)
}

func getAwsCloudWatchLogsSubscriptionFilterInput(d *schema.ResourceData) cloudwatchlogs.PutSubscriptionFilterInput {
	name := d.Get("name").(string)
	destination_arn := d.Get("destination_arn").(string)
	filter_pattern := d.Get("filter_pattern").(string)
	log_group_name := d.Get("log_group_name").(string)

	params := cloudwatchlogs.PutSubscriptionFilterInput{
		FilterName:     aws.String(name),
		DestinationArn: aws.String(destination_arn),
		FilterPattern:  aws.String(filter_pattern),
		LogGroupName:   aws.String(log_group_name),
	}

	if _, ok := d.GetOk("role_arn"); ok {
		params.RoleArn = aws.String(d.Get("role_arn").(string))
	}

	return params
}

func resourceAwsCloudwatchLogSubscriptionFilterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log_group_name := d.Get("log_group_name").(string)
	name := d.Get("name").(string) // "name" is a required field in the schema

	req := &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName:     aws.String(log_group_name),
		FilterNamePrefix: aws.String(name),
	}

	resp, err := conn.DescribeSubscriptionFilters(req)
	if err != nil {
		return fmt.Errorf("Error reading SubscriptionFilters for log group %s with name prefix %s: %#v", log_group_name, d.Get("name").(string), err)
	}

	for _, subscriptionFilter := range resp.SubscriptionFilters {
		if *subscriptionFilter.LogGroupName == log_group_name {
			d.SetId(cloudwatchLogsSubscriptionFilterId(log_group_name))
			return nil // OK, matching subscription filter found
		}
	}

	return fmt.Errorf("Subscription filter for log group %s with name prefix %s not found!", log_group_name, d.Get("name").(string))
}

func resourceAwsCloudwatchLogSubscriptionFilterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log_group_name := d.Get("log_group_name").(string)
	name := d.Get("name").(string)
	destination_arn := d.Get("destination_arn").(string)

	if strings.HasPrefix(destination_arn, "arn:aws:lambda") {
		// access permissions should also be cleaned up
		lambda_conn := meta.(*AWSClient).lambdaconn

		lambda_arn_sliced := strings.Split(destination_arn, ":")
		function_name := lambda_arn_sliced[len(lambda_arn_sliced)-1]
		statement_id := lambdaPermissionStatementId(log_group_name, destination_arn)

		if permissionExists(function_name, statement_id, lambda_conn) {
			_, err := lambda_conn.RemovePermission(&lambda.RemovePermissionInput{
				FunctionName: aws.String(function_name),
				StatementId:  aws.String(statement_id),
			})
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					log.Printf("[WARN] Error removing the access permission SID (%s) for lambda function (%s), message: \"%s\", code: \"%s\"",
						statement_id, function_name, awsErr.Message(), awsErr.Code())
				} else {
					log.Printf("[WARN] Error removing the access permission from lambda function %s: %#v", function_name, err)
				}
			}
		}
	}

	params := &cloudwatchlogs.DeleteSubscriptionFilterInput{
		FilterName:   aws.String(name),           // Required
		LogGroupName: aws.String(log_group_name), // Required
	}
	_, err := conn.DeleteSubscriptionFilter(params)

	if err != nil {
		return fmt.Errorf(
			"Error deleting Subscription Filter from log group: %s with name filter name %s", log_group_name, name)
	}
	d.SetId("")
	return nil
}

func permissionExists(function_name string, statementid string, lambda_conn *lambda.Lambda) bool {

	resp, err := lambda_conn.GetPolicy(&lambda.GetPolicyInput{
		FunctionName: aws.String(function_name),
	})

	type PolicyDocument struct {
		Version   string
		Statement []struct {
			Resource, Effect, Sid string
		}
	}

	if err != nil {
		log.Printf("[DEBUG] GetPolicy returns \"%#v\" - maybe no access permissions exists?", err)
		return false
	} else {
		dec := json.NewDecoder(strings.NewReader(*resp.Policy))
		for {
			var m PolicyDocument
			if err := dec.Decode(&m); err == io.EOF {
				break
			} else if err != nil {
				log.Printf("[DEBUG] Decoding access policy failed \"%#v\"", resp.Policy)
				log.Fatal(err)
			}

			for _, statement := range m.Statement {
				if statement.Sid == statementid {
					return true
				}
			}
		}
		log.Printf("[DEBUG] Statement Id \"%s\" not found in policy for function \"%s\"", function_name, statementid)
		return false
	}

}

func lambdaPermissionStatementId(log_group_name string, lambda_arn string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s-", log_group_name))
	buf.WriteString(fmt.Sprintf("%s-", lambda_arn))

	return fmt.Sprintf("InvokePermissionsForCWL%d", hashcode.String(buf.String()))
}

func cloudwatchLogsSubscriptionFilterId(log_group_name string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s-", log_group_name)) // only one filter allowed per log_group_name at the moment

	return fmt.Sprintf("cwlsf-%d", hashcode.String(buf.String()))
}
