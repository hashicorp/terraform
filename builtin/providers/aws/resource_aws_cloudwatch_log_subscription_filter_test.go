package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudwatchLogSubscriptionFilter_basic(t *testing.T) {
	var conf lambda.GetFunctionOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudwatchLogSubscriptionFilterDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudwatchLogSubscriptionFilterConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsCloudwatchLogSubscriptionFilterExists("aws_cloudwatch_log_subscription_filter.test_lambdafunction_logfilter", &conf),
					testAccCheckAWSCloudwatchLogSubscriptionFilterAttributes(&conf),
				),
			},
		},
	})
}

func testAccCheckCloudwatchLogSubscriptionFilterDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).lambdaconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_log_subscription_filter" {
			continue
		}

		_, err := conn.GetFunction(&lambda.GetFunctionInput{
			FunctionName: aws.String(rs.Primary.ID),
		})

		if err == nil {
			return fmt.Errorf("Lambda Function still exists")
		}

	}

	return nil

}

func testAccCheckAwsCloudwatchLogSubscriptionFilterExists(n string, function *lambda.GetFunctionOutput) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Lambda function not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Lambda function ID not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lambdaconn

		params := &lambda.GetFunctionInput{
			FunctionName: aws.String("example_lambda_name"),
		}

		getFunction, err := conn.GetFunction(params)
		if err != nil {
			return err
		}

		*function = *getFunction

		return nil
	}
}

func testAccCheckAWSCloudwatchLogSubscriptionFilterAttributes(function *lambda.GetFunctionOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		const expectedName = "example_lambda_name"
		if *c.FunctionName != expectedName {
			return fmt.Errorf("Expected function name %s, got %s", expectedName, *c.FunctionName)
		}

		if *c.FunctionArn == "" {
			return fmt.Errorf("Could not read Lambda Function's ARN")
		}

		return nil
	}
}

const testAccAWSCloudwatchLogSubscriptionFilterConfig = `
resource "aws_cloudwatch_log_subscription_filter" "test_lambdafunction_logfilter" {
    name = "test_lambdafunction_logfilter"
    log_group_name = "example_lambda_name"
    filter_pattern = "logtype test"
    destination_arn = "${aws_lambda_function.test_lambdafunction.arn}"
}
resource "aws_lambda_function" "test_lambdafunction" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "example_lambda_name"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.handler"
}
resource "aws_cloudwatch_log_group" "logs" {
  name  = "example_lambda_name"
  retention_in_days = 1
}
resource "aws_lambda_permission" "allow_cloudwatch_logs" {
    statement_id = "AllowExecutionFromCloudWatchLogs"
    action = "lambda:*"
    function_name = "${aws_lambda_function.test_lambdafunction.arn}"
    principal = "logs.eu-west-1.amazonaws.com"
}
resource "aws_iam_role" "iam_for_lambda" {
    name = "test_lambdafuntion_iam_role"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
resource "aws_iam_role_policy" "test_lambdafunction_iam_policy" {
    name = "test_lambdafunction_iam_policy"
    role = "${aws_iam_role.iam_for_lambda.id}"
    policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Stmt1441111030000",
            "Effect": "Allow",
            "Action": [
                "dynamodb:*"
            ],
            "Resource": [
                "*"
            ]
        }
    ]
}
EOF
}
`
