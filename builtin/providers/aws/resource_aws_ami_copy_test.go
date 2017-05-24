package aws

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAMICopy(t *testing.T) {
	var amiId string
	snapshots := []string{}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAMICopyConfig,
				Check: func(state *terraform.State) error {
					rs, ok := state.RootModule().Resources["aws_ami_copy.test"]
					if !ok {
						return fmt.Errorf("AMI resource not found")
					}

					amiId = rs.Primary.ID

					if amiId == "" {
						return fmt.Errorf("AMI id is not set")
					}

					conn := testAccProvider.Meta().(*AWSClient).ec2conn
					req := &ec2.DescribeImagesInput{
						ImageIds: []*string{aws.String(amiId)},
					}
					describe, err := conn.DescribeImages(req)
					if err != nil {
						return err
					}

					if len(describe.Images) != 1 ||
						*describe.Images[0].ImageId != rs.Primary.ID {
						return fmt.Errorf("AMI not found")
					}

					image := describe.Images[0]
					if expected := "available"; *image.State != expected {
						return fmt.Errorf("invalid image state; expected %v, got %v", expected, image.State)
					}
					if expected := "machine"; *image.ImageType != expected {
						return fmt.Errorf("wrong image type; expected %v, got %v", expected, image.ImageType)
					}
					if expected := "terraform-acc-ami-copy"; *image.Name != expected {
						return fmt.Errorf("wrong name; expected %v, got %v", expected, image.Name)
					}

					for _, bdm := range image.BlockDeviceMappings {
						// The snapshot ID might not be set,
						// even for a block device that is an
						// EBS volume.
						if bdm.Ebs != nil && bdm.Ebs.SnapshotId != nil {
							snapshots = append(snapshots, *bdm.Ebs.SnapshotId)
						}
					}

					if expected := 1; len(snapshots) != expected {
						return fmt.Errorf("wrong number of snapshots; expected %v, got %v", expected, len(snapshots))
					}

					return nil
				},
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			conn := testAccProvider.Meta().(*AWSClient).ec2conn
			diReq := &ec2.DescribeImagesInput{
				ImageIds: []*string{aws.String(amiId)},
			}
			diRes, err := conn.DescribeImages(diReq)
			if err != nil {
				return err
			}

			if len(diRes.Images) > 0 {
				state := diRes.Images[0].State
				return fmt.Errorf("AMI %v remains in state %v", amiId, state)
			}

			stillExist := make([]string, 0, len(snapshots))
			checkErrors := make(map[string]error)
			for _, snapshotId := range snapshots {
				dsReq := &ec2.DescribeSnapshotsInput{
					SnapshotIds: []*string{aws.String(snapshotId)},
				}
				_, err := conn.DescribeSnapshots(dsReq)
				if err == nil {
					stillExist = append(stillExist, snapshotId)
					continue
				}

				awsErr, ok := err.(awserr.Error)
				if !ok {
					checkErrors[snapshotId] = err
					continue
				}

				if awsErr.Code() != "InvalidSnapshot.NotFound" {
					checkErrors[snapshotId] = err
					continue
				}
			}

			if len(stillExist) > 0 || len(checkErrors) > 0 {
				errParts := []string{
					"Expected all snapshots to be gone, but:",
				}
				for _, snapshotId := range stillExist {
					errParts = append(
						errParts,
						fmt.Sprintf("- %v still exists", snapshotId),
					)
				}
				for snapshotId, err := range checkErrors {
					errParts = append(
						errParts,
						fmt.Sprintf("- checking %v gave error: %v", snapshotId, err),
					)
				}
				return errors.New(strings.Join(errParts, "\n"))
			}

			return nil
		},
	})
}

var testAccAWSAMICopyConfig = `
provider "aws" {
	region = "us-east-1"
}

// An AMI can't be directly copied from one account to another, and
// we can't rely on any particular AMI being available since anyone
// can run this test in whatever account they like.
// Therefore we jump through some hoops here:
//  - Spin up an EC2 instance based on a public AMI
//  - Create an AMI by snapshotting that EC2 instance, using
//    aws_ami_from_instance .
//  - Copy the new AMI using aws_ami_copy .
//
// Thus this test can only succeed if the aws_ami_from_instance resource
// is working. If it's misbehaving it will likely cause this test to fail too.

// Since we're booting a t2.micro HVM instance we need a VPC for it to boot
// up into.

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccAWSAMICopyConfig"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_instance" "test" {
    // This AMI has one block device mapping, so we expect to have
    // one snapshot in our created AMI.
    // This is an Ubuntu Linux HVM AMI. A public HVM AMI is required
    // because paravirtual images cannot be copied between accounts.
    ami = "ami-0f8bce65"
    instance_type = "t2.micro"
    tags {
        Name = "terraform-acc-ami-copy-victim"
    }

    subnet_id = "${aws_subnet.foo.id}"
}

resource "aws_ami_from_instance" "test" {
    name = "terraform-acc-ami-copy-victim"
    description = "Testing Terraform aws_ami_from_instance resource"
    source_instance_id = "${aws_instance.test.id}"
}

resource "aws_ami_copy" "test" {
    name = "terraform-acc-ami-copy"
    description = "Testing Terraform aws_ami_copy resource"
    source_ami_id = "${aws_ami_from_instance.test.id}"
    source_ami_region = "us-east-1"
}
`
