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

func TestAccAWSOpsworksInstance(t *testing.T) {
	stackName := fmt.Sprintf("tf-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksInstanceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksInstanceConfigCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "hostname", "tf-acc1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "instance_type", "t2.micro",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "state", "stopped",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "layer_ids.#", "1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "install_updates_on_boot", "true",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "architecture", "x86_64",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "os", "Amazon Linux 2014.09", // inherited from opsworks_stack_test
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "root_device_type", "ebs", // inherited from opsworks_stack_test
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "availability_zone", "us-west-2a", // inherited from opsworks_stack_test
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsOpsworksInstanceConfigUpdate(stackName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "hostname", "tf-acc1",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "instance_type", "t2.small",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "layer_ids.#", "2",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "os", "Amazon Linux 2015.09",
					),
				),
			},
		},
	})
}

func testAccCheckAwsOpsworksInstanceDestroy(s *terraform.State) error {
	opsworksconn := testAccProvider.Meta().(*AWSClient).opsworksconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_opsworks_instance" {
			continue
		}
		req := &opsworks.DescribeInstancesInput{
			InstanceIds: []*string{
				aws.String(rs.Primary.ID),
			},
		}

		_, err := opsworksconn.DescribeInstances(req)
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

	return fmt.Errorf("Fall through error on OpsWorks instance test")
}

func testAccAwsOpsworksInstanceConfigCreate(name string) string {
	return fmt.Sprintf(`
resource "aws_security_group" "tf-ops-acc-web" {
  name = "%s-web"
  ingress {
    from_port = 80
    to_port = 80
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "tf-ops-acc-php" {
  name = "%s-php"
  ingress {
    from_port = 8080
    to_port = 8080
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_opsworks_static_web_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-web.id}",
  ]
}

resource "aws_opsworks_php_app_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-php.id}",
  ]
}

resource "aws_opsworks_instance" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"
  layer_ids = [
    "${aws_opsworks_static_web_layer.tf-acc.id}",
  ]
  instance_type = "t2.micro"
  state = "stopped"
  hostname = "tf-acc1"
}

%s

`, name, name, testAccAwsOpsworksStackConfigVpcCreate(name))
}

func testAccAwsOpsworksInstanceConfigUpdate(name string) string {
	return fmt.Sprintf(`
resource "aws_security_group" "tf-ops-acc-web" {
  name = "%s-web"
  ingress {
    from_port = 80
    to_port = 80
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "tf-ops-acc-php" {
  name = "%s-php"
  ingress {
    from_port = 8080
    to_port = 8080
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_opsworks_static_web_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-web.id}",
  ]
}

resource "aws_opsworks_php_app_layer" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  custom_security_group_ids = [
    "${aws_security_group.tf-ops-acc-php.id}",
  ]
}

resource "aws_opsworks_instance" "tf-acc" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"
  layer_ids = [
    "${aws_opsworks_static_web_layer.tf-acc.id}",
    "${aws_opsworks_php_app_layer.tf-acc.id}",
  ]
  instance_type = "t2.small"
  state = "stopped"
  hostname = "tf-acc1"
  os = "Amazon Linux 2015.09"
}

%s

`, name, name, testAccAwsOpsworksStackConfigVpcCreate(name))
}
