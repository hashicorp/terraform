package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCloudWatchMetricAlarm_importBasic(t *testing.T) {
	resourceName := "aws_cloudwatch_metric_alarm.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchMetricAlarmDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudWatchMetricAlarmConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
