package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudFormation_basic(t *testing.T) {
	var stack cloudformation.Stack

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudFormationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.network", &stack),
				),
			},
		},
	})
}

func TestAccAWSCloudFormation_defaultParams(t *testing.T) {
	var stack cloudformation.Stack

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudFormationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig_defaultParams,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.asg-demo", &stack),
				),
			},
		},
	})
}

func TestAccAWSCloudFormation_allAttributes(t *testing.T) {
	var stack cloudformation.Stack

	expectedPolicyBody := "{\"Statement\":[{\"Action\":\"Update:*\",\"Effect\":\"Deny\",\"Principal\":\"*\",\"Resource\":\"LogicalResourceId/StaticVPC\"},{\"Action\":\"Update:*\",\"Effect\":\"Allow\",\"Principal\":\"*\",\"Resource\":\"*\"}]}"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudFormationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig_allAttributesWithBodies,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.full", &stack),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "name", "tf-full-stack"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "capabilities.#", "1"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "capabilities.1328347040", "CAPABILITY_IAM"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "disable_rollback", "false"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "notification_arns.#", "1"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "parameters.#", "1"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "parameters.VpcCIDR", "10.0.0.0/16"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "policy_body", expectedPolicyBody),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "tags.#", "2"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "tags.First", "Mickey"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "tags.Second", "Mouse"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "timeout_in_minutes", "10"),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig_allAttributesWithBodies_modified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.full", &stack),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "name", "tf-full-stack"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "capabilities.#", "1"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "capabilities.1328347040", "CAPABILITY_IAM"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "disable_rollback", "false"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "notification_arns.#", "1"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "parameters.#", "1"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "parameters.VpcCIDR", "10.0.0.0/16"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "policy_body", expectedPolicyBody),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "tags.#", "2"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "tags.First", "Mickey"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "tags.Second", "Mouse"),
					resource.TestCheckResourceAttr("aws_cloudformation_stack.full", "timeout_in_minutes", "10"),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/4332
func TestAccAWSCloudFormation_withParams(t *testing.T) {
	var stack cloudformation.Stack

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudFormationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig_withParams,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.with_params", &stack),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig_withParams_modified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.with_params", &stack),
				),
			},
		},
	})
}

// Regression for https://github.com/hashicorp/terraform/issues/4534
func TestAccAWSCloudFormation_withUrl_withParams(t *testing.T) {
	var stack cloudformation.Stack

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudFormationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig_templateUrl_withParams,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.with-url-and-params", &stack),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudFormationConfig_templateUrl_withParams_modified,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFormationStackExists("aws_cloudformation_stack.with-url-and-params", &stack),
				),
			},
		},
	})
}

func testAccCheckCloudFormationStackExists(n string, stack *cloudformation.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cfconn
		params := &cloudformation.DescribeStacksInput{
			StackName: aws.String(rs.Primary.ID),
		}
		resp, err := conn.DescribeStacks(params)
		if err != nil {
			return err
		}
		if len(resp.Stacks) == 0 {
			return fmt.Errorf("CloudFormation stack not found")
		}

		return nil
	}
}

func testAccCheckAWSCloudFormationDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cfconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudformation_stack" {
			continue
		}

		params := cloudformation.DescribeStacksInput{
			StackName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeStacks(&params)

		if err != nil {
			return err
		}

		for _, s := range resp.Stacks {
			if *s.StackId == rs.Primary.ID && *s.StackStatus != "DELETE_COMPLETE" {
				return fmt.Errorf("CloudFormation stack still exists: %q", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccAWSCloudFormationConfig = `
resource "aws_cloudformation_stack" "network" {
  name = "tf-networking-stack"
  template_body = <<STACK
{
  "Resources" : {
    "MyVPC": {
      "Type" : "AWS::EC2::VPC",
      "Properties" : {
        "CidrBlock" : "10.0.0.0/16",
        "Tags" : [
          {"Key": "Name", "Value": "Primary_CF_VPC"}
        ]
      }
    }
  },
  "Outputs" : {
    "DefaultSgId" : {
      "Description": "The ID of default security group",
      "Value" : { "Fn::GetAtt" : [ "MyVPC", "DefaultSecurityGroup" ]}
    },
    "VpcID" : {
      "Description": "The VPC ID",
      "Value" : { "Ref" : "MyVPC" }
    }
  }
}
STACK
}`

var testAccAWSCloudFormationConfig_defaultParams = `
resource "aws_cloudformation_stack" "asg-demo" {
  name = "tf-asg-demo-stack"
  template_body = <<BODY
{
    "Parameters": {
        "TopicName": {
            "Type": "String"
        },
        "VPCCIDR": {
            "Type": "String",
            "Default": "10.10.0.0/16"
        }
    },
    "Resources": {
        "NotificationTopic": {
            "Type": "AWS::SNS::Topic",
            "Properties": {
                "TopicName": {
                    "Ref": "TopicName"
                }
            }
        },
        "MyVPC": {
            "Type": "AWS::EC2::VPC",
            "Properties": {
                "CidrBlock": {
                    "Ref": "VPCCIDR"
                },
                "Tags": [
                    {
                        "Key": "Name",
                        "Value": "Primary_CF_VPC"
                    }
                ]
            }
        }
    },
    "Outputs": {
        "VPCCIDR": {
            "Value": {
                "Ref": "VPCCIDR"
            }
        }
    }
}
BODY

  parameters {
    TopicName = "ExampleTopic"
  }
}
`

var testAccAWSCloudFormationConfig_allAttributesWithBodies_tpl = `
resource "aws_cloudformation_stack" "full" {
  name = "tf-full-stack"
  template_body = <<STACK
{
  "Parameters" : {
    "VpcCIDR" : {
      "Description" : "CIDR to be used for the VPC",
      "Type" : "String"
    }
  },
  "Resources" : {
    "MyVPC": {
      "Type" : "AWS::EC2::VPC",
      "Properties" : {
        "CidrBlock" : {"Ref": "VpcCIDR"},
        "Tags" : [
          {"Key": "Name", "Value": "%s"}
        ]
      }
    },
    "StaticVPC": {
      "Type" : "AWS::EC2::VPC",
      "Properties" : {
        "CidrBlock" : {"Ref": "VpcCIDR"},
        "Tags" : [
          {"Key": "Name", "Value": "Static_CF_VPC"}
        ]
      }
    },
    "InstanceRole" : {
      "Type" : "AWS::IAM::Role",
      "Properties" : {
        "AssumeRolePolicyDocument": {
          "Version": "2012-10-17",
          "Statement": [ {
            "Effect": "Allow",
            "Principal": { "Service": "ec2.amazonaws.com" },
            "Action": "sts:AssumeRole"
          } ]
        },
        "Path" : "/",
        "Policies" : [ {
          "PolicyName": "terraformtest",
          "PolicyDocument": {
            "Version": "2012-10-17",
            "Statement": [ {
              "Effect": "Allow",
              "Action": [ "ec2:DescribeSnapshots" ],
              "Resource": [ "*" ]
            } ]
          }
        } ]
      }
    }
  }
}
STACK
  parameters {
    VpcCIDR = "10.0.0.0/16"
  }

  policy_body = <<POLICY
%s
POLICY
  capabilities = ["CAPABILITY_IAM"]
  notification_arns = ["${aws_sns_topic.cf-updates.arn}"]
  on_failure = "DELETE"
  timeout_in_minutes = 10
  tags {
    First = "Mickey"
    Second = "Mouse"
  }
}

resource "aws_sns_topic" "cf-updates" {
  name = "tf-cf-notifications"
}
`

var policyBody = `
{
  "Statement" : [
    {
      "Effect" : "Deny",
      "Action" : "Update:*",
      "Principal": "*",
      "Resource" : "LogicalResourceId/StaticVPC"
    },
    {
      "Effect" : "Allow",
      "Action" : "Update:*",
      "Principal": "*",
      "Resource" : "*"
    }
  ]
}
`

var testAccAWSCloudFormationConfig_allAttributesWithBodies = fmt.Sprintf(
	testAccAWSCloudFormationConfig_allAttributesWithBodies_tpl,
	"Primary_CF_VPC",
	policyBody)
var testAccAWSCloudFormationConfig_allAttributesWithBodies_modified = fmt.Sprintf(
	testAccAWSCloudFormationConfig_allAttributesWithBodies_tpl,
	"Primary_CloudFormation_VPC",
	policyBody)

var tpl_testAccAWSCloudFormationConfig_withParams = `
resource "aws_cloudformation_stack" "with_params" {
  name = "tf-stack-with-params"
  parameters {
    VpcCIDR = "%s"
  }
  template_body = <<STACK
{
  "Parameters" : {
    "VpcCIDR" : {
      "Description" : "CIDR to be used for the VPC",
      "Type" : "String"
    }
  },
  "Resources" : {
    "MyVPC": {
      "Type" : "AWS::EC2::VPC",
      "Properties" : {
        "CidrBlock" : {"Ref": "VpcCIDR"},
        "Tags" : [
          {"Key": "Name", "Value": "Primary_CF_VPC"}
        ]
      }
    }
  }
}
STACK

  on_failure = "DELETE"
  timeout_in_minutes = 1
}
`

var testAccAWSCloudFormationConfig_withParams = fmt.Sprintf(
	tpl_testAccAWSCloudFormationConfig_withParams,
	"10.0.0.0/16")
var testAccAWSCloudFormationConfig_withParams_modified = fmt.Sprintf(
	tpl_testAccAWSCloudFormationConfig_withParams,
	"12.0.0.0/16")

var tpl_testAccAWSCloudFormationConfig_templateUrl_withParams = `
resource "aws_s3_bucket" "b" {
  bucket = "%s"
  acl = "public-read"
  policy = <<POLICY
{
  "Version":"2008-10-17",
  "Statement": [
    {
      "Sid":"AllowPublicRead",
      "Effect":"Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::%s/*"
    }
  ]
}
POLICY

  website {
      index_document = "index.html"
      error_document = "error.html"
  }
}

resource "aws_s3_bucket_object" "object" {
  bucket = "${aws_s3_bucket.b.id}"
  key = "tf-cf-stack.json"
  source = "test-fixtures/cloudformation-template.json"
}

resource "aws_cloudformation_stack" "with-url-and-params" {
  name = "tf-stack-template-url-with-params"
  parameters {
    VpcCIDR = "%s"
  }
  template_url = "https://${aws_s3_bucket.b.id}.s3-us-west-2.amazonaws.com/${aws_s3_bucket_object.object.key}"
  on_failure = "DELETE"
  timeout_in_minutes = 1
}
`

var cfRandInt = rand.New(rand.NewSource(time.Now().UnixNano())).Int()
var cfBucketName = "tf-stack-with-url-and-params-" + fmt.Sprintf("%d", cfRandInt)

var testAccAWSCloudFormationConfig_templateUrl_withParams = fmt.Sprintf(
	tpl_testAccAWSCloudFormationConfig_templateUrl_withParams,
	cfBucketName, cfBucketName, "11.0.0.0/16")
var testAccAWSCloudFormationConfig_templateUrl_withParams_modified = fmt.Sprintf(
	tpl_testAccAWSCloudFormationConfig_templateUrl_withParams,
	cfBucketName, cfBucketName, "13.0.0.0/16")
