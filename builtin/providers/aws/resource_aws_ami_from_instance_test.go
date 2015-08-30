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

func TestAccAWSAMIFromInstance(t *testing.T) {
	var amiId string
	snapshots := []string{}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSAMIFromInstanceConfig,
				Check: func(state *terraform.State) error {
					rs, ok := state.RootModule().Resources["aws_ami_from_instance.test"]
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
					if expected := "terraform-acc-ami-from-instance"; *image.Name != expected {
						return fmt.Errorf("wrong name; expected %v, got %v", expected, image.Name)
					}

					for _, bdm := range image.BlockDeviceMappings {
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

var testAccAWSAMIFromInstanceConfig = `
provider "aws" {
	region = "us-east-1"
}

resource "aws_instance" "test" {
    // This AMI has one block device mapping, so we expect to have
    // one snapshot in our created AMI.
    ami = "ami-408c7f28"
    instance_type = "t1.micro"
}

resource "aws_ami_from_instance" "test" {
    name = "terraform-acc-ami-from-instance"
    description = "Testing Terraform aws_ami_from_instance resource"
    source_instance_id = "${aws_instance.test.id}"
}
`
