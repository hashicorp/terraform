package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/acctest"
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
				Config: testAccAWSLambdaConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", "example_lambda_name", &conf),
					testAccCheckAwsLambdaFunctionName(&conf, "example_lambda_name"),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":example_lambda_name"),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_VPC(t *testing.T) {
	var conf lambda.GetFunctionOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLambdaConfigWithVPC,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_test", "example_lambda_name", &conf),
					testAccCheckAwsLambdaFunctionName(&conf, "example_lambda_name"),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":example_lambda_name"),
					testAccCheckAWSLambdaFunctionVersion(&conf, "$LATEST"),
				),
			},
		},
	})
}

func TestAccAWSLambdaFunction_s3(t *testing.T) {
	var conf lambda.GetFunctionOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLambdaFunctionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSLambdaConfigS3,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsLambdaFunctionExists("aws_lambda_function.lambda_function_s3test", "example_lambda_name_s3", &conf),
					testAccCheckAwsLambdaFunctionName(&conf, "example_lambda_name_s3"),
					testAccCheckAwsLambdaFunctionArnHasSuffix(&conf, ":example_lambda_name_s3"),
					testAccCheckAWSLambdaFunctionVersion(&conf, "$LATEST"),
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

func testAccCheckAwsLambdaFunctionExists(res, funcName string, function *lambda.GetFunctionOutput) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[res]
		if !ok {
			return fmt.Errorf("Lambda function not found: %s", res)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Lambda function ID not set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lambdaconn

		params := &lambda.GetFunctionInput{
			FunctionName: aws.String(funcName),
		}

		getFunction, err := conn.GetFunction(params)
		if err != nil {
			return err
		}

		*function = *getFunction

		return nil
	}
}

func testAccCheckAwsLambdaFunctionName(function *lambda.GetFunctionOutput, expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if *c.FunctionName != expectedName {
			return fmt.Errorf("Expected function name %s, got %s", expectedName, *c.FunctionName)
		}

		return nil
	}
}

func testAccCheckAWSLambdaFunctionVersion(function *lambda.GetFunctionOutput, expectedVersion string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if *c.Version != expectedVersion {
			return fmt.Errorf("Expected version %s, got %s", expectedVersion, *c.Version)
		}
		return nil
	}
}

func testAccCheckAwsLambdaFunctionArnHasSuffix(function *lambda.GetFunctionOutput, arnSuffix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		c := function.Configuration
		if !strings.HasSuffix(*c.FunctionArn, arnSuffix) {
			return fmt.Errorf("Expected function ARN %s to have suffix %s", *c.FunctionArn, arnSuffix)
		}

		return nil
	}
}

const baseAccAWSLambdaConfig = `
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

`

const testAccAWSLambdaConfigBasic = baseAccAWSLambdaConfig + `
resource "aws_lambda_function" "lambda_function_test" {
    filename = "test-fixtures/lambdatest.zip"
    function_name = "example_lambda_name"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
}
`

const testAccAWSLambdaConfigWithVPC = baseAccAWSLambdaConfig + `
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

var testAccAWSLambdaConfigS3 = fmt.Sprintf(`
resource "aws_s3_bucket" "lambda_bucket" {
  bucket = "tf-test-bucket-%d"
}

resource "aws_s3_bucket_object" "lambda_code" {
  bucket = "${aws_s3_bucket.lambda_bucket.id}"
  key = "lambdatest.zip"
  source = "test-fixtures/lambdatest.zip"
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

resource "aws_lambda_function" "lambda_function_s3test" {
    s3_bucket = "${aws_s3_bucket.lambda_bucket.id}"
    s3_key = "${aws_s3_bucket_object.lambda_code.id}"
    function_name = "example_lambda_name_s3"
    role = "${aws_iam_role.iam_for_lambda.arn}"
    handler = "exports.example"
}
`, acctest.RandInt())
