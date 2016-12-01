package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEBSSnapshot_basic(t *testing.T) {
	var v ec2.Snapshot
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsEbsSnapshotConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnapshotExists("aws_ebs_snapshot.test", &v),
				),
			},
		},
	})
}

func TestAccAWSEBSSnapshot_withDescription(t *testing.T) {
	var v ec2.Snapshot
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsEbsSnapshotConfigWithDescription,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSnapshotExists("aws_ebs_snapshot.test", &v),
					resource.TestCheckResourceAttr("aws_ebs_snapshot.test", "description", "EBS Snapshot Acceptance Test"),
				),
			},
		},
	})
}

func testAccCheckSnapshotExists(n string, v *ec2.Snapshot) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		request := &ec2.DescribeSnapshotsInput{
			SnapshotIds: []*string{aws.String(rs.Primary.ID)},
		}

		response, err := conn.DescribeSnapshots(request)
		if err == nil {
			if response.Snapshots != nil && len(response.Snapshots) > 0 {
				*v = *response.Snapshots[0]
				return nil
			}
		}
		return fmt.Errorf("Error finding EC2 Snapshot %s", rs.Primary.ID)
	}
}

const testAccAwsEbsSnapshotConfig = `
resource "aws_ebs_volume" "test" {
	availability_zone = "us-west-2a"
	size = 1
}

resource "aws_ebs_snapshot" "test" {
	volume_id = "${aws_ebs_volume.test.id}"
}
`

const testAccAwsEbsSnapshotConfigWithDescription = `
resource "aws_ebs_volume" "description_test" {
	availability_zone = "us-west-2a"
	size = 1
}

resource "aws_ebs_snapshot" "test" {
	volume_id = "${aws_ebs_volume.description_test.id}"
	description = "EBS Snapshot Acceptance Test"
}
`
