package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSnapshotCreateVolumePermission_Basic(t *testing.T) {
	var snapshotId, accountId string

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Scaffold everything
			resource.TestStep{
				Config: testAccAWSSnapshotCreateVolumePermissionConfig(true),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGetAttr("aws_ebs_snapshot.example_snapshot", "id", &snapshotId),
					testCheckResourceGetAttr("data.aws_caller_identity.current", "account_id", &accountId),
					testAccAWSSnapshotCreateVolumePermissionExists(&accountId, &snapshotId),
				),
			},
			// Drop just create volume permission to test destruction
			resource.TestStep{
				Config: testAccAWSSnapshotCreateVolumePermissionConfig(false),
				Check: resource.ComposeTestCheckFunc(
					testAccAWSSnapshotCreateVolumePermissionDestroyed(&accountId, &snapshotId),
				),
			},
		},
	})
}

func testAccAWSSnapshotCreateVolumePermissionExists(accountId, snapshotId *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasCreateVolumePermission(conn, *snapshotId, *accountId); err != nil {
			return err
		} else if !has {
			return fmt.Errorf("create volume permission does not exist for '%s' on '%s'", *accountId, *snapshotId)
		}
		return nil
	}
}

func testAccAWSSnapshotCreateVolumePermissionDestroyed(accountId, snapshotId *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasCreateVolumePermission(conn, *snapshotId, *accountId); err != nil {
			return err
		} else if has {
			return fmt.Errorf("create volume permission still exists for '%s' on '%s'", *accountId, *snapshotId)
		}
		return nil
	}
}

func testAccAWSSnapshotCreateVolumePermissionConfig(includeCreateVolumePermission bool) string {
	base := `
data "aws_caller_identity" "current" {}

resource "aws_ebs_volume" "example" {
  availability_zone = "us-west-2a"
  size              = 40

  tags {
    Name = "ebs_snap_perm"
  }
}

resource "aws_ebs_snapshot" "example_snapshot" {
  volume_id = "${aws_ebs_volume.example.id}"
}
`

	if !includeCreateVolumePermission {
		return base
	}

	return base + fmt.Sprintf(`
resource "aws_snapshot_create_volume_permission" "self-test" {
  snapshot_id = "${aws_ebs_snapshot.example_snapshot.id}"
  account_id  = "${data.aws_caller_identity.current.account_id}"
}
`)
}
