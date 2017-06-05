package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
)

///////////////////////////////
//// Tests for the No-VPC case
///////////////////////////////

func TestAccAWSOpsworksStackNoVpc(t *testing.T) {
	stackName := fmt.Sprintf("tf-opsworks-acc-%d", acctest.RandInt())
	var opsstack opsworks.Stack
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksStackConfigNoVpcCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksStackExists(
						"aws_opsworks_stack.tf-acc", false, &opsstack),
					testAccCheckAWSOpsworksCreateStackAttributes(
						&opsstack, "us-east-1a", stackName),
					testAccAwsOpsworksStackCheckResourceAttrsCreate(
						"us-east-1a", stackName),
				),
			},
		},
	})
}

func TestAccAWSOpsworksStackNoVpcChangeServiceRoleForceNew(t *testing.T) {
	stackName := fmt.Sprintf("tf-opsworks-acc-%d", acctest.RandInt())
	var before, after opsworks.Stack
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksStackConfigNoVpcCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksStackExists(
						"aws_opsworks_stack.tf-acc", false, &before),
				),
			},
			{
				Config: testAccAwsOpsworksStackConfigNoVpcCreateUpdateServiceRole(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksStackExists(
						"aws_opsworks_stack.tf-acc", false, &after),
					testAccCheckAWSOpsworksStackRecreated(t, &before, &after),
				),
			},
		},
	})
}

func TestAccAWSOpsworksStackVpc(t *testing.T) {
	stackName := fmt.Sprintf("tf-opsworks-acc-%d", acctest.RandInt())
	var opsstack opsworks.Stack
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksStackConfigVpcCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksStackExists(
						"aws_opsworks_stack.tf-acc", true, &opsstack),
					testAccCheckAWSOpsworksCreateStackAttributes(
						&opsstack, "us-west-2a", stackName),
					testAccAwsOpsworksStackCheckResourceAttrsCreate(
						"us-west-2a", stackName),
				),
			},
			{
				Config: testAccAWSOpsworksStackConfigVpcUpdate(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksStackExists(
						"aws_opsworks_stack.tf-acc", true, &opsstack),
					testAccCheckAWSOpsworksUpdateStackAttributes(
						&opsstack, "us-west-2a", stackName),
					testAccAwsOpsworksStackCheckResourceAttrsUpdate(
						"us-west-2a", stackName),
				),
			},
		},
	})
}

// Tests the addition of regional endpoints and supporting the classic link used
// to create Stack's prior to v0.9.0.
// See https://github.com/hashicorp/terraform/issues/12842
func TestAccAWSOpsWorksStack_classic_endpoints(t *testing.T) {
	stackName := fmt.Sprintf("tf-opsworks-acc-%d", acctest.RandInt())
	rInt := acctest.RandInt()
	var opsstack opsworks.Stack
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksStackDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsWorksStack_classic_endpoint(stackName, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksStackExists(
						"aws_opsworks_stack.main", false, &opsstack),
				),
			},
			// Ensure that changing to us-west-2 region results in no plan
			{
				Config:   testAccAwsOpsWorksStack_regional_endpoint(stackName, rInt),
				PlanOnly: true,
			},
		},
	})

}

func testAccCheckAWSOpsworksStackRecreated(t *testing.T,
	before, after *opsworks.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.StackId == *after.StackId {
			t.Fatalf("Expected change of Opsworks StackIds, but both were %v", before.StackId)
		}
		return nil
	}
}

func testAccAwsOpsWorksStack_classic_endpoint(rName string, rInt int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}

resource "aws_opsworks_stack" "main" {
  name                         = "%s"
  region = "us-west-2"
  service_role_arn             = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"

  configuration_manager_version = "12"
  default_availability_zone = "us-west-2b"
}

resource "aws_iam_role" "opsworks_service" {
  name = "tf_opsworks_service_%d"

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
  name = "tf_opsworks_service_%d"
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
  name = "tf_opsworks_instance_%d"

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
  name  = "%s_profile"
  roles = ["${aws_iam_role.opsworks_instance.name}"]
}`, rName, rInt, rInt, rInt, rName)
}

func testAccAwsOpsWorksStack_regional_endpoint(rName string, rInt int) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-west-2"
}

resource "aws_opsworks_stack" "main" {
  name                         = "%s"
  region = "us-west-2"
  service_role_arn             = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"

  configuration_manager_version = "12"
  default_availability_zone = "us-west-2b"
}

resource "aws_iam_role" "opsworks_service" {
  name = "tf_opsworks_service_%d"

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
  name = "tf_opsworks_service_%d"
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
  name = "tf_opsworks_instance_%d"

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
  name  = "%s_profile"
  roles = ["${aws_iam_role.opsworks_instance.name}"]
}`, rName, rInt, rInt, rInt, rName)
}

////////////////////////////
//// Checkers and Utilities
////////////////////////////

