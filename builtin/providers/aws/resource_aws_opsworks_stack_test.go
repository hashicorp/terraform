package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
)

//////////////////////////////////////////////////
//// Helper configs for the necessary IAM objects
//////////////////////////////////////////////////

var testAccAwsOpsworksStackIamConfig = `
resource "aws_iam_role" "opsworks_service" {
    name = "terraform_testacc_opsworks_service"
    assume_role_policy = <<EOT
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "opsworks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOT
}

resource "aws_iam_role_policy" "opsworks_service" {
    name = "terraform_testacc_opsworks_service"
    role = "${aws_iam_role.opsworks_service.id}"
    policy = <<EOT
{
  "Statement": [
    {
      "Action": [
        "ec2:*",
        "iam:PassRole",
        "cloudwatch:GetMetricStatistics",
        "elasticloadbalancing:*",
        "rds:*"
      ],
      "Effect": "Allow",
      "Resource": ["*"]
    }
  ]
}
EOT
}

resource "aws_iam_role" "opsworks_instance" {
    name = "terraform_testacc_opsworks_instance"
    assume_role_policy = <<EOT
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOT
}

resource "aws_iam_instance_profile" "opsworks_instance" {
    name = "terraform_testacc_opsworks_instance"
    roles = ["${aws_iam_role.opsworks_instance.name}"]
}

`

///////////////////////////////
//// Tests for the No-VPC case
///////////////////////////////

var testAccAwsOpsworksStackConfigNoVpcCreate = testAccAwsOpsworksStackIamConfig + `
resource "aws_opsworks_stack" "tf-acc" {
  name = "tf-opsworks-acc"
  region = "us-east-1"
  service_role_arn = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_availability_zone = "us-east-1c"
  default_os = "Amazon Linux 2014.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
}
`
var testAccAWSOpsworksStackConfigNoVpcUpdate = testAccAwsOpsworksStackIamConfig + `
resource "aws_opsworks_stack" "tf-acc" {
  name = "tf-opsworks-acc"
  region = "us-east-1"
  service_role_arn = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_availability_zone = "us-east-1c"
  default_os = "Amazon Linux 2014.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
  use_custom_cookbooks = true
  manage_berkshelf = true
  custom_cookbooks_source {
    type = "git"
    revision = "master"
    url = "https://github.com/aws/opsworks-example-cookbooks.git"
  }
}
`

func TestAccAWSOpsworksStackNoVpc(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksStackConfigNoVpcCreate,
				Check:  testAccAwsOpsworksStackCheckResourceAttrsCreate("us-east-1c"),
			},
			resource.TestStep{
				Config: testAccAWSOpsworksStackConfigNoVpcUpdate,
				Check:  testAccAwsOpsworksStackCheckResourceAttrsUpdate("us-east-1c"),
			},
		},
	})
}

////////////////////////////
//// Tests for the VPC case
////////////////////////////

var testAccAwsOpsworksStackConfigVpcCreate = testAccAwsOpsworksStackIamConfig + `
resource "aws_vpc" "tf-acc" {
  cidr_block = "10.3.5.0/24"
}
resource "aws_subnet" "tf-acc" {
  vpc_id = "${aws_vpc.tf-acc.id}"
  cidr_block = "${aws_vpc.tf-acc.cidr_block}"
  availability_zone = "us-west-2a"
}
resource "aws_opsworks_stack" "tf-acc" {
  name = "tf-opsworks-acc"
  region = "us-west-2"
  vpc_id = "${aws_vpc.tf-acc.id}"
  default_subnet_id = "${aws_subnet.tf-acc.id}"
  service_role_arn = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_os = "Amazon Linux 2014.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
}
`

