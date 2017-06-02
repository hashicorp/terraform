package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSFlowLog_basic(t *testing.T) {
	var flowLog ec2.FlowLog

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_flow_log.test_flow_log",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFlowLogConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists("aws_flow_log.test_flow_log", &flowLog),
					testAccCheckAWSFlowLogAttributes(&flowLog),
				),
			},
		},
	})
}

func TestAccAWSFlowLog_subnet(t *testing.T) {
	var flowLog ec2.FlowLog

	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "aws_flow_log.test_flow_log_subnet",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckFlowLogDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFlowLogConfig_subnet(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlowLogExists("aws_flow_log.test_flow_log_subnet", &flowLog),
					testAccCheckAWSFlowLogAttributes(&flowLog),
				),
			},
		},
	})
}

func testAccCheckFlowLogExists(n string, flowLog *ec2.FlowLog) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Flow Log ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		describeOpts := &ec2.DescribeFlowLogsInput{
			FlowLogIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeFlowLogs(describeOpts)
		if err != nil {
			return err
		}

		if len(resp.FlowLogs) > 0 {
			*flowLog = *resp.FlowLogs[0]
			return nil
		}
		return fmt.Errorf("No Flow Logs found for id (%s)", rs.Primary.ID)
	}
}

func testAccCheckAWSFlowLogAttributes(flowLog *ec2.FlowLog) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if flowLog.FlowLogStatus != nil && *flowLog.FlowLogStatus == "ACTIVE" {
			return nil
		}
		if flowLog.FlowLogStatus == nil {
			return fmt.Errorf("Flow Log status is not ACTIVE, is nil")
		} else {
			return fmt.Errorf("Flow Log status is not ACTIVE, got: %s", *flowLog.FlowLogStatus)
		}
	}
}

func testAccCheckFlowLogDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_flow_log" {
			continue
		}

		return nil
	}

	return nil
}

func testAccFlowLogConfig_basic(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "default" {
        cidr_block = "10.0.0.0/16"
        tags {
                Name = "tf-flow-log-test"
        }
}

resource "aws_subnet" "test_subnet" {
        vpc_id = "${aws_vpc.default.id}"
        cidr_block = "10.0.1.0/24"

        tags {
                Name = "tf-flow-test"
        }
}

resource "aws_iam_role" "test_role" {
    name = "tf_test_flow_log_basic_%d"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.amazonaws.com"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}

resource "aws_cloudwatch_log_group" "foobar" {
    name = "tf-test-fl-%d"
}
resource "aws_flow_log" "test_flow_log" {
        log_group_name = "${aws_cloudwatch_log_group.foobar.name}"
        iam_role_arn = "${aws_iam_role.test_role.arn}"
        vpc_id = "${aws_vpc.default.id}"
        traffic_type = "ALL"
}

resource "aws_flow_log" "test_flow_log_subnet" {
        log_group_name = "${aws_cloudwatch_log_group.foobar.name}"
        iam_role_arn = "${aws_iam_role.test_role.arn}"
        subnet_id = "${aws_subnet.test_subnet.id}"
        traffic_type = "ALL"
}
`, rInt, rInt)
}

func testAccFlowLogConfig_subnet(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "default" {
        cidr_block = "10.0.0.0/16"
        tags {
                Name = "tf-flow-log-test"
        }
}

resource "aws_subnet" "test_subnet" {
        vpc_id = "${aws_vpc.default.id}"
        cidr_block = "10.0.1.0/24"

        tags {
                Name = "tf-flow-test"
        }
}

resource "aws_iam_role" "test_role" {
    name = "tf_test_flow_log_subnet_%d"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "ec2.amazonaws.com"
        ]
      },
      "Action": [
        "sts:AssumeRole"
      ]
    }
  ]
}
EOF
}
resource "aws_cloudwatch_log_group" "foobar" {
    name = "tf-test-fl-%d"
}

resource "aws_flow_log" "test_flow_log_subnet" {
        log_group_name = "${aws_cloudwatch_log_group.foobar.name}"
        iam_role_arn = "${aws_iam_role.test_role.arn}"
        subnet_id = "${aws_subnet.test_subnet.id}"
        traffic_type = "ALL"
}
`, rInt, rInt)
}
