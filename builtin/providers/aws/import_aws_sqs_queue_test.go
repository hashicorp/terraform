package aws

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSSQSQueue_importBasic(t *testing.T) {
	resourceName := "aws_sqs_queue.queue"
	queueName := fmt.Sprintf("sqs-queue-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSQSQueueDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSQSConfigWithDefaults(queueName),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("aws_sqs_queue.queue", "fifo_queue", "false"),
				),
			},
		},
	})
}

func TestAccAWSSQSQueue_importFifo(t *testing.T) {
	resourceName := "aws_sqs_queue.queue"
	queueName := fmt.Sprintf("sqs-queue-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSQSQueueDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSQSFifoConfigWithDefaults(queueName),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("aws_sqs_queue.queue", "fifo_queue", "true"),
				),
			},
		},
	})
}
