package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/devicefarm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDeviceFarmProject_basic(t *testing.T) {
	var afterCreate, afterUpdate devicefarm.Project
	beforeInt := acctest.RandInt()
	afterInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDeviceFarmProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceFarmProjectConfig(beforeInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDeviceFarmProjectExists(
						"aws_devicefarm_project.foo", &afterCreate),
					resource.TestCheckResourceAttr(
						"aws_devicefarm_project.foo", "name", fmt.Sprintf("tf-testproject-%d", beforeInt)),
				),
			},

			{
				Config: testAccDeviceFarmProjectConfig(afterInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDeviceFarmProjectExists(
						"aws_devicefarm_project.foo", &afterUpdate),
					resource.TestCheckResourceAttr(
						"aws_devicefarm_project.foo", "name", fmt.Sprintf("tf-testproject-%d", afterInt)),
					testAccCheckDeviceFarmProjectNotRecreated(
						t, &afterCreate, &afterUpdate),
				),
			},
		},
	})
}

func testAccCheckDeviceFarmProjectNotRecreated(t *testing.T,
	before, after *devicefarm.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *before.Arn != *after.Arn {
			t.Fatalf("Expected DeviceFarm Project ARNs to be the same. But they were: %v, %v", *before.Arn, *after.Arn)
		}
		return nil
	}
}

func testAccCheckDeviceFarmProjectExists(n string, v *devicefarm.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).devicefarmconn
		resp, err := conn.GetProject(
			&devicefarm.GetProjectInput{Arn: aws.String(rs.Primary.ID)})
		if err != nil {
			return err
		}
		if resp.Project == nil {
			return fmt.Errorf("DeviceFarmProject not found")
		}

		*v = *resp.Project

		return nil
	}
}

func testAccCheckDeviceFarmProjectDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).devicefarmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_devicefarm_project" {
			continue
		}

		// Try to find the resource
		resp, err := conn.GetProject(
			&devicefarm.GetProjectInput{Arn: aws.String(rs.Primary.ID)})
		if err == nil {
			if resp.Project != nil {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		if dferr, ok := err.(awserr.Error); ok && dferr.Code() == "DeviceFarmProjectNotFoundFault" {
			return nil
		}
	}

	return nil
}

func testAccDeviceFarmProjectConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_devicefarm_project" "foo" {
	name = "tf-testproject-%d"
}`, rInt)
}
