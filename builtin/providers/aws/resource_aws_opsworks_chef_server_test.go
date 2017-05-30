package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworkscm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsOpsworksChefServer(t *testing.T) {
	serverName := fmt.Sprintf("tf-acc-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAWSOpsworksChefServerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksChefServerConfig(serverName, "t2.medium"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAwsOpsworksChefServerExists(serverName, "t2.medium"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_chef_server.tf-acc", "name", serverName,
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_chef_server.tf-acc", "instance_type", "t2.medium",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_chef_server.tf-acc", "backup_automatically", "false",
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsOpsworksChefServerConfig(serverName, "m4.large"),
				Check: resource.ComposeTestCheckFunc(
					testCheckAwsOpsworksChefServerExists(serverName, "m4.large"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_chef_server.tf-acc", "name", serverName,
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_chef_server.tf-acc", "instance_type", "m4.large",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_chef_server.tf-acc", "backup_automatically", "false",
					),
				),
			},
		},
	})
}

// Returns a TestCheckFunc which validates the created resource after apply.
//
// The returned TestCheckFunc returns an error if the resource wasn't created
// correctly, or nil otherwise.
func testCheckAwsOpsworksChefServerExists(name string, instanceType string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// TODO: Capture response and validate resource attributes match
		connection := testAccProvider.Meta().(*AWSClient).opsworkscmconn
		response, err := connection.DescribeServers(
			&opsworkscm.DescribeServersInput{ServerName: aws.String(name)},
		)

		if err != nil {
			return err
		}

		server := response.Servers[0]
		if *server.ServerName != name {
			return fmt.Errorf("Chef server name mismatch:\nexpected: %#v\n  actual: %#v", name, *server.ServerName)
		}

		if *server.InstanceType != instanceType {
			return fmt.Errorf("Chef server type mismatch:\nexpected: %#v\n  actual: %#v", instanceType, *server.InstanceType)
		}

		return nil
	}
}

// Determines whether the resource was destroyed correctly after cleanup.
//
// Returns an error if the resource exists and destroy is required, or nil
// otherwise.
func testCheckAWSOpsworksChefServerDestroy(s *terraform.State) error {
	connection := testAccProvider.Meta().(*AWSClient).opsworkscmconn

	for _, resource := range s.RootModule().Resources {
		if resource.Type != "aws_opsworks_chef_server" {
			continue
		}

		response, err := connection.DescribeServers(
			&opsworkscm.DescribeServersInput{
				ServerName: aws.String(resource.Primary.ID),
			},
		)

		if err == nil {
			return fmt.Errorf("Opsworks Chef Server still exists:\n%#v", response.Servers)
		}

		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				continue // it must have been destroyed correctly
			}
		}

		return err // any other errors are unexpected
	}

	return nil
}

// Generates the Terraform configuration to use to create/update the server
func testAccAwsOpsworksChefServerConfig(name string, instanceType string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_chef_server" "tf-acc" {
	name = "%[1]s"

	instance_type = "%[2]s"
	subnet_ids    = ["${aws_subnet.tf-acc.id}"]

	associate_public_ip_address = true
	backup_automatically        = false

	instance_profile_arn = "${aws_iam_instance_profile.tf-acc.arn}"
	service_role_arn     = "${aws_iam_role.%[1]s-ServiceRole.arn}"

	// Explicitly depend on things this needs to function
	depends_on = [
		// The instance depends on the route table for internet access
		"aws_route_table_association.public",

		// Needs the policies to be attached to the roles in order to take effect
		"aws_iam_role_policy_attachment.%[1]s-InstanceRole-AmazonEC2RoleforSSM",
		"aws_iam_role_policy_attachment.%[1]s-InstanceRole-AWSOpsWorksCMInstanceProfileRole",
		"aws_iam_role_policy_attachment.%[1]s-InstanceRole-InstancePolicy",
		"aws_iam_role_policy_attachment.%[1]s-ServiceRole-AWSOpsWorksCMServiceRole",
		"aws_iam_role_policy_attachment.%[1]s-ServiceRole-OpsworksCMPolicy",
	]
}

// VPC and subnet to isolate all resources under test
resource "aws_vpc" "tf-acc" {
	cidr_block           = "10.3.5.0/24"
	enable_dns_hostnames = true
	enable_dns_support   = true
	tags {
		Name = "testAccAwsOpsworksChefServerConfigVpc"
	}
}
resource "aws_subnet" "tf-acc" {
	vpc_id            = "${aws_vpc.tf-acc.id}"
	cidr_block        = "${aws_vpc.tf-acc.cidr_block}"
	availability_zone = "us-east-1a"
	tags {
		Name = "testAccAwsOpsworksChefServerConfigVpc"
	}
}
resource "aws_internet_gateway" "tf-acc" {
	vpc_id = "${aws_vpc.tf-acc.id}"
  tags {
		Name = "testAccAwsOpsworksChefServerConfigVpc"
  }
}
resource "aws_route_table" "tf-acc" {
	vpc_id = "${aws_vpc.tf-acc.id}"
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.tf-acc.id}"
  }
  tags {
		Name = "testAccAwsOpsworksChefServerConfigVpc"
	}
}
resource "aws_route_table_association" "public" {
	route_table_id = "${aws_route_table.tf-acc.id}"
	subnet_id      = "${aws_subnet.tf-acc.id}"
}

