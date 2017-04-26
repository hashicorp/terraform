package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudwatchLogSubscriptionFilter_basic(t *testing.T) {
	var conf lambda.GetFunctionOutput

	rstring := acctest.RandString(5)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudwatchLogSubscriptionFilterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudwatchLogSubscriptionFilterConfig(rstring),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsCloudwatchLogSubscriptionFilterExists("aws_cloudwatch_log_subscription_filter.test_lambdafunction_logfilter", &conf, rstring),
					testAccCheckAWSCloudwatchLogSubscriptionFilterAttributes(&conf, rstring),
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

func testAccCheckAwsCloudwatchLogSubscriptionFilterExists(n string, function *lambda.GetFunctionOutput, rstring string) resource.TestCheckFunc {
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
			FunctionName: aws.String("example_lambda_name_" + rstring),
		}

		getFunction, err := conn.GetFunction(params)
		if err != nil {
			return err
		}

		*function = *getFunction

		return nil
	}
}

func testAccCheckAWSCloudwatchLogSubscriptionFilterAttributes(function *lambda.GetFunctionOutput, rstring string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		expectedName := fmt.Sprintf("example_lambda_name_%s", rstring)
		if *c.FunctionName != expectedName {
			return fmt.Errorf("Expected function name %s, got %s", expectedName, *c.FunctionName)
		}

		if *c.FunctionArn == "" {
			return fmt.Errorf("Could not read Lambda Function's ARN")
		}

		return nil
	}
}

func testAccAWSCloudwatchLogSubscriptionFilterConfig(rstring string) string {
	return fmt.Sprintf(`
resource "aws_cloudwatch_log_subscription_filter" "test_lambdafunction_logfilter" {
  name            = "test_lambdafunction_logfilter_%s"
  log_group_name  = "${aws_cloudwatch_log_group.logs.name}"
  filter_pattern  = "logtype test"
  destination_arn = "${aws_lambda_function.test_lambdafunction.arn}"
}

resource "aws_lambda_function" "test_lambdafunction" {
  filename      = "test-fixtures/lambdatest.zip"
  function_name = "example_lambda_name_%s"
  role          = "${aws_iam_role.iam_for_lambda.arn}"
  runtime       = "nodejs4.3"
  handler       = "exports.handler"
}

resource "aws_cloudwatch_log_group" "logs" {
  name              = "example_lambda_name_%s"
  retention_in_days = 1
}

resource "aws_lambda_permission" "allow_cloudwatch_logs" {
  statement_id  = "AllowExecutionFromCloudWatchLogs"
  action        = "lambda:*"
  function_name = "${aws_lambda_function.test_lambdafunction.arn}"
  principal     = "logs.us-west-2.amazonaws.com"
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "test_lambdafuntion_iam_role_%s"

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
  name = "test_lambdafunction_iam_policy_%s"
  role = "${aws_iam_role.iam_for_lambda.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Stmt1441111030000",
      "Effect": "Allow",
      "Action": [
        "lambda:*"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}
`, rstring, rstring, rstring, rstring, rstring)
}
