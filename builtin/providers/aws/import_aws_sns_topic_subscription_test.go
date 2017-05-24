package aws

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccAWSSNSTopicSubscription_importBasic(t *testing.T) {
	resourceName := "aws_sns_topic.test_topic"
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSTopicSubscriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSNSTopicSubscriptionConfig(ri),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
