package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSCloudWatchMetricAlarm_importBasic(t *testing.T) {
	rInt := acctest.RandInt()
	resourceName := "aws_cloudwatch_metric_alarm.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCloudWatchMetricAlarmDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudWatchMetricAlarmConfig(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
