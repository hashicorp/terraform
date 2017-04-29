package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSSMMaintenanceWindow_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMMaintenanceWindowBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMMaintenanceWindowExists("aws_ssm_maintenance_window.foo"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "schedule", "cron(0 16 ? * TUE *)"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "duration", "3"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "cutoff", "1"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "name", fmt.Sprintf("maintenance-window-%s", name)),
				),
			},
			{
				Config: testAccAWSSSMMaintenanceWindowBasicConfigUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMMaintenanceWindowExists("aws_ssm_maintenance_window.foo"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "schedule", "cron(0 16 ? * WED *)"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "duration", "10"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "cutoff", "8"),
					resource.TestCheckResourceAttr(
						"aws_ssm_maintenance_window.foo", "name", fmt.Sprintf("updated-maintenance-window-%s", name)),
				),
			},
		},
	})
}

func testAccCheckAWSSSMMaintenanceWindowExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Maintenance Window ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		resp, err := conn.DescribeMaintenanceWindows(&ssm.DescribeMaintenanceWindowsInput{
			Filters: []*ssm.MaintenanceWindowFilter{
				{
					Key:    aws.String("Name"),
					Values: []*string{aws.String(rs.Primary.Attributes["name"])},
				},
			},
		})

		for _, i := range resp.WindowIdentities {
			if *i.WindowId == rs.Primary.ID {
				return nil
			}
		}
		if err != nil {
			return err
		}

		return fmt.Errorf("No AWS SSM Maintenance window found")
	}
}

func testAccCheckAWSSSMMaintenanceWindowDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_maintenance_window" {
			continue
		}

		out, err := conn.DescribeMaintenanceWindows(&ssm.DescribeMaintenanceWindowsInput{
			Filters: []*ssm.MaintenanceWindowFilter{
				{
					Key:    aws.String("Name"),
					Values: []*string{aws.String(rs.Primary.Attributes["name"])},
				},
			},
		})

		if err != nil {
			return err
		}

		if len(out.WindowIdentities) > 0 {
			return fmt.Errorf("Expected AWS SSM Maintenance Document to be gone, but was still found")
		}

		return nil
	}

	return nil
}

func testAccAWSSSMMaintenanceWindowBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_maintenance_window" "foo" {
  name = "maintenance-window-%s"
  schedule = "cron(0 16 ? * TUE *)"
  duration = 3
  cutoff = 1
}

`, rName)
}

func testAccAWSSSMMaintenanceWindowBasicConfigUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_maintenance_window" "foo" {
  name = "updated-maintenance-window-%s"
  schedule = "cron(0 16 ? * WED *)"
  duration = 10
  cutoff = 8
}

`, rName)
}
