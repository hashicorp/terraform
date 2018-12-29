package aws

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lightsail"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSLightsailStaticIpAttachment_basic(t *testing.T) {
	var staticIp lightsail.StaticIp
	staticIpName := fmt.Sprintf("tf-test-lightsail-%s", acctest.RandString(5))
	instanceName := fmt.Sprintf("tf-test-lightsail-%s", acctest.RandString(5))
	keypairName := fmt.Sprintf("tf-test-lightsail-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailStaticIpAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailStaticIpAttachmentConfig_basic(staticIpName, instanceName, keypairName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailStaticIpAttachmentExists("aws_lightsail_static_ip_attachment.test", &staticIp),
				),
			},
		},
	})
}

func TestAccAWSLightsailStaticIpAttachment_disappears(t *testing.T) {
	var staticIp lightsail.StaticIp
	staticIpName := fmt.Sprintf("tf-test-lightsail-%s", acctest.RandString(5))
	instanceName := fmt.Sprintf("tf-test-lightsail-%s", acctest.RandString(5))
	keypairName := fmt.Sprintf("tf-test-lightsail-%s", acctest.RandString(5))

	staticIpDestroy := func(*terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).lightsailconn
		_, err := conn.DetachStaticIp(&lightsail.DetachStaticIpInput{
			StaticIpName: aws.String(staticIpName),
		})

		if err != nil {
			return fmt.Errorf("Error deleting Lightsail Static IP in disappear test")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSLightsailStaticIpAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSLightsailStaticIpAttachmentConfig_basic(staticIpName, instanceName, keypairName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSLightsailStaticIpAttachmentExists("aws_lightsail_static_ip_attachment.test", &staticIp),
					staticIpDestroy,
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSLightsailStaticIpAttachmentExists(n string, staticIp *lightsail.StaticIp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No Lightsail Static IP Attachment ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		resp, err := conn.GetStaticIp(&lightsail.GetStaticIpInput{
			StaticIpName: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}

		if resp == nil || resp.StaticIp == nil {
			return fmt.Errorf("Static IP (%s) not found", rs.Primary.ID)
		}

		if !*resp.StaticIp.IsAttached {
			return fmt.Errorf("Static IP (%s) not attached", rs.Primary.ID)
		}

		*staticIp = *resp.StaticIp
		return nil
	}
}

func testAccCheckAWSLightsailStaticIpAttachmentDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_lightsail_static_ip_attachment" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).lightsailconn

		resp, err := conn.GetStaticIp(&lightsail.GetStaticIpInput{
			StaticIpName: aws.String(rs.Primary.ID),
		})

		if err == nil {
			if *resp.StaticIp.IsAttached {
				return fmt.Errorf("Lightsail Static IP %q is still attached (to %q)", rs.Primary.ID, *resp.StaticIp.AttachedTo)
			}
		}

		// Verify the error
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFoundException" {
				return nil
			}
		}
		return err
	}

	return nil
}

func testAccAWSLightsailStaticIpAttachmentConfig_basic(staticIpName, instanceName, keypairName string) string {
	return fmt.Sprintf(`
provider "aws" {
  region = "us-east-1"
}

resource "aws_lightsail_static_ip_attachment" "test" {
  static_ip_name = "${aws_lightsail_static_ip.test.name}"
  instance_name = "${aws_lightsail_instance.test.name}"
}

resource "aws_lightsail_static_ip" "test" {
  name = "%s"
}

resource "aws_lightsail_instance" "test" {
  name              = "%s"
  availability_zone = "us-east-1b"
  blueprint_id      = "wordpress_4_6_1"
  bundle_id         = "micro_1_0"
  key_pair_name     = "${aws_lightsail_key_pair.test.name}"
}

resource "aws_lightsail_key_pair" "test" {
  name = "%s"
}
`, staticIpName, instanceName, keypairName)
}
