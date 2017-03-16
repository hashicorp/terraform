package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsAutoscalingAttachment_basic(t *testing.T) {

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAutoscalingAttachment_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingAttachmentExists("aws_autoscaling_group.asg", 0),
				),
			},
			// Add in one association
			resource.TestStep{
				Config: testAccAWSAutoscalingAttachment_associated(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingAttachmentExists("aws_autoscaling_group.asg", 1),
				),
			},
			// Test adding a 2nd
			resource.TestStep{
				Config: testAccAWSAutoscalingAttachment_double_associated(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingAttachmentExists("aws_autoscaling_group.asg", 2),
				),
			},
			// Now remove that newest one
			resource.TestStep{
				Config: testAccAWSAutoscalingAttachment_associated(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingAttachmentExists("aws_autoscaling_group.asg", 1),
				),
			},
			// Now remove them both
			resource.TestStep{
				Config: testAccAWSAutoscalingAttachment_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSAutocalingAttachmentExists("aws_autoscaling_group.asg", 0),
				),
			},
		},
	})
}

func testAccCheckAWSAutocalingAttachmentExists(asgname string, loadBalancerCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[asgname]
		if !ok {
			return fmt.Errorf("Not found: %s", asgname)
		}

		conn := testAccProvider.Meta().(*AWSClient).autoscalingconn
		asg := rs.Primary.ID

		actual, err := conn.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{aws.String(asg)},
		})

		if err != nil {
			return fmt.Errorf("Recieved an error when attempting to load %s:  %s", asg, err)
		}

		if loadBalancerCount != len(actual.AutoScalingGroups[0].LoadBalancerNames) {
			return fmt.Errorf("Error: ASG has the wrong number of load balacners associated.  Expected [%d] but got [%d]", loadBalancerCount, len(actual.AutoScalingGroups[0].LoadBalancerNames))
		}

		return nil
	}
}

func testAccAWSAutoscalingAttachment_basic(rInt int) string {
	return fmt.Sprintf(`
resource "aws_elb" "foo" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port     = 8000
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }
}

resource "aws_elb" "bar" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]

  listener {
    instance_port     = 8000
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }
}

resource "aws_launch_configuration" "as_conf" {
    name = "test_config_%d"
    image_id = "ami-f34032c3"
    instance_type = "t1.micro"
}

resource "aws_autoscaling_group" "asg" {
  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  name = "asg-lb-assoc-terraform-test_%d"
  max_size = 1
  min_size = 0
  desired_capacity = 0
  health_check_grace_period = 300
  force_delete = true
  launch_configuration = "${aws_launch_configuration.as_conf.name}"

  tag {
    key = "Name"
    value = "terraform-asg-lg-assoc-test"
    propagate_at_launch = true
  }
}`, rInt, rInt)
}

func testAccAWSAutoscalingAttachment_associated(rInt int) string {
	return testAccAWSAutoscalingAttachment_basic(rInt) + `
resource "aws_autoscaling_attachment" "asg_attachment_foo" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  elb                    = "${aws_elb.foo.id}"
}`
}

func testAccAWSAutoscalingAttachment_double_associated(rInt int) string {
	return testAccAWSAutoscalingAttachment_associated(rInt) + `
resource "aws_autoscaling_attachment" "asg_attachment_bar" {
  autoscaling_group_name = "${aws_autoscaling_group.asg.id}"
  elb                    = "${aws_elb.bar.id}"
}`
}
