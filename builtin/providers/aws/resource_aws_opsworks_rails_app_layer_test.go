package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// These tests assume the existence of predefined Opsworks IAM roles named `aws-opsworks-ec2-role`
// and `aws-opsworks-service-role`.

func TestAccAWSOpsworksRailsAppLayer(t *testing.T) {
	stackName := fmt.Sprintf("tf-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksRailsAppLayerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksRailsAppLayerConfigVpcCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_rails_app_layer.tf-acc", "name", stackName,
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_rails_app_layer.tf-acc", "manage_bundler", "true",
					),
				),
			},
			{
				Config: testAccAwsOpsworksRailsAppLayerNoManageBundlerConfigVpcCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_rails_app_layer.tf-acc", "name", stackName,
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_rails_app_layer.tf-acc", "manage_bundler", "false",
					),
				),
			},
		},
	})
}

func testAccCheckAwsOpsworksRailsAppLayerDestroy(s *terraform.State) error {
	opsworksconn := testAccProvider.Meta().(*AWSClient).opsworksconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_opsworks_rails_app_layer" {
			continue
		}
		req := &opsworks.DescribeLayersInput{
			LayerIds: []*string{
				aws.String(rs.Primary.ID),
			},
		}

		_, err := opsworksconn.DescribeLayers(req)
		if err != nil {
			if awserr, ok := err.(awserr.Error); ok {
				if awserr.Code() == "ResourceNotFoundException" {
					// not found, good to go
					return nil
				}
			}
			return err
		}
	}

	return fmt.Errorf("Fall through error on OpsWorks custom layer test")
}

func testAccAwsOpsworksRailsAppLayerConfigVpcCreate(name string) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "us-west-2"
}

resource "aws_opsworks_rails_app_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"
  name = "%s"
  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-layer1.id}",
    "${aws_security_group.tf-ops-acc-layer2.id}",
  ]
}

%s

%s

`, name, testAccAwsOpsworksStackConfigVpcCreate(name), testAccAwsOpsworksCustomLayerSecurityGroups(name))
}

func testAccAwsOpsworksRailsAppLayerNoManageBundlerConfigVpcCreate(name string) string {
	return fmt.Sprintf(`
provider "aws" {
	region = "us-west-2"
}

resource "aws_opsworks_rails_app_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"
  name = "%s"
  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-layer1.id}",
    "${aws_security_group.tf-ops-acc-layer2.id}",
  ]
  manage_bundler = false
}

%s

%s

`, name, testAccAwsOpsworksStackConfigVpcCreate(name), testAccAwsOpsworksCustomLayerSecurityGroups(name))
}