var testAccAWSOpsworksStackConfigVpcUpdate = testAccAwsOpsworksStackIamConfig + `
resource "aws_vpc" "tf-acc" {
  cidr_block = "10.3.5.0/24"
}
resource "aws_subnet" "tf-acc" {
  vpc_id = "${aws_vpc.tf-acc.id}"
  cidr_block = "${aws_vpc.tf-acc.cidr_block}"
  availability_zone = "us-west-2a"
}
resource "aws_opsworks_stack" "tf-acc" {
  name = "tf-opsworks-acc"
  region = "us-west-2"
  vpc_id = "${aws_vpc.tf-acc.id}"
  default_subnet_id = "${aws_subnet.tf-acc.id}"
  service_role_arn = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_os = "Amazon Linux 2014.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
  use_custom_cookbooks = true
  manage_berkshelf = true
  custom_cookbooks_source {
    type = "git"
    revision = "master"
    url = "https://github.com/aws/opsworks-example-cookbooks.git"
  }
}
`

func TestAccAWSOpsworksStackVpc(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksStackConfigVpcCreate,
				Check:  testAccAwsOpsworksStackCheckResourceAttrsCreate("us-west-2a"),
			},
			resource.TestStep{
				Config: testAccAWSOpsworksStackConfigVpcUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsOpsworksStackCheckResourceAttrsUpdate("us-west-2a"),
					testAccAwsOpsworksCheckVpc,
				),
			},
		},
	})
}

////////////////////////////
//// Checkers and Utilities
////////////////////////////

func testAccAwsOpsworksStackCheckResourceAttrsCreate(zone string) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"name",
			"tf-opsworks-acc",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_availability_zone",
			zone,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_os",
			"Amazon Linux 2014.09",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_root_device_type",
			"ebs",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"custom_json",
			`{"key": "value"}`,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"configuration_manager_version",
			"11.10",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"use_opsworks_security_groups",
			"false",
		),
	)
}

func testAccAwsOpsworksStackCheckResourceAttrsUpdate(zone string) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"name",
			"tf-opsworks-acc",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_availability_zone",
			zone,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_os",
			"Amazon Linux 2014.09",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_root_device_type",
			"ebs",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"custom_json",
			`{"key": "value"}`,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"configuration_manager_version",
			"11.10",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"use_opsworks_security_groups",
			"false",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"use_custom_cookbooks",
			"true",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"manage_berkshelf",
			"true",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"custom_cookbooks_source.0.type",
			"git",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"custom_cookbooks_source.0.revision",
			"master",
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"custom_cookbooks_source.0.url",
			"https://github.com/aws/opsworks-example-cookbooks.git",
		),
	)
}

func testAccAwsOpsworksCheckVpc(s *terraform.State) error {
	rs, ok := s.RootModule().Resources["aws_opsworks_stack.tf-acc"]
	if !ok {
		return fmt.Errorf("Not found: %s", "aws_opsworks_stack.tf-acc")
	}
	if rs.Primary.ID == "" {
		return fmt.Errorf("No ID is set")
	}

	p := rs.Primary

	opsworksconn := testAccProvider.Meta().(*AWSClient).opsworksconn
	describeOpts := &opsworks.DescribeStacksInput{
		StackIds: []*string{aws.String(p.ID)},
	}
	resp, err := opsworksconn.DescribeStacks(describeOpts)
	if err != nil {
		return err
	}
	if len(resp.Stacks) == 0 {
		return fmt.Errorf("No stack %s not found", p.ID)
	}
	if p.Attributes["vpc_id"] != *resp.Stacks[0].VpcId {
		return fmt.Errorf("VPCID Got %s, expected %s", *resp.Stacks[0].VpcId, p.Attributes["vpc_id"])
	}
	if p.Attributes["default_subnet_id"] != *resp.Stacks[0].DefaultSubnetId {
		return fmt.Errorf("VPCID Got %s, expected %s", *resp.Stacks[0].DefaultSubnetId, p.Attributes["default_subnet_id"])
	}
	return nil
}

func testAccCheckAwsOpsworksStackDestroy(s *terraform.State) error {
	opsworksconn := testAccProvider.Meta().(*AWSClient).opsworksconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_opsworks_stack" {
			continue
		}

		req := &opsworks.DescribeStacksInput{
			StackIds: []*string{
				aws.String(rs.Primary.ID),
			},
		}

		_, err := opsworksconn.DescribeStacks(req)
		if err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				if awserr.Code() == "ResourceNotFoundException" {
					// not found, all good
					return nil
				}
			}
			return err
		}
	}
	return fmt.Errorf("Fall through error for OpsWorks stack test")
}
