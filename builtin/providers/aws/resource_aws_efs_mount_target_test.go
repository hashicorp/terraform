package aws

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/efs"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEFSMountTarget_basic(t *testing.T) {
	var mount efs.MountTargetDescription
	ct := fmt.Sprintf("createtoken-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsMountTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEFSMountTargetConfig(ct),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsMountTarget(
						"aws_efs_mount_target.alpha",
						&mount,
					),
					resource.TestMatchResourceAttr(
						"aws_efs_mount_target.alpha",
						"dns_name",
						regexp.MustCompile("^[^.]+.efs.us-west-2.amazonaws.com$"),
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSEFSMountTargetConfigModified(ct),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsMountTarget(
						"aws_efs_mount_target.alpha",
						&mount,
					),
					resource.TestMatchResourceAttr(
						"aws_efs_mount_target.alpha",
						"dns_name",
						regexp.MustCompile("^[^.]+.efs.us-west-2.amazonaws.com$"),
					),
					testAccCheckEfsMountTarget(
						"aws_efs_mount_target.beta",
						&mount,
					),
					resource.TestMatchResourceAttr(
						"aws_efs_mount_target.beta",
						"dns_name",
						regexp.MustCompile("^[^.]+.efs.us-west-2.amazonaws.com$"),
					),
				),
			},
		},
	})
}

func TestAccAWSEFSMountTarget_disappears(t *testing.T) {
	var mount efs.MountTargetDescription

	ct := fmt.Sprintf("createtoken-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVpnGatewayDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEFSMountTargetConfig(ct),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEfsMountTarget(
						"aws_efs_mount_target.alpha",
						&mount,
					),
					testAccAWSEFSMountTargetDisappears(&mount),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestResourceAWSEFSMountTarget_mountTargetDnsName(t *testing.T) {
	actual := resourceAwsEfsMountTargetDnsName("fs-123456ab", "non-existent-1")

	expected := "fs-123456ab.efs.non-existent-1.amazonaws.com"
	if actual != expected {
		t.Fatalf("Expected EFS mount target DNS name to be %s, got %s",
			expected, actual)
	}
}

func TestResourceAWSEFSMountTarget_hasEmptyMountTargets(t *testing.T) {
	mto := &efs.DescribeMountTargetsOutput{
		MountTargets: []*efs.MountTargetDescription{},
	}

	var actual bool

	actual = hasEmptyMountTargets(mto)
	if !actual {
		t.Fatalf("Expected return value to be true, got %t", actual)
	}

	// Add an empty mount target.
	mto.MountTargets = append(mto.MountTargets, &efs.MountTargetDescription{})

	actual = hasEmptyMountTargets(mto)
	if actual {
		t.Fatalf("Expected return value to be false, got %t", actual)
	}

}

func testAccCheckEfsMountTargetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).efsconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_efs_mount_target" {
			continue
		}

		resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
			MountTargetId: aws.String(rs.Primary.ID),
		})
		if err != nil {
			if efsErr, ok := err.(awserr.Error); ok && efsErr.Code() == "MountTargetNotFound" {
				// gone
				return nil
			}
			return fmt.Errorf("Error describing EFS Mount in tests: %s", err)
		}
		if len(resp.MountTargets) > 0 {
			return fmt.Errorf("EFS Mount target %q still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckEfsMountTarget(resourceID string, mount *efs.MountTargetDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		fs, ok := s.RootModule().Resources[resourceID]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceID)
		}

		conn := testAccProvider.Meta().(*AWSClient).efsconn
		mt, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
			MountTargetId: aws.String(fs.Primary.ID),
		})
		if err != nil {
			return err
		}

		if *mt.MountTargets[0].MountTargetId != fs.Primary.ID {
			return fmt.Errorf("Mount target ID mismatch: %q != %q",
				*mt.MountTargets[0].MountTargetId, fs.Primary.ID)
		}

		*mount = *mt.MountTargets[0]

		return nil
	}
}

func testAccAWSEFSMountTargetDisappears(mount *efs.MountTargetDescription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).efsconn

		_, err := conn.DeleteMountTarget(&efs.DeleteMountTargetInput{
			MountTargetId: mount.MountTargetId,
		})

		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "MountTargetNotFound" {
				return nil
			}
			return err
		}

		return resource.Retry(3*time.Minute, func() *resource.RetryError {
			resp, err := conn.DescribeMountTargets(&efs.DescribeMountTargetsInput{
				MountTargetId: mount.MountTargetId,
			})
			if err != nil {
				if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "MountTargetNotFound" {
					return nil
				}
				return resource.NonRetryableError(
					fmt.Errorf("Error reading EFS mount target: %s", err))
			}
			if resp.MountTargets == nil || len(resp.MountTargets) < 1 {
				return nil
			}
			if *resp.MountTargets[0].LifeCycleState == "deleted" {
				return nil
			}
			return resource.RetryableError(fmt.Errorf(
				"Waiting for EFS mount target: %s", *mount.MountTargetId))
		})
	}

}

func testAccAWSEFSMountTargetConfig(ct string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "foo" {
	creation_token = "%s"
}

resource "aws_efs_mount_target" "alpha" {
	file_system_id = "${aws_efs_file_system.foo.id}"
	subnet_id = "${aws_subnet.alpha.id}"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.0.0.0/16"
	tags {
		Name = "testAccAWSEFSMountTargetConfig"
	}
}

resource "aws_subnet" "alpha" {
	vpc_id = "${aws_vpc.foo.id}"
	availability_zone = "us-west-2a"
	cidr_block = "10.0.1.0/24"
}
`, ct)
}

func testAccAWSEFSMountTargetConfigModified(ct string) string {
	return fmt.Sprintf(`
resource "aws_efs_file_system" "foo" {
	creation_token = "%s"
}

resource "aws_efs_mount_target" "alpha" {
	file_system_id = "${aws_efs_file_system.foo.id}"
	subnet_id = "${aws_subnet.alpha.id}"
}

resource "aws_efs_mount_target" "beta" {
	file_system_id = "${aws_efs_file_system.foo.id}"
	subnet_id = "${aws_subnet.beta.id}"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.0.0.0/16"
	tags {
		Name = "testAccAWSEFSMountTargetConfigModified"
	}
}

resource "aws_subnet" "alpha" {
	vpc_id = "${aws_vpc.foo.id}"
	availability_zone = "us-west-2a"
	cidr_block = "10.0.1.0/24"
}

resource "aws_subnet" "beta" {
	vpc_id = "${aws_vpc.foo.id}"
	availability_zone = "us-west-2b"
	cidr_block = "10.0.2.0/24"
}
`, ct)
}
