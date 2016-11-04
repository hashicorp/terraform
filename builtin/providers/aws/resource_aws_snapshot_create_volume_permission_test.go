package aws

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSnapshotCreateVolumePermission_Basic(t *testing.T) {
	snapshot_id := ""
	account_id := os.Getenv("AWS_ACCOUNT_ID")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if os.Getenv("AWS_ACCOUNT_ID") == "" {
				t.Fatal("AWS_ACCOUNT_ID must be set")
			}
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			// Scaffold everything
			resource.TestStep{
				Config: testAccAWSSnapshotCreateVolumePermissionConfig(account_id, true),
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceGetAttr("aws_ami_copy.test", "block_device_mappings.0.ebs.snapshot_id", &snapshot_id),
					testAccAWSSnapshotCreateVolumePermissionExists(account_id, &snapshot_id),
				),
			},
			// Drop just create volume permission to test destruction
			resource.TestStep{
				Config: testAccAWSSnapshotCreateVolumePermissionConfig(account_id, false),
				Check: resource.ComposeTestCheckFunc(
					testAccAWSSnapshotCreateVolumePermissionDestroyed(account_id, &snapshot_id),
				),
			},
		},
	})
}

func testAccAWSSnapshotCreateVolumePermissionExists(account_id string, snapshot_id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasCreateVolumePermission(conn, *snapshot_id, account_id); err != nil {
			return err
		} else if !has {
			return fmt.Errorf("create volume permission does not exist for '%s' on '%s'", account_id, *snapshot_id)
		}
		return nil
	}
}

func testAccAWSSnapshotCreateVolumePermissionDestroyed(account_id string, snapshot_id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasCreateVolumePermission(conn, *snapshot_id, account_id); err != nil {
			return err
		} else if has {
			return fmt.Errorf("create volume permission still exists for '%s' on '%s'", account_id, *snapshot_id)
		}
		return nil
	}
}

func testAccAWSSnapshotCreateVolumePermissionConfig(account_id string, includeCreateVolumePermission bool) string {
	base := `
resource "aws_ami_copy" "test" {
  name = "create-volume-permission-test"
  description = "Create Volume Permission Test Copy"
  source_ami_id = "ami-7172b611"
  source_ami_region = "us-west-2"
}
`

	if !includeCreateVolumePermission {
		return base
	}

	return base + fmt.Sprintf(`
resource "aws_snapshot_create_volume_permission" "self-test" {
  snapshot_id   = "${lookup(lookup(element(aws_ami_copy.test.block_device_mappings, 0), "ebs"), "snapshot_id")}"
  account_id = "%s"
}
`, account_id)
}
