package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func testAccAWSSpotDatafeedSubscription_importBasic(t *testing.T) {
	resourceName := "aws_spot_datafeed_subscription.default"
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotDatafeedSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSpotDatafeedSubscription(ri),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
