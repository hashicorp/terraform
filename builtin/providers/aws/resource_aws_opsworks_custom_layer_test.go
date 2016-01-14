package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// These tests assume the existence of predefined Opsworks IAM roles named `aws-opsworks-ec2-role`
// and `aws-opsworks-service-role`.

func TestAccAWSOpsworksCustomLayer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksCustomLayerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksCustomLayerConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "name", "tf-ops-acc-custom-layer",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "auto_assign_elastic_ips", "false",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "auto_healing", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "drain_elb_on_shutdown", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "instance_shutdown_timeout", "300",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "custom_security_group_ids.#", "2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "system_packages.#", "2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "system_packages.1368285564", "git",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "system_packages.2937857443", "golang",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.#", "1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.type", "gp2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.number_of_disks", "2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.mount_point", "/home",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.size", "100",
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsOpsworksCustomLayerConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "name", "tf-ops-acc-custom-layer",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "drain_elb_on_shutdown", "false",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "instance_shutdown_timeout", "120",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "custom_security_group_ids.#", "3",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "system_packages.#", "3",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "system_packages.1368285564", "git",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "system_packages.2937857443", "golang",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "system_packages.4101929740", "subversion",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.#", "2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.type", "gp2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.number_of_disks", "2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.mount_point", "/home",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.3575749636.size", "100",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.1266957920.type", "io1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.1266957920.number_of_disks", "4",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.1266957920.mount_point", "/var",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.1266957920.size", "100",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.1266957920.raid_level", "1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_custom_layer.tf-acc", "ebs_volume.1266957920.iops", "3000",
					),
				),
			},
		},
	})
}

func testAccCheckAwsOpsworksCustomLayerDestroy(s *terraform.State) error {
	opsworksconn := testAccProvider.Meta().(*AWSClient).opsworksconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_opsworks_custom_layer" {
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

var testAccAwsOpsworksCustomLayerSecurityGroups = `
resource "aws_security_group" "tf-ops-acc-layer1" {
  name = "tf-ops-acc-layer1"
  ingress {
    from_port = 8
    to_port = -1
    protocol = "icmp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
resource "aws_security_group" "tf-ops-acc-layer2" {
  name = "tf-ops-acc-layer2"
  ingress {
    from_port = 8
    to_port = -1
    protocol = "icmp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
`

var testAccAwsOpsworksCustomLayerConfigCreate = testAccAwsOpsworksStackConfigNoVpcCreate + testAccAwsOpsworksCustomLayerSecurityGroups + `
provider "aws" {
	region = "us-east-1"
}

resource "aws_opsworks_custom_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"
  name = "tf-ops-acc-custom-layer"
  short_name = "tf-ops-acc-custom-layer"
  auto_assign_public_ips = true
  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-layer1.id}",
    "${aws_security_group.tf-ops-acc-layer2.id}",
  ]
  drain_elb_on_shutdown = true
  instance_shutdown_timeout = 300
  system_packages = [
    "git",
    "golang",
  ]
  ebs_volume {
    type = "gp2"
    number_of_disks = 2
    mount_point = "/home"
    size = 100
    raid_level = 0
  }
}
`

var testAccAwsOpsworksCustomLayerConfigUpdate = testAccAwsOpsworksStackConfigNoVpcCreate + testAccAwsOpsworksCustomLayerSecurityGroups + `
resource "aws_security_group" "tf-ops-acc-layer3" {
  name = "tf-ops-acc-layer3"
  ingress {
    from_port = 8
    to_port = -1
    protocol = "icmp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
resource "aws_opsworks_custom_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"
  name = "tf-ops-acc-custom-layer"
  short_name = "tf-ops-acc-custom-layer"
  auto_assign_public_ips = true
  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-layer1.id}",
    "${aws_security_group.tf-ops-acc-layer2.id}",
    "${aws_security_group.tf-ops-acc-layer3.id}",
  ]
  drain_elb_on_shutdown = false
  instance_shutdown_timeout = 120
  system_packages = [
    "git",
    "golang",
    "subversion",
  ]
  ebs_volume {
    type = "gp2"
    number_of_disks = 2
    mount_point = "/home"
    size = 100
    raid_level = 0
  }
  ebs_volume {
    type = "io1"
    number_of_disks = 4
    mount_point = "/var"
    size = 100
    raid_level = 1
    iops = 3000
  }
}
`
