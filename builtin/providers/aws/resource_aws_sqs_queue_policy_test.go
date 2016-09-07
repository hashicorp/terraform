package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSSQSQueuePolicy_basic(t *testing.T) {
	queueName := fmt.Sprintf("sqs-queue-%s", acctest.RandString(5))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSQSQueueDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSQSPolicyConfig_basic(queueName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSQSExistsWithDefaults("aws_sqs_queue.q"),
					resource.TestMatchResourceAttr("aws_sqs_queue_policy.test", "policy",
						regexp.MustCompile("^{\"Version\":\"2012-10-17\".+")),
				),
			},
		},
	})
}

func testAccAWSSQSPolicyConfig_basic(r string) string {
	return fmt.Sprintf(testAccAWSSQSPolicyConfig_basic_tpl, r)
}

const testAccAWSSQSPolicyConfig_basic_tpl = `
resource "aws_sqs_queue" "q" {
  name = "%s"
}

resource "aws_sqs_queue_policy" "test" {
  queue_url = "${aws_sqs_queue.q.id}"
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "sqspolicy",
  "Statement": [
    {
      "Sid": "First",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sqs:SendMessage",
      "Resource": "${aws_sqs_queue.q.arn}",
      "Condition": {
        "ArnEquals": {
          "aws:SourceArn": "${aws_sqs_queue.q.arn}"
        }
      }
    }
  ]
}
POLICY
}
`
