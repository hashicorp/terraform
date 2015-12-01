package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/devicefarm"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDeviceFarmdevicePool_basic(t *testing.T) {
	var v devicefarm.DevicePool

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDeviceFarmDevicePoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDeviceFarmProjectConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDeviceFarmDevicePoolExists(
						"aws_devicefarm_devicepool.foo", &v),
				),
			},
		},
	})
}

func TestAccAWSDeviceFarmDevicePool_update(t *testing.T) {
	var afterCreate, afterUpdate devicefarm.DevicePool

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDeviceFarmDevicePoolDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDeviceFarmDevicePoolConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDeviceFarmDevicePoolExists(
						"aws_devicefarm_devicepool.foo", &afterCreate),
					resource.TestCheckResourceAttr(
						"aws_devicefarm_devicepool.foo", "description", "TestDescription"),
				),
			},

			resource.TestStep{
				Config: testAccDeviceFarmDevicePoolConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDeviceFarmDevicePoolExists(
						"aws_devicefarm_devicepool.foo", &afterUpdate),
					resource.TestCheckResourceAttr(
						"aws_devicefarm_devicepool.foo", "description", "TestDescriptionUpdated"),
					testAccCheckDeviceFarmDevicePoolNotRecreated(
						t, &afterCreate, &afterUpdate),
				),
			},
		},
	})
}

func testAccCheckDeviceFarmDevicePoolNotRecreated(t *testing.T,
	before, after *devicefarm.DevicePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.Arn != *after.Arn {
			t.Fatalf("Expected DeviceFarm DevicePool ARNs to be the same. But they were: %v, %v", *before.Arn, *after.Arn)
		}
		return nil
	}
}

func testAccCheckDeviceFarmDevicePoolExists(n string, v *devicefarm.DevicePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).devicefarmconn
		resp, err := conn.GetDevicePool(
			&devicefarm.GetDevicePoolInput{Arn: aws.String(rs.Primary.ID)})
		if err != nil {
			return err
		}
		if resp.DevicePool == nil {
			return fmt.Errorf("DeviceFarmDevicePool not found")
		}

		*v = *resp.DevicePool

		return nil
	}
}

func testAccCheckDeviceFarmDevicePoolDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).devicefarmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_devicefarm_devicepool" {
			continue
		}

		// Try to find the resource
		resp, err := conn.GetDevicePool(
			&devicefarm.GetDevicePoolInput{Arn: aws.String(rs.Primary.ID)})
		if err == nil {
			if resp.DevicePool != nil {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		// Verify the error is what we want
		dferr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if dferr.Code() != "DeviceFarmDevicePoolNotFoundFault" {
			return err
		}
	}

	return nil
}

const testAccDeviceFarmDevicePoolConfig = `
provider "aws" {
	region = "us-west-2"
}

resource "aws_devicefarm_project" "foo" {
	name = "tf-testproject-01"
}

resource "aws_devicefarm_devicepool" "foo" {
    name = "MyDevicePool"
    description = "TestDescription"
    project_arn = "${aws_devicefarm_project.foo.arn}"

    rules {
    	attribute = "PLATFORM"
    	operator = "EQUALS"
    	value = "IOS"
  	}
}
`

const testAccDeviceFarmDevicePoolConfigUpdate = `
provider "aws" {
	region = "us-west-2"
}

resource "aws_devicefarm_project" "foo" {
	name = "tf-testproject-01"
}

resource "aws_devicefarm_devicepool" "foo" {
    name = "MyDevicePool"
    description = "TestDescriptionUpdated"
    project_arn = "${aws_devicefarm_project.foo.arn}"

    rules {
    	attribute = "PLATFORM"
    	operator = "EQUALS"
    	value = "IOS"
  	}
}
`