func testAccAwsOpsworksStackCheckResourceAttrsCreate(zone, stackName string) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"name",
			stackName,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_availability_zone",
			zone,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_os",
			"Amazon Linux 2016.09",
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

func testAccAwsOpsworksStackCheckResourceAttrsUpdate(zone, stackName string) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"name",
			stackName,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_availability_zone",
			zone,
		),
		resource.TestCheckResourceAttr(
			"aws_opsworks_stack.tf-acc",
			"default_os",
			"Amazon Linux 2015.09",
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

func testAccCheckAWSOpsworksStackExists(
	n string, vpc bool, opsstack *opsworks.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).opsworksconn

		params := &opsworks.DescribeStacksInput{
			StackIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeStacks(params)

		if err != nil {
			return err
		}

		if v := len(resp.Stacks); v != 1 {
			return fmt.Errorf("Expected 1 response returned, got %d", v)
		}

		*opsstack = *resp.Stacks[0]

		if vpc {
			if rs.Primary.Attributes["vpc_id"] != *opsstack.VpcId {
				return fmt.Errorf("VPCID Got %s, expected %s", *opsstack.VpcId, rs.Primary.Attributes["vpc_id"])
			}
			if rs.Primary.Attributes["default_subnet_id"] != *opsstack.DefaultSubnetId {
				return fmt.Errorf("Default subnet Id Got %s, expected %s", *opsstack.DefaultSubnetId, rs.Primary.Attributes["default_subnet_id"])
			}
		}

		return nil
	}
}

func testAccCheckAWSOpsworksCreateStackAttributes(
	opsstack *opsworks.Stack, zone, stackName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *opsstack.Name != stackName {
			return fmt.Errorf("Unnexpected stackName: %s", *opsstack.Name)
		}

		if *opsstack.DefaultAvailabilityZone != zone {
			return fmt.Errorf("Unnexpected DefaultAvailabilityZone: %s", *opsstack.DefaultAvailabilityZone)
		}

		if *opsstack.DefaultOs != "Amazon Linux 2016.09" {
			return fmt.Errorf("Unnexpected stackName: %s", *opsstack.DefaultOs)
		}

		if *opsstack.DefaultRootDeviceType != "ebs" {
			return fmt.Errorf("Unnexpected DefaultRootDeviceType: %s", *opsstack.DefaultRootDeviceType)
		}

		if *opsstack.CustomJson != `{"key": "value"}` {
			return fmt.Errorf("Unnexpected CustomJson: %s", *opsstack.CustomJson)
		}

		if *opsstack.ConfigurationManager.Version != "11.10" {
			return fmt.Errorf("Unnexpected Version: %s", *opsstack.ConfigurationManager.Version)
		}

		if *opsstack.UseOpsworksSecurityGroups {
			return fmt.Errorf("Unnexpected UseOpsworksSecurityGroups: %t", *opsstack.UseOpsworksSecurityGroups)
		}

		return nil
	}
}

func testAccCheckAWSOpsworksUpdateStackAttributes(
	opsstack *opsworks.Stack, zone, stackName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *opsstack.Name != stackName {
			return fmt.Errorf("Unnexpected stackName: %s", *opsstack.Name)
		}

		if *opsstack.DefaultAvailabilityZone != zone {
			return fmt.Errorf("Unnexpected DefaultAvailabilityZone: %s", *opsstack.DefaultAvailabilityZone)
		}

		if *opsstack.DefaultOs != "Amazon Linux 2015.09" {
			return fmt.Errorf("Unnexpected stackName: %s", *opsstack.DefaultOs)
		}

		if *opsstack.DefaultRootDeviceType != "ebs" {
			return fmt.Errorf("Unnexpected DefaultRootDeviceType: %s", *opsstack.DefaultRootDeviceType)
		}

		if *opsstack.CustomJson != `{"key": "value"}` {
			return fmt.Errorf("Unnexpected CustomJson: %s", *opsstack.CustomJson)
		}

		if *opsstack.ConfigurationManager.Version != "11.10" {
			return fmt.Errorf("Unnexpected Version: %s", *opsstack.ConfigurationManager.Version)
		}

		if !*opsstack.UseCustomCookbooks {
			return fmt.Errorf("Unnexpected UseCustomCookbooks: %t", *opsstack.UseCustomCookbooks)
		}

		if !*opsstack.ChefConfiguration.ManageBerkshelf {
			return fmt.Errorf("Unnexpected ManageBerkshelf: %t", *opsstack.ChefConfiguration.ManageBerkshelf)
		}

		if *opsstack.CustomCookbooksSource.Type != "git" {
			return fmt.Errorf("Unnexpected *opsstack.CustomCookbooksSource.Type: %s", *opsstack.CustomCookbooksSource.Type)
		}

		if *opsstack.CustomCookbooksSource.Revision != "master" {
			return fmt.Errorf("Unnexpected *opsstack.CustomCookbooksSource.Type: %s", *opsstack.CustomCookbooksSource.Revision)
		}

		if *opsstack.CustomCookbooksSource.Url != "https://github.com/aws/opsworks-example-cookbooks.git" {
			return fmt.Errorf("Unnexpected *opsstack.CustomCookbooksSource.Type: %s", *opsstack.CustomCookbooksSource.Url)
		}

		return nil
	}
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

