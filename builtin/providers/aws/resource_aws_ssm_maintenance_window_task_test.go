package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSSMMaintenanceWindowTask_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMMaintenanceWindowTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMMaintenanceWindowTaskBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMMaintenanceWindowTaskExists("aws_ssm_maintenance_window_task.target"),
				),
			},
		},
	})
}

func testAccCheckAWSSSMMaintenanceWindowTaskExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Maintenance Window Task Window ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		resp, err := conn.DescribeMaintenanceWindowTasks(&ssm.DescribeMaintenanceWindowTasksInput{
			WindowId: aws.String(rs.Primary.Attributes["window_id"]),
		})
		if err != nil {
			return err
		}

		for _, i := range resp.Tasks {
			if *i.WindowTaskId == rs.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf("No AWS SSM Maintenance window task found")
	}
}

func testAccCheckAWSSSMMaintenanceWindowTaskDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_maintenance_window_target" {
			continue
		}

		out, err := conn.DescribeMaintenanceWindowTasks(&ssm.DescribeMaintenanceWindowTasksInput{
			WindowId: aws.String(rs.Primary.Attributes["window_id"]),
		})

		if err != nil {
			// Verify the error is what we want
			if ae, ok := err.(awserr.Error); ok && ae.Code() == "DoesNotExistException" {
				continue
			}
			return err
		}

		if len(out.Tasks) > 0 {
			return fmt.Errorf("Expected AWS SSM Maintenance Task to be gone, but was still found")
		}

		return nil
	}

	return nil
}

func testAccAWSSSMMaintenanceWindowTaskBasicConfig(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_maintenance_window" "foo" {
  name = "maintenance-window-%s"
  schedule = "cron(0 16 ? * TUE *)"
  duration = 3
  cutoff = 1
}

resource "aws_ssm_maintenance_window_task" "target" {
  window_id = "${aws_ssm_maintenance_window.foo.id}"
  task_type = "RUN_COMMAND"
  task_arn = "AWS-RunShellScript"
  priority = 1
  service_role_arn = "${aws_iam_role.ssm_role.arn}"
  max_concurrency = "2"
  max_errors = "1"
  targets {
    key = "InstanceIds"
    values = ["${aws_instance.foo.id}"]
  }
  task_parameters {
    name = "commands"
    values = ["pwd"]
  }
}

resource "aws_instance" "foo" {
  ami = "ami-4fccb37f"

  instance_type = "m1.small"
}

resource "aws_iam_role" "ssm_role" {
  name = "ssm-role-%s"

  assume_role_policy = <<POLICY
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {
                "Service": "events.amazonaws.com"
            },
            "Effect": "Allow",
            "Sid": ""
        }
    ]
}
POLICY
}

resource "aws_iam_role_policy" "bar" {
  name = "ssm_role_policy_%s"
  role = "${aws_iam_role.ssm_role.name}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": {
    "Effect": "Allow",
    "Action": "ssm:*",
    "Resource": "*"
  }
}
EOF
}

`, rName, rName, rName)
}
