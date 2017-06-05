package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSNSTopicSubscription_basic(t *testing.T) {
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSTopicSubscriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicSubscriptionConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test_topic"),
					testAccCheckAWSSNSTopicSubscriptionExists("aws_sns_topic_subscription.test_subscription"),
				),
			},
		},
	})
}

func TestAccAWSSNSTopicSubscription_autoConfirmingEndpoint(t *testing.T) {
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSTopicSubscriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicSubscriptionConfig_autoConfirmingEndpoint(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test_topic"),
					testAccCheckAWSSNSTopicSubscriptionExists("aws_sns_topic_subscription.test_subscription"),
				),
			},
		},
	})
}

func testAccCheckAWSSNSTopicSubscriptionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).snsconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_sns_topic" {
			continue
		}

		// Try to find key pair
		req := &sns.GetSubscriptionAttributesInput{
			SubscriptionArn: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetSubscriptionAttributes(req)

		if err == nil {
			return fmt.Errorf("Subscription still exists, can't continue.")
		}

		// Verify the error is an API error, not something else
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSNSTopicSubscriptionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNS subscription with that ARN exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).snsconn

		params := &sns.GetSubscriptionAttributesInput{
			SubscriptionArn: aws.String(rs.Primary.ID),
		}
		_, err := conn.GetSubscriptionAttributes(params)

		if err != nil {
			return err
		}

		return nil
	}
}

func testAccAWSSNSTopicSubscriptionConfig(i int) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "test_topic" {
    name = "terraform-test-topic-%d"
}

resource "aws_sqs_queue" "test_queue" {
	name = "terraform-subscription-test-queue-%d"
}

resource "aws_sns_topic_subscription" "test_subscription" {
    topic_arn = "${aws_sns_topic.test_topic.arn}"
    protocol = "sqs"
    endpoint = "${aws_sqs_queue.test_queue.arn}"
}
`, i, i)
}

func testAccAWSSNSTopicSubscriptionConfig_autoConfirmingEndpoint(i int) string {
	return fmt.Sprintf(`
resource "aws_sns_topic" "test_topic" {
  name = "tf-acc-test-sns-%d"
}

resource "aws_api_gateway_rest_api" "test" {
  name        = "tf-acc-test-sns-%d"
  description = "Terraform Acceptance test for SNS subscription"
}

resource "aws_api_gateway_method" "test" {
  rest_api_id   = "${aws_api_gateway_rest_api.test.id}"
  resource_id   = "${aws_api_gateway_rest_api.test.root_resource_id}"
  http_method   = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_method_response" "test" {
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  status_code = "200"

  response_parameters {
    "method.response.header.Access-Control-Allow-Origin" = true
  }
}

resource "aws_api_gateway_integration" "test" {
  rest_api_id             = "${aws_api_gateway_rest_api.test.id}"
  resource_id             = "${aws_api_gateway_rest_api.test.root_resource_id}"
  http_method             = "${aws_api_gateway_method.test.http_method}"
  integration_http_method = "POST"
  type                    = "AWS"
  uri                     = "${aws_lambda_function.lambda.invoke_arn}"
}

resource "aws_api_gateway_integration_response" "test" {
  depends_on  = ["aws_api_gateway_integration.test"]
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  resource_id = "${aws_api_gateway_rest_api.test.root_resource_id}"
  http_method = "${aws_api_gateway_method.test.http_method}"
  status_code = "${aws_api_gateway_method_response.test.status_code}"

  response_parameters {
    "method.response.header.Access-Control-Allow-Origin" = "'*'"
  }
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "tf-acc-test-sns-%d"

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

resource "aws_iam_role_policy" "policy" {
  name = "tf-acc-test-sns-%d"
  role = "${aws_iam_role.iam_for_lambda.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "logs:*"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_lambda_permission" "apigw_lambda" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.lambda.arn}"
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_deployment.test.execution_arn}/*"
}

resource "aws_lambda_function" "lambda" {
  filename         = "test-fixtures/lambda_confirm_sns.zip"
  function_name    = "tf-acc-test-sns-%d"
  role             = "${aws_iam_role.iam_for_lambda.arn}"
  handler          = "main.confirm_subscription"
  source_code_hash = "${base64sha256(file("test-fixtures/lambda_confirm_sns.zip"))}"
  runtime          = "python3.6"
}

resource "aws_api_gateway_deployment" "test" {
  depends_on  = ["aws_api_gateway_integration_response.test"]
  rest_api_id = "${aws_api_gateway_rest_api.test.id}"
  stage_name  = "acctest"
}

resource "aws_sns_topic_subscription" "test_subscription" {
  depends_on             = ["aws_lambda_permission.apigw_lambda"]
  topic_arn              = "${aws_sns_topic.test_topic.arn}"
  protocol               = "https"
  endpoint               = "${aws_api_gateway_deployment.test.invoke_url}"
  endpoint_auto_confirms = true
}
`, i, i, i, i, i)
}
