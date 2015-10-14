package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSVolumeAttachment_basic(t *testing.T) {
	var i ec2.Instance
	var v ec2.Volume

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVolumeAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccVolumeAttachmentConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_volume_attachment.ebs_att", "device_name", "/dev/sdh"),
					testAccCheckInstanceExists(
						"aws_instance.web", &i),
					testAccCheckVolumeExists(
						"aws_ebs_volume.example", &v),
					testAccCheckVolumeAttachmentExists(
						"aws_volume_attachment.ebs_att", &i, &v),
				),
			},
		},
	})
}

func testAccCheckVolumeAttachmentExists(n string, i *ec2.Instance, v *ec2.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		for _, b := range i.BlockDeviceMappings {
			if rs.Primary.Attributes["device_name"] == *b.DeviceName {
				if b.Ebs.VolumeId != nil && rs.Primary.Attributes["volume_id"] == *b.Ebs.VolumeId {
					// pass
					return nil
				}
			}
		}

		return fmt.Errorf("Error finding instance/volume")
	}
}

func testAccCheckVolumeAttachmentDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		log.Printf("\n\n----- This is never called")
		if rs.Type != "aws_volume_attachment" {
			continue
		}
	}
	return nil
}

const testAccVolumeAttachmentConfig = `
resource "aws_instance" "web" {
	ami = "ami-21f78e11"
  availability_zone = "us-west-2a"
	instance_type = "t1.micro"
	tags {
		Name = "HelloWorld"
	}
}

resource "aws_ebs_volume" "example" {
  availability_zone = "us-west-2a"
	size = 1
}

resource "aws_volume_attachment" "ebs_att" {
  device_name = "/dev/sdh"
	volume_id = "${aws_ebs_volume.example.id}"
	instance_id = "${aws_instance.web.id}"
}
`
