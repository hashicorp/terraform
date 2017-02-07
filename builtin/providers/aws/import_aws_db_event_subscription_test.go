package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDBEventSubscription_importBasic(t *testing.T) {
	resourceName := "aws_db_event_subscription.bar"
	rInt := acctest.RandInt()
	subscriptionName := fmt.Sprintf("tf-acc-test-rds-event-subs-%d", rInt)

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
				ImportStateId:     subscriptionName,
			},
		},
	})
}
