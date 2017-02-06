package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDBEventSubscription_importBasic(t *testing.T) {
	resourceName := "aws_db_event_subscription.bar"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDBEventSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSDBEventSubscriptionConfig(rInt),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
