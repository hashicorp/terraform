package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLambdaFunction_basic(t *testing.T) {
	var conf lambda.GetFunctionOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLambdaConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", &conf),
					testAccCheckAWSLambdaAttributes(&conf),
				),
			},
		},
	})
}

func testAccCheckLambdaFunctionDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).lambdaconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lambda_function" {
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

func testAccCheckAwsLambdaFunctionExists(n string, function *lambda.GetFunctionOutput) resource.TestCheckFunc {
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

func testAccCheckAWSLambdaAttributes(function *lambda.GetFunctionOutput) resource.TestCheckFunc {
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

const testAccAWSLambdaConfig = `
resource "aws_iam_role_policy" "iam_policy_for_lambda" {
    name = "iam_policy_for_lambda"
    role = "${aws_iam_role.iam_for_lambda.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "arn:aws:logs:*:*:*"
        },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateNetworkInterface"
      ],
      "Resource": [
        "*"
      ]
    }
  ]
}
EOF
}

resource "aws_iam_role" "iam_for_lambda" {
    name = "iam_for_lambda"
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

resource "aws_vpc" "vpc_for_lambda" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "subnet_for_lambda" {
    vpc_id = "${aws_vpc.vpc_for_lambda.id}"
    cidr_block = "10.0.1.0/24"

    tags {
        Name = "lambda"
    }
}

resource "aws_security_group" "sg_for_lambda" {
  name = "sg_for_lambda"
  description = "Allow all inbound traffic for lambda test"
  vpc_id = "${aws_vpc.vpc_for_lambda.id}"

  ingress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "example_lambda_name"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"

    vpc_config = {
        subnet_ids = ["${aws_subnet.subnet_for_lambda.id}"]
        security_group_ids = ["${aws_security_group.sg_for_lambda.id}"]
    }
}
`