//////////////////////////////////////////////////
//// Helper configs for the necessary IAM objects
//////////////////////////////////////////////////

func testAccAwsOpsworksStackConfigNoVpcCreate(name string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_opsworks_stack" "tf-acc" {
  name = "%s"
  region = "us-east-1"
  service_role_arn = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_availability_zone = "us-east-1a"
  default_os = "Amazon Linux 2016.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
}

resource "aws_iam_role" "opsworks_service" {
    name = "%s_opsworks_service"
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
    name = "%s_opsworks_service"
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
    name = "%s_opsworks_instance"
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
    name = "%s_opsworks_instance"
    roles = ["${aws_iam_role.opsworks_instance.name}"]
}`, name, name, name, name, name)
}

func testAccAwsOpsworksStackConfigNoVpcCreateUpdateServiceRole(name string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}
resource "aws_opsworks_stack" "tf-acc" {
  name = "%s"
  region = "us-east-1"
  service_role_arn = "${aws_iam_role.opsworks_service_new.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_availability_zone = "us-east-1a"
  default_os = "Amazon Linux 2016.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
}

resource "aws_iam_role" "opsworks_service" {
    name = "%s_opsworks_service"
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

resource "aws_iam_role" "opsworks_service_new" {
    name = "%s_opsworks_service_new"
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

resource "aws_iam_role_policy" "opsworks_service_new" {
    name = "%s_opsworks_service_new"
    role = "${aws_iam_role.opsworks_service_new.id}"
    policy = <<EOT
{
  "Statement": [
    {
      "Action": [
        "ec2:*",
        "iam:PassRole",
        "cloudwatch:*",
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

resource "aws_iam_role_policy" "opsworks_service" {
    name = "%s_opsworks_service"
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
    name = "%s_opsworks_instance"
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
    name = "%s_opsworks_instance"
    roles = ["${aws_iam_role.opsworks_instance.name}"]
}`, name, name, name, name, name, name, name)
}

////////////////////////////
//// Tests for the VPC case
////////////////////////////

func testAccAwsOpsworksStackConfigVpcCreate(name string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "tf-acc" {
  cidr_block = "10.3.5.0/24"
	tags {
		Name = "testAccAwsOpsworksStackConfigVpcCreate"
	}
}
resource "aws_subnet" "tf-acc" {
  vpc_id = "${aws_vpc.tf-acc.id}"
  cidr_block = "${aws_vpc.tf-acc.cidr_block}"
  availability_zone = "us-west-2a"
}
resource "aws_opsworks_stack" "tf-acc" {
  name = "%s"
  region = "us-west-2"
  vpc_id = "${aws_vpc.tf-acc.id}"
  default_subnet_id = "${aws_subnet.tf-acc.id}"
  service_role_arn = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_os = "Amazon Linux 2016.09"
  default_root_device_type = "ebs"
  custom_json = "{\"key\": \"value\"}"
  configuration_manager_version = "11.10"
  use_opsworks_security_groups = false
}

resource "aws_iam_role" "opsworks_service" {
    name = "%s_opsworks_service"
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
    name = "%s_opsworks_service"
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
    name = "%s_opsworks_instance"
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
    name = "%s_opsworks_instance"
    roles = ["${aws_iam_role.opsworks_instance.name}"]
}
`, name, name, name, name, name)
}

func testAccAWSOpsworksStackConfigVpcUpdate(name string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "tf-acc" {
  cidr_block = "10.3.5.0/24"
	tags {
		Name = "testAccAWSOpsworksStackConfigVpcUpdate"
	}
}
resource "aws_subnet" "tf-acc" {
  vpc_id = "${aws_vpc.tf-acc.id}"
  cidr_block = "${aws_vpc.tf-acc.cidr_block}"
  availability_zone = "us-west-2a"
}
resource "aws_opsworks_stack" "tf-acc" {
  name = "%s"
  region = "us-west-2"
  vpc_id = "${aws_vpc.tf-acc.id}"
  default_subnet_id = "${aws_subnet.tf-acc.id}"
  service_role_arn = "${aws_iam_role.opsworks_service.arn}"
  default_instance_profile_arn = "${aws_iam_instance_profile.opsworks_instance.arn}"
  default_os = "Amazon Linux 2015.09"
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

resource "aws_iam_role" "opsworks_service" {
    name = "%s_opsworks_service"
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
    name = "%s_opsworks_service"
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
    name = "%s_opsworks_instance"
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
    name = "%s_opsworks_instance"
    roles = ["${aws_iam_role.opsworks_instance.name}"]
}

`, name, name, name, name, name)
}
