package rabbitmq

import (
	"fmt"
	"strings"
	"testing"

	"github.com/michaelklishin/rabbit-hole"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccQueue(t *testing.T) {
	var queueInfo rabbithole.QueueInfo
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccQueueCheckDestroy(&queueInfo),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccQueueConfig_basic,
				Check: testAccQueueCheck(
					"rabbitmq_queue.test", &queueInfo,
				),
			},
		},
	})
}

func testAccQueueCheck(rn string, queueInfo *rabbithole.QueueInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("queue id not set")
		}

		rmqc := testAccProvider.Meta().(*rabbithole.Client)
		queueParts := strings.Split(rs.Primary.ID, "@")

		queues, err := rmqc.ListQueuesIn(queueParts[1])
		if err != nil {
			return fmt.Errorf("Error retrieving queue: %s", err)
		}

		for _, queue := range queues {
			if queue.Name == queueParts[0] && queue.Vhost == queueParts[1] {
				queueInfo = &queue
				return nil
			}
		}

		return fmt.Errorf("Unable to find queue %s", rn)
	}
}

func testAccQueueCheckDestroy(queueInfo *rabbithole.QueueInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rmqc := testAccProvider.Meta().(*rabbithole.Client)

		queues, err := rmqc.ListQueuesIn(queueInfo.Vhost)
		if err != nil {
			return fmt.Errorf("Error retrieving queue: %s", err)
		}

		for _, queue := range queues {
			if queue.Name == queueInfo.Name && queue.Vhost == queueInfo.Vhost {
				return fmt.Errorf("Queue %s@%s still exist", queueInfo.Name, queueInfo.Vhost)
			}
		}

		return nil
	}
}

const testAccQueueConfig_basic = `
resource "rabbitmq_vhost" "test" {
    name = "test"
}

resource "rabbitmq_permissions" "guest" {
    user = "guest"
    vhost = "${rabbitmq_vhost.test.name}"
    permissions {
        configure = ".*"
        write = ".*"
        read = ".*"
    }
}

resource "rabbitmq_queue" "test" {
    name = "test"
    vhost = "${rabbitmq_permissions.guest.vhost}"
    settings {
        durable = false
        auto_delete = true
    }
}`
