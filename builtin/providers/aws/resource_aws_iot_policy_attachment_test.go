package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSIoTPolicyAttachment_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSIoTPolicyAttachmentDestroy_basic,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSIoTPolicyAttachment_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSIoTPolicyAttachmentExists_basic("aws_iot_policy_attachment.cert_policies"),
				),
			},
		},
	})
}

func testAccCheckAWSIoTPolicyAttachmentDestroy_basic(s *terraform.State) error {
	return nil
}

func testAccCheckAWSIoTPolicyAttachmentExists_basic(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s in %+v", name, s.RootModule().Resources)
		}

		return nil
	}
}

var testAccAWSIoTPolicyAttachment_basic = `
resource "aws_iot_certificate" "cert" {
	csr = "${file("test-fixtures/csr.pem")}"
  active = true
}

resource "aws_iot_policy" "pubsub" {
  name = "PubSubToAnyTopic"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["iot:*"],
    "Resource": ["*"]
  }]
}
EOF
}

resource "aws_iot_policy_attachment" "cert_policies" {
  name = "cert_policies"
  principal = "${aws_iot_certificate.cert.arn}"
  policies = ["${aws_iot_policy.pubsub.name}"]
}
`
