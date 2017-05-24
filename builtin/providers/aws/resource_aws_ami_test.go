package aws

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAMI_basic(t *testing.T) {
	var ami ec2.Image
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAmiDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAmiConfig_basic(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAmiExists("aws_ami.foo", &ami),
					resource.TestCheckResourceAttr(
						"aws_ami.foo", "name", fmt.Sprintf("tf-testing-%d", rInt)),
				),
			},
		},
	})
}

func TestAccAWSAMI_snapshotSize(t *testing.T) {
	var ami ec2.Image
	var bd ec2.BlockDeviceMapping
	rInt := acctest.RandInt()

	expectedDevice := &ec2.EbsBlockDevice{
		DeleteOnTermination: aws.Bool(true),
		Encrypted:           aws.Bool(false),
		Iops:                aws.Int64(0),
		VolumeSize:          aws.Int64(20),
		VolumeType:          aws.String("standard"),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAmiDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAmiConfig_snapshotSize(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAmiExists("aws_ami.foo", &ami),
					testAccCheckAmiBlockDevice(&ami, &bd, "/dev/sda1"),
					testAccCheckAmiEbsBlockDevice(&bd, expectedDevice),
					resource.TestCheckResourceAttr(
						"aws_ami.foo", "name", fmt.Sprintf("tf-testing-%d", rInt)),
					resource.TestCheckResourceAttr(
						"aws_ami.foo", "architecture", "x86_64"),
				),
			},
		},
	})
}

func testAccCheckAmiDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ami" {
			continue
		}

		// Try to find the AMI
		log.Printf("AMI-ID: %s", rs.Primary.ID)
		DescribeAmiOpts := &ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeImages(DescribeAmiOpts)
		if err != nil {
			if isAWSErr(err, "InvalidAMIID", "NotFound") {
				log.Printf("[DEBUG] AMI not found, passing")
				return nil
			}
			return err
		}

		if len(resp.Images) > 0 {
			state := resp.Images[0].State
			return fmt.Errorf("AMI %s still exists in the state: %s.", *resp.Images[0].ImageId, *state)
		}
	}
	return nil
}

func testAccCheckAmiExists(n string, ami *ec2.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("AMI Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No AMI ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		opts := &ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String(rs.Primary.ID)},
		}
		resp, err := conn.DescribeImages(opts)
		if err != nil {
			return err
		}
		if len(resp.Images) == 0 {
			return fmt.Errorf("AMI not found")
		}
		*ami = *resp.Images[0]
		return nil
	}
}

func testAccCheckAmiBlockDevice(ami *ec2.Image, blockDevice *ec2.BlockDeviceMapping, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		devices := make(map[string]*ec2.BlockDeviceMapping)
		for _, device := range ami.BlockDeviceMappings {
			devices[*device.DeviceName] = device
		}

		// Check if the block device exists
		if _, ok := devices[n]; !ok {
			return fmt.Errorf("block device doesn't exist: %s", n)
		}

		*blockDevice = *devices[n]
		return nil
	}
}

func testAccCheckAmiEbsBlockDevice(bd *ec2.BlockDeviceMapping, ed *ec2.EbsBlockDevice) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Test for things that ed has, don't care about unset values
		cd := bd.Ebs
		if ed.VolumeType != nil {
			if *ed.VolumeType != *cd.VolumeType {
				return fmt.Errorf("Volume type mismatch. Expected: %s Got: %s",
					*ed.VolumeType, *cd.VolumeType)
			}
		}
		if ed.DeleteOnTermination != nil {
			if *ed.DeleteOnTermination != *cd.DeleteOnTermination {
				return fmt.Errorf("DeleteOnTermination mismatch. Expected: %t Got: %t",
					*ed.DeleteOnTermination, *cd.DeleteOnTermination)
			}
		}
		if ed.Encrypted != nil {
			if *ed.Encrypted != *cd.Encrypted {
				return fmt.Errorf("Encrypted mismatch. Expected: %t Got: %t",
					*ed.Encrypted, *cd.Encrypted)
			}
		}
		// Integer defaults need to not be `0` so we don't get a panic
		if ed.Iops != nil && *ed.Iops != 0 {
			if *ed.Iops != *cd.Iops {
				return fmt.Errorf("IOPS mismatch. Expected: %d Got: %d",
					*ed.Iops, *cd.Iops)
			}
		}
		if ed.VolumeSize != nil && *ed.VolumeSize != 0 {
			if *ed.VolumeSize != *cd.VolumeSize {
				return fmt.Errorf("Volume Size mismatch. Expected: %d Got: %d",
					*ed.VolumeSize, *cd.VolumeSize)
			}
		}

		return nil
	}
}

func testAccAmiConfig_basic(rInt int) string {
	return fmt.Sprintf(`
resource "aws_ebs_volume" "foo" {
 	availability_zone = "us-west-2a"
 	size = 8
 	tags {
 	  Name = "testAccAmiConfig_basic"
 	}
}

resource "aws_ebs_snapshot" "foo" {
  volume_id = "${aws_ebs_volume.foo.id}"
}

resource "aws_ami" "foo" {
  name = "tf-testing-%d"
  virtualization_type = "hvm"
  root_device_name = "/dev/sda1"
  ebs_block_device {
    device_name = "/dev/sda1"
    snapshot_id = "${aws_ebs_snapshot.foo.id}"
  }
}
	`, rInt)
}

func testAccAmiConfig_snapshotSize(rInt int) string {
	return fmt.Sprintf(`
resource "aws_ebs_volume" "foo" {
 	availability_zone = "us-west-2a"
 	size = 20
 	tags {
 	  Name = "testAccAmiConfig_snapshotSize"
 	}
}

resource "aws_ebs_snapshot" "foo" {
  volume_id = "${aws_ebs_volume.foo.id}"
}

resource "aws_ami" "foo" {
  name = "tf-testing-%d"
  virtualization_type = "hvm"
  root_device_name = "/dev/sda1"
  ebs_block_device {
    device_name = "/dev/sda1"
    snapshot_id = "${aws_ebs_snapshot.foo.id}"
  }
}
	`, rInt)
}
