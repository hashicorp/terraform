package aws

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSSNSTopicPolicy_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSNSTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSNSTopicConfig_withPolicy,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSNSTopicExists("aws_sns_topic.test"),
					resource.TestMatchResourceAttr("aws_sns_topic_policy.custom", "policy",
						regexp.MustCompile("^{\"Version\":\"2012-10-17\".+")),
				),
			},
		},
	})
}

const testAccAWSSNSTopicConfig_withPolicy = `
resource "aws_sns_topic" "test" {
    name = "tf-acc-test-topic-with-policy"
}

resource "aws_sns_topic_policy" "custom" {
	arn = "${aws_sns_topic.test.arn}"
	policy = <<POLICY
{
   "Version":"2012-10-17",
   "Id": "default",
   "Statement":[{
   	"Sid":"default",
   	"Effect":"Allow",
   	"Principal":{"AWS":"*"},
   	"Action":["SNS:GetTopicAttributes","SNS:SetTopicAttributes","SNS:AddPermission","SNS:RemovePermission","SNS:DeleteTopic"],
   	"Resource":"${aws_sns_topic.test.arn}"
  }]
}
POLICY
}
`
