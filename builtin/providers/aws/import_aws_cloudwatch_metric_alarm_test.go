package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
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
