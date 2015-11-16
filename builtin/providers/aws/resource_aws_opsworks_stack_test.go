package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/opsworks"
)

// These tests assume the existence of predefined Opsworks IAM roles named `aws-opsworks-ec2-role`
// and `aws-opsworks-service-role`.

///////////////////////////////
//// Tests for the No-VPC case
///////////////////////////////

var testAccAwsOpsworksStackConfigNoVpcCreate = `
resource "aws_opsworks_stack" "tf-acc" {
  name = "tf-opsworks-acc"
  region = "us-west-2"
  service_role_arn = "%s"
  default_instance_profile_arn = "%s"
  default_availability_zone = "us-west-2a"
  default_os = "Amazon Linux 2014.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
}
`
var testAccAWSOpsworksStackConfigNoVpcUpdate = `
resource "aws_opsworks_stack" "tf-acc" {
  name = "tf-opsworks-acc"
  region = "us-west-2"
  service_role_arn = "%s"
  default_instance_profile_arn = "%s"
  default_availability_zone = "us-west-2a"
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

func TestAccAwsOpsworksStackNoVpc(t *testing.T) {
	opsiam := testAccAwsOpsworksStackIam{}
	testAccAwsOpsworksStackPopulateIam(t, &opsiam)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccAwsOpsworksStackConfigNoVpcCreate, opsiam.ServiceRoleArn, opsiam.InstanceProfileArn),
				Check:  testAccAwsOpsworksStackCheckResourceAttrsCreate,
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccAWSOpsworksStackConfigNoVpcUpdate, opsiam.ServiceRoleArn, opsiam.InstanceProfileArn),
				Check:  testAccAwsOpsworksStackCheckResourceAttrsUpdate,
			},
		},
	})
}

////////////////////////////
//// Tests for the VPC case
////////////////////////////

var testAccAwsOpsworksStackConfigVpcCreate = `
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
  service_role_arn = "%s"
  default_instance_profile_arn = "%s"
  default_os = "Amazon Linux 2014.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
}
`

var testAccAWSOpsworksStackConfigVpcUpdate = `
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
  service_role_arn = "%s"
  default_instance_profile_arn = "%s"
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

func TestAccAwsOpsworksStackVpc(t *testing.T) {
	opsiam := testAccAwsOpsworksStackIam{}
	testAccAwsOpsworksStackPopulateIam(t, &opsiam)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccAwsOpsworksStackConfigVpcCreate, opsiam.ServiceRoleArn, opsiam.InstanceProfileArn),
				Check:  testAccAwsOpsworksStackCheckResourceAttrsCreate,
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccAWSOpsworksStackConfigVpcUpdate, opsiam.ServiceRoleArn, opsiam.InstanceProfileArn),
				Check: resource.ComposeTestCheckFunc(
					testAccAwsOpsworksStackCheckResourceAttrsUpdate,
					testAccAwsOpsworksCheckVpc,
				),
			},
		},
	})
}

////////////////////////////
//// Checkers and Utilities
////////////////////////////

var testAccAwsOpsworksStackCheckResourceAttrsCreate = resource.ComposeTestCheckFunc(
	resource.TestCheckResourceAttr(
		"aws_opsworks_stack.tf-acc",
		"name",
		"tf-opsworks-acc",
	),
	resource.TestCheckResourceAttr(
		"aws_opsworks_stack.tf-acc",
		"default_availability_zone",
		"us-west-2a",
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

var testAccAwsOpsworksStackCheckResourceAttrsUpdate = resource.ComposeTestCheckFunc(
	resource.TestCheckResourceAttr(
		"aws_opsworks_stack.tf-acc",
		"name",
		"tf-opsworks-acc",
	),
	resource.TestCheckResourceAttr(
		"aws_opsworks_stack.tf-acc",
		"default_availability_zone",
		"us-west-2a",
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
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

// Holds the two IAM object ARNs used in stack objects we'll create.
type testAccAwsOpsworksStackIam struct {
	ServiceRoleArn     string
	InstanceProfileArn string
}

func testAccAwsOpsworksStackPopulateIam(t *testing.T, opsiam *testAccAwsOpsworksStackIam) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccInstanceConfig_pre, // noop
				Check:  testAccCheckAwsOpsworksEnsureIam(t, opsiam),
			},
		},
	})
}

func testAccCheckAwsOpsworksEnsureIam(t *testing.T, opsiam *testAccAwsOpsworksStackIam) func(*terraform.State) error {
	return func(_ *terraform.State) error {
		iamconn := testAccProvider.Meta().(*AWSClient).iamconn

		serviceRoleOpts := &iam.GetRoleInput{
			RoleName: aws.String("aws-opsworks-service-role"),
		}
		respServiceRole, err := iamconn.GetRole(serviceRoleOpts)
		if err != nil {
			return err
		}

		instanceProfileOpts := &iam.GetInstanceProfileInput{
			InstanceProfileName: aws.String("aws-opsworks-ec2-role"),
		}
		respInstanceProfile, err := iamconn.GetInstanceProfile(instanceProfileOpts)
		if err != nil {
			return err
		}

		opsiam.ServiceRoleArn = *respServiceRole.Role.Arn
		opsiam.InstanceProfileArn = *respInstanceProfile.InstanceProfile.Arn

		t.Logf("[DEBUG] ServiceRoleARN for OpsWorks: %s", opsiam.ServiceRoleArn)
		t.Logf("[DEBUG] Instance Profile ARN for OpsWorks: %s", opsiam.InstanceProfileArn)

		return nil

	}
}
