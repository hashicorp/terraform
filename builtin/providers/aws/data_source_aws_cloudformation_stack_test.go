package aws

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCloudFormationStack_dataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsCloudFormationStackDataSourceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "outputs.%", "1"),
					resource.TestMatchResourceAttr("data.aws_cloudformation_stack.network", "outputs.VPCId",
						regexp.MustCompile("^vpc-[a-z0-9]{8}$")),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "capabilities.#", "0"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "disable_rollback", "false"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "notification_arns.#", "0"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "parameters.%", "1"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "parameters.CIDR", "10.10.10.0/24"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "timeout_in_minutes", "6"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "tags.%", "2"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "tags.Name", "Form the Cloud"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.network", "tags.Second", "meh"),
				),
			},
		},
	})
}

const testAccCheckAwsCloudFormationStackDataSourceConfig_basic = `
resource "aws_cloudformation_stack" "cfs" {
  name = "tf-acc-ds-networking-stack"
  parameters {
    CIDR = "10.10.10.0/24"
  }
  timeout_in_minutes = 6
  template_body = <<STACK
{
  "Parameters": {
    "CIDR": {
      "Type": "String"
    }
  },
  "Resources" : {
    "myvpc": {
      "Type" : "AWS::EC2::VPC",
      "Properties" : {
        "CidrBlock" : { "Ref" : "CIDR" },
        "Tags" : [
          {"Key": "Name", "Value": "Primary_CF_VPC"}
        ]
      }
    }
  },
  "Outputs" : {
    "VPCId" : {
      "Value" : { "Ref" : "myvpc" },
      "Description" : "VPC ID"
    }
  }
}
STACK
  tags {
    Name = "Form the Cloud"
    Second = "meh"
  }
}

data "aws_cloudformation_stack" "network" {
  name = "${aws_cloudformation_stack.cfs.name}"
}
`

func TestAccAWSCloudFormationStack_dataSource_yaml(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsCloudFormationStackDataSourceConfig_yaml,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "outputs.%", "1"),
					resource.TestMatchResourceAttr("data.aws_cloudformation_stack.yaml", "outputs.VPCId",
						regexp.MustCompile("^vpc-[a-z0-9]{8}$")),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "capabilities.#", "0"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "disable_rollback", "false"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "notification_arns.#", "0"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "parameters.%", "1"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "parameters.CIDR", "10.10.10.0/24"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "timeout_in_minutes", "6"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "tags.%", "2"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "tags.Name", "Form the Cloud"),
					resource.TestCheckResourceAttr("data.aws_cloudformation_stack.yaml", "tags.Second", "meh"),
				),
			},
		},
	})
}

const testAccCheckAwsCloudFormationStackDataSourceConfig_yaml = `
resource "aws_cloudformation_stack" "yaml" {
  name = "tf-acc-ds-yaml-stack"
  parameters {
    CIDR = "10.10.10.0/24"
  }
  timeout_in_minutes = 6
  template_body = <<STACK
Parameters:
  CIDR:
    Type: String

Resources:
  myvpc:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: !Ref CIDR
      Tags:
        -
          Key: Name
          Value: Primary_CF_VPC

Outputs:
  VPCId:
    Value: !Ref myvpc
    Description: VPC ID
STACK
  tags {
    Name = "Form the Cloud"
    Second = "meh"
  }
}

data "aws_cloudformation_stack" "yaml" {
  name = "${aws_cloudformation_stack.yaml.name}"
}
`
