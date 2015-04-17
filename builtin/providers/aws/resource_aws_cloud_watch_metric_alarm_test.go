package aws

//import(
//    "fmt"
//    "math/rand"
//    "strings"
//    "testing"
//    "time"
//
//    "github.com/awslabs/aws-sdk-go/aws"
//    "github.com/awslabs/aws-sdk-go/service/autoscaling"
//    "github.com/hashicorp/terraform/help/resource"
//    "github.com/hashicorp/terraform/terraform"
//)

func TestAccAWSCloudWatchMetricAlarm_basic(t* testing.T) {
    var alarm cloudwatch.MetricAlarm

    resource.Test(t, resource.TestCase{
        PreCheck:       func () { testAccPreCheck(t) },
        Providers:      testAccProviders,
        CheckDestroy:   testAccCheckAWSCloudWatchMetricAlarmDestroy,
        Steps:          []resource.TestStep{
            resource.TestStep{
                Config: testAccAWSCloudWatchMetricAlarmConfig,
                Check:  resource.ComposeTestCheckFunc(
                    testAccCheckCloudWatchMetricAlarmExists("aws_cloudwatch_metric_alarm.foobar", &alarm),
                    resource.TestCheckResourceAttr("aws_cloudwatch_metric_alarm.foobar", "", ""),
                    resource.TestCheckResourceAttr("aws_cloudwatch_metric_alarm.foobar", "", ""),
                ),
            },
        },
    })
}

func testAccCheckCloudWatchMetricAlarmExists(n string, alarm *cloudwatch.MetricAlarm) resource.TestCheckFunc {
    return func(s *terraform.State) error {
        rs, ok := s.RootModule().Resources[n]
        if !ok {
            return fmt.Errorf("Not found: %s", n)
        }

        conn := testAccProvider.Meta().(*AWSClient).cloudwatchconn
        params := cloudwatch.DeleteAlarmsInput{
            AlarmNames: []*string{aws.String(rs.Primary.ID)}
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
    return nil
}

var testAccAWSCloudWatchMetricAlarmConfig = fmt.Sprintf(`
resource "aws_cloudwatch_metric_alarm" "foobar" {
    alarm_name = "terraform-test-foobar5"
    comparison_operator = "GreaterThanOrEqualToThreshold"
    evaluation_periods = "2"
    metric_name = "CPUUtilization"
    namespace = "AWS/EC2"
    period = "120"
    statistic = "Average"
    threshold = "80"
    alarm_description = "This metric monitor ec2 cpu utilization"
    insufficient_data_actions = []
}
`)
