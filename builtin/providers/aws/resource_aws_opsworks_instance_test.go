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

func TestAccAWSOpsworksInstance_importBasic(t *testing.T) {
	stackName := fmt.Sprintf("tf-%d", acctest.RandInt())
	resourceName := "aws_opsworks_instance.tf-acc"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksInstanceConfigCreate(stackName),
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"}, //state is something we pass to the API and get back as status :(
			},
		},
	})
}

func TestAccAWSOpsworksInstance(t *testing.T) {
	stackName := fmt.Sprintf("tf-%d", acctest.RandInt())
	var opsinst opsworks.Instance
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksInstanceConfigCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksInstanceExists(
						"aws_opsworks_instance.tf-acc", &opsinst),
					testAccCheckAWSOpsworksInstanceAttributes(&opsinst),
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
						"aws_opsworks_instance.tf-acc", "tenancy", "default",
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "os", "Amazon Linux 2016.09", // inherited from opsworks_stack_test
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "root_device_type", "ebs", // inherited from opsworks_stack_test
					),
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "availability_zone", "us-west-2a", // inherited from opsworks_stack_test
					),
				),
			},
			{
				Config: testAccAwsOpsworksInstanceConfigUpdate(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksInstanceExists(
						"aws_opsworks_instance.tf-acc", &opsinst),
					testAccCheckAWSOpsworksInstanceAttributes(&opsinst),
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
					resource.TestCheckResourceAttr(
						"aws_opsworks_instance.tf-acc", "tenancy", "default",
					),
				),
			},
		},
	})
}

func TestAccAWSOpsworksInstance_UpdateHostNameForceNew(t *testing.T) {
	stackName := fmt.Sprintf("tf-%d", acctest.RandInt())

	var before, after opsworks.Instance
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksInstanceConfigCreate(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksInstanceExists("aws_opsworks_instance.tf-acc", &before),
					resource.TestCheckResourceAttr("aws_opsworks_instance.tf-acc", "hostname", "tf-acc1"),
				),
			},
			{
				Config: testAccAwsOpsworksInstanceConfigUpdateHostName(stackName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksInstanceExists("aws_opsworks_instance.tf-acc", &after),
					resource.TestCheckResourceAttr("aws_opsworks_instance.tf-acc", "hostname", "tf-acc2"),
					testAccCheckAwsOpsworksInstanceRecreated(t, &before, &after),
				),
			},
		},
	})
}

func testAccCheckAwsOpsworksInstanceRecreated(t *testing.T,
	before, after *opsworks.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.InstanceId == *after.InstanceId {
			t.Fatalf("Expected change of OpsWorks Instance IDs, but both were %s", *before.InstanceId)
		}
		return nil
	}
}

func testAccCheckAWSOpsworksInstanceExists(
	n string, opsinst *opsworks.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Opsworks Instance is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).opsworksconn

		params := &opsworks.DescribeInstancesInput{
			InstanceIds: []*string{&rs.Primary.ID},
		}
		resp, err := conn.DescribeInstances(params)

		if err != nil {
			return err
		}

		if v := len(resp.Instances); v != 1 {
			return fmt.Errorf("Expected 1 request returned, got %d", v)
		}

		*opsinst = *resp.Instances[0]

		return nil
	}
}

func testAccCheckAWSOpsworksInstanceAttributes(
	opsinst *opsworks.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Depending on the timing, the state could be requested or stopped
		if *opsinst.Status != "stopped" && *opsinst.Status != "requested" {
			return fmt.Errorf("Unexpected request status: %s", *opsinst.Status)
		}
		if *opsinst.AvailabilityZone != "us-west-2a" {
			return fmt.Errorf("Unexpected availability zone: %s", *opsinst.AvailabilityZone)
		}
		if *opsinst.Architecture != "x86_64" {
			return fmt.Errorf("Unexpected architecture: %s", *opsinst.Architecture)
		}
		if *opsinst.Tenancy != "default" {
			return fmt.Errorf("Unexpected tenancy: %s", *opsinst.Tenancy)
		}
		if *opsinst.InfrastructureClass != "ec2" {
			return fmt.Errorf("Unexpected infrastructure class: %s", *opsinst.InfrastructureClass)
		}
		if *opsinst.RootDeviceType != "ebs" {
			return fmt.Errorf("Unexpected root device type: %s", *opsinst.RootDeviceType)
		}
		if *opsinst.VirtualizationType != "hvm" {
			return fmt.Errorf("Unexpected virtualization type: %s", *opsinst.VirtualizationType)
		}
		return nil
	}
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

func testAccAwsOpsworksInstanceConfigUpdateHostName(name string) string {
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
  hostname = "tf-acc2"
}

%s

`, name, name, testAccAwsOpsworksStackConfigVpcCreate(name))
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
