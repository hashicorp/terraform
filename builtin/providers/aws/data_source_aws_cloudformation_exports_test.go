package aws

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCloudformationExports_dataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsCloudformationExportsJson,
			},
			resource.TestStep{
				Config: testAccAWSCloudformationExportsDataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_cloudformation_exports.cfn", "values.waiter", "waiter"),
					resource.TestMatchResourceAttr("data.aws_cloudformation_exports.cfn", "values.my-vpc-id",
						regexp.MustCompile("^vpc-[a-z0-9]{8}$")),
					resource.TestMatchResourceAttr("data.aws_cloudformation_exports.cfn", "stack_ids.waiter",
						regexp.MustCompile("^arn:aws:cloudformation")),
				),
			},
		},
	})
}

const testAccCheckAwsCloudformationExportsJson = `
resource "aws_cloudformation_stack" "cfs" {
  name = "tf-waiter-stack"
  timeout_in_minutes = 6
  template_body = <<STACK
{
  "Resources": {
    "waiter": {
      "Type": "AWS::CloudFormation::WaitConditionHandle",
      "Properties": { }
    }
  },
  "Outputs": {
    "WaitHandle": {
      "Value": "waiter" ,
      "Description": "VPC ID",
      "Export": {
        "Name": "waiter" 
      }
    }
  }
}
STACK
}
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
    Export:
      Name: my-vpc-id
STACK
  tags {
    Name = "Form the Cloud"
    Second = "meh"
  }
}
`
const testAccAWSCloudformationExportsDataSource = `
data "aws_cloudformation_exports" "cfn" { }
`
