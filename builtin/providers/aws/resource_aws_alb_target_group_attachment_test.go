package aws

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"strconv"
	"testing"
)

func TestAccAWSALBTargetGroupAttachment_basic(t *testing.T) {
	targetGroupName := fmt.Sprintf("test-target-group-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_alb_target_group.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckAWSALBTargetGroupAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSALBTargetGroupAttachmentConfig_basic(targetGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSALBTargetGroupAttachmentExists("aws_alb_target_group_attachment.test"),
				),
			},
		},
	})
}

func testAccCheckAWSALBTargetGroupAttachmentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Target Group Attachment ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elbv2conn

		port, _ := strconv.Atoi(rs.Primary.Attributes["port"])
		describe, err := conn.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: aws.String(rs.Primary.Attributes["target_group_arn"]),
			Targets: []*elbv2.TargetDescription{
				{
					Id:   aws.String(rs.Primary.Attributes["target_id"]),
					Port: aws.Int64(int64(port)),
				},
			},
		})

		if err != nil {
			return err
		}

		if len(describe.TargetHealthDescriptions) != 1 {
			return errors.New("Target Group Attachment not found")
		}

		return nil
	}
}

func testAccCheckAWSALBTargetGroupAttachmentDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elbv2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_alb_target_group_attachment" {
			continue
		}

		port, _ := strconv.Atoi(rs.Primary.Attributes["port"])
		describe, err := conn.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
			TargetGroupArn: aws.String(rs.Primary.Attributes["target_group_arn"]),
			Targets: []*elbv2.TargetDescription{
				{
					Id:   aws.String(rs.Primary.Attributes["target_id"]),
					Port: aws.Int64(int64(port)),
				},
			},
		})
		if err == nil {
			if len(describe.TargetHealthDescriptions) != 0 {
				return fmt.Errorf("Target Group Attachment %q still exists", rs.Primary.ID)
			}
		}

		// Verify the error
		if isTargetGroupNotFound(err) || isInvalidTarget(err) {
			return nil
		} else {
			return errwrap.Wrapf("Unexpected error checking ALB destroyed: {{err}}", err)
		}
	}

	return nil
}

func testAccAWSALBTargetGroupAttachmentConfig_basic(targetGroupName string) string {
	return fmt.Sprintf(`
resource "aws_alb_target_group_attachment" "test" {
  target_group_arn = "${aws_alb_target_group.test.arn}"
  target_id = "${aws_instance.test.id}"
  port = 80
}

resource "aws_instance" "test" {
  ami = "ami-f701cb97"
  instance_type = "t2.micro"
  subnet_id = "${aws_subnet.subnet.id}"
}

resource "aws_alb_target_group" "test" {
  name = "%s"
  port = 443
  protocol = "HTTPS"
  vpc_id = "${aws_vpc.test.id}"

  deregistration_delay = 200

  stickiness {
    type = "lb_cookie"
    cookie_duration = 10000
  }

  health_check {
    path = "/health"
    interval = 60
    port = 8081
    protocol = "HTTP"
    timeout = 3
    healthy_threshold = 3
    unhealthy_threshold = 3
    matcher = "200-299"
  }
}

resource "aws_subnet" "subnet" {
  cidr_block = "10.0.1.0/24"
  vpc_id = "${aws_vpc.test.id}"

}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
}`, targetGroupName)
}