// Create an instance profile
resource "aws_iam_instance_profile" "tf-acc" {
	name = "%[1]s-InstanceProfile"
	role = "${aws_iam_role.%[1]s-InstanceRole.name}"
}
resource "aws_iam_role" "%[1]s-InstanceRole" {
	name               = "%[1]s-InstanceRole"
	assume_role_policy = "${data.aws_iam_policy_document.%[1]s-InstanceRole-AssumeRolePolicy.json}"
}
data "aws_iam_policy_document" "%[1]s-InstanceRole-AssumeRolePolicy" {
	statement {
		actions = ["sts:AssumeRole"]
		principals {
			type        = "Service"
			identifiers = ["ec2.amazonaws.com"]
		}
	}
}
resource "aws_iam_role_policy_attachment" "%[1]s-InstanceRole-AmazonEC2RoleforSSM" {
	role       = "${aws_iam_role.%[1]s-InstanceRole.name}"
	policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2RoleforSSM"
}
resource "aws_iam_role_policy_attachment" "%[1]s-InstanceRole-AWSOpsWorksCMInstanceProfileRole" {
	role       = "${aws_iam_role.%[1]s-InstanceRole.name}"
	policy_arn = "arn:aws:iam::aws:policy/AWSOpsWorksCMInstanceProfileRole"
}
resource "aws_iam_role_policy_attachment" "%[1]s-InstanceRole-InstancePolicy" {
	role       = "${aws_iam_role.%[1]s-InstanceRole.name}"
	policy_arn = "${aws_iam_policy.%[1]s-InstancePolicy.arn}"
}
resource "aws_iam_policy" "%[1]s-InstancePolicy" {
	name   = "%[1]s-InstancePolicy"
	policy = "${data.aws_iam_policy_document.%[1]s-InstancePolicy.json}"
}
data "aws_iam_policy_document" "%[1]s-InstancePolicy" {
	statement {
		actions = [
			"s3:DeleteObject",
			"s3:PutObject",
			"s3:AbortMultipartUpload",
			"s3:List*",
			"s3:Get*",
		]
		resources = ["arn:aws:s3:::aws-*"]
	}
}

// Create Service Role
resource "aws_iam_role" "%[1]s-ServiceRole" {
	name               = "%[1]s-ServiceRole"
	assume_role_policy = "${data.aws_iam_policy_document.%[1]s-ServiceRole-AssumeRolePolicy.json}"
}
data "aws_iam_policy_document" "%[1]s-ServiceRole-AssumeRolePolicy" {
	statement {
		actions = ["sts:AssumeRole"]
		principals {
			type        = "Service"
			identifiers = ["opsworks-cm.amazonaws.com"]
		}
	}
}
resource "aws_iam_role_policy_attachment" "%[1]s-ServiceRole-AWSOpsWorksCMServiceRole" {
	role       = "${aws_iam_role.%[1]s-ServiceRole.name}"
	policy_arn = "arn:aws:iam::aws:policy/service-role/AWSOpsWorksCMServiceRole"
}
resource "aws_iam_role_policy_attachment" "%[1]s-ServiceRole-OpsworksCMPolicy" {
	role       = "${aws_iam_role.%[1]s-ServiceRole.name}"
	policy_arn = "${aws_iam_policy.%[1]s-OpsworksCMPolicy.arn}"
}
resource "aws_iam_policy" "%[1]s-OpsworksCMPolicy" {
	name   = "%[1]s-OpsworksCMPolicy"
	policy = "${data.aws_iam_policy_document.%[1]s-OpsworksCMPolicy.json}"
}

data "aws_iam_policy_document" "%[1]s-OpsworksCMPolicy" {
	statement {
		actions   = [
			"s3:CreateBucket",
			"s3:DeleteBucket",
			"s3:DeleteObject",
			"s3:GetObject",
			"s3:HeadBucket",
			"s3:ListBucket",
			"s3:ListObjects",
		]
		resources = ["*"]
	}

	statement {
		actions   = [
			"ssm:DescribeAssociation",
			"ssm:GetDocument",
			"ssm:ListAssociations",
			"ssm:UpdateAssociationStatus",
			"ssm:UpdateInstanceInformation",
			"ssm:SendCommand",
			"ssm:ListCommandInvocations",
			"ssm:ListCommands",
			"ssm:SendCommand",
		]
		resources = ["*"]
	}

	statement {
		actions   = ["iam:PassRole"]
		resources = ["*"]
	}

	statement {
		actions   = [
			"ec2:CreateSecurityGroup",
			"ec2:AuthorizeSecurityGroupIngress",
			"ec2:RunInstances",
			"ec2:DescribeAccountAttributes",
			"ec2:DescribeInstanceStatus",
			"ec2:DescribeSecurityGroups",
			"ec2:createTags",
			"ec2:AllocateAddress",
			"ec2:AssociateAddress",
			"ec2:DescribeInstances",
			"ec2:DescribeAddresses",
			"ec2:DescribeSubnets",
			"ec2:DeleteSecurityGroup",
			"ec2:DisassociateAddress",
			"ec2:ReleaseAddress",
			"ec2:TerminateInstances",
		]
		resources = ["*"]
	}

	statement {
		actions   = [
			"cloudformation:CreateStack",
			"cloudformation:UpdateStack",
			"cloudformation:DescribeStacks",
			"cloudformation:DescribeStackEvents",
			"cloudformation:DeleteStack",
			"cloudformation:DescribeStackResources",
		]
		resources = ["*"]
	}
}
	`, name, instanceType)
}
