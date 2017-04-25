package rabbitmq

import (
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccQueue_importBasic(t *testing.T) {
	resourceName := "rabbitmq_queue.test"
	var queue rabbithole.QueueInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccQueueCheckDestroy(&queue),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccQueueConfig_basic,
				Check: testAccQueueCheck(
					resourceName, &queue,
				),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
