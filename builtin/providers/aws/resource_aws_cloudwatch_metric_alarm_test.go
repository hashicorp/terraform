package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudWatchMetricAlarm_basic(t *testing.T) {
	var alarm cloudwatch.MetricAlarm

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchMetricAlarmDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchMetricAlarmConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudWatchMetricAlarmExists("aws_cloudwatch_metric_alarm.foobar", &alarm),
					resource.TestCheckResourceAttr("aws_cloudwatch_metric_alarm.foobar", "metric_name", "CPUUtilization"),
					resource.TestCheckResourceAttr("aws_cloudwatch_metric_alarm.foobar", "statistic", "Average"),
					testAccCheckCloudWatchMetricAlarmDimension(
						"aws_cloudwatch_metric_alarm.foobar", "InstanceId", "i-abc123"),
				),
			},
		},
	})
}

func testAccCheckCloudWatchMetricAlarmDimension(n, k, v string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		key := fmt.Sprintf("dimensions.%s", k)
		val, ok := rs.Primary.Attributes[key]
		if !ok {
			return fmt.Errorf("Could not find dimension: %s", k)
		}
		if val != v {
			return fmt.Errorf("Expected dimension %s => %s; got: %s", k, v, val)
		}
		return nil
	}
}

func testAccCheckCloudWatchMetricAlarmExists(n string, alarm *cloudwatch.MetricAlarm) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudwatchconn
		params := cloudwatch.DescribeAlarmsInput{
			AlarmNames: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeAlarms(&params)
		if err != nil {
			return err
		}
		if len(resp.MetricAlarms) == 0 {
			return fmt.Errorf("Alarm not found")
		}
		*alarm = *resp.MetricAlarms[0]

		return nil
	}
}

func testAccCheckAWSCloudWatchMetricAlarmDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudwatchconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudwatch_metric_alarm" {
			continue
		}

		params := cloudwatch.DescribeAlarmsInput{
			AlarmNames: []*string{aws.String(rs.Primary.ID)},
		}

		resp, err := conn.DescribeAlarms(&params)

		if err == nil {
			if len(resp.MetricAlarms) != 0 &&
				*resp.MetricAlarms[0].AlarmName == rs.Primary.ID {
				return fmt.Errorf("Alarm Still Exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccAWSCloudWatchMetricAlarmConfig = fmt.Sprintf(`
resource "aws_cloudwatch_metric_alarm" "foobar" {
  alarm_name                = "terraform-test-foobar5"
  comparison_operator       = "GreaterThanOrEqualToThreshold"
  evaluation_periods        = "2"
  metric_name               = "CPUUtilization"
  namespace                 = "AWS/EC2"
  period                    = "120"
  statistic                 = "Average"
  threshold                 = "80"
  alarm_description         = "This metric monitors ec2 cpu utilization"
  insufficient_data_actions = []
  dimensions {
    InstanceId = "i-abc123"
  }
}
`)
