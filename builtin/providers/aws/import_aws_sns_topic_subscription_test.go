package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSSNSTopicSubscription_importBasic(t *testing.T) {
	resourceName := "aws_sns_topic.test_topic"
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSTopicSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSNSTopicSubscriptionConfig(ri),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
