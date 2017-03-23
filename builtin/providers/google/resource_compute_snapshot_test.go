package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"google.golang.org/api/compute/v1"
)

func TestAccComputeSnapshot_basic(t *testing.T) {
	snapshotName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var snapshot compute.Snapshot
	diskName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSnapshotDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSnapshot_basic(snapshotName, diskName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSnapshotExists(
						"google_compute_snapshot.foobar", &snapshot),
				),
			},
		},
	})
}

func TestAccComputeSnapshot_encryption(t *testing.T) {
	snapshotName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))
	var snapshot compute.Snapshot
	diskName := fmt.Sprintf("tf-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeSnapshotDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeSnapshot_encryption(snapshotName, diskName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeSnapshotExists(
						"google_compute_snapshot.foobar", &snapshot),
					testAccCheckSnapshotEncryptionKey(
						"google_compute_snapshot.foobar", &snapshot),
				),
			},
		},
	})
}

func testAccCheckComputeSnapshotDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_compute_snapshot" {
			continue
		}

		_, err := config.clientCompute.Snapshots.Get(
			config.Project, rs.Primary.ID).Do()
		if err == nil {
			return fmt.Errorf("Snapshot still exists")
		}
	}

	return nil
}

func testAccCheckComputeSnapshotExists(n string, snapshot *compute.Snapshot) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)

		found, err := config.clientCompute.Snapshots.Get(
			config.Project, rs.Primary.ID).Do()
		if err != nil {
			return err
		}

		if found.Name != rs.Primary.ID {
			return fmt.Errorf("Snapshot %s not found", n)
		}

		attr := rs.Primary.Attributes["snapshot_encryption_key_sha256"]
		if found.SnapshotEncryptionKey != nil && found.SnapshotEncryptionKey.Sha256 != attr {
			return fmt.Errorf("Snapshot %s has mismatched encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SnapshotEncryptionKey.Sha256)
		} else if found.SnapshotEncryptionKey == nil && attr != "" {
			return fmt.Errorf("Snapshot %s has mismatched encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SnapshotEncryptionKey)
		}

		attr = rs.Primary.Attributes["snapshot_encryption_key_raw"]
		if found.SnapshotEncryptionKey != nil && found.SnapshotEncryptionKey.RawKey != attr {
			return fmt.Errorf("Snapshot %s has mismatched encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SnapshotEncryptionKey.RawKey)
		} else if found.SnapshotEncryptionKey == nil && attr != "" {
			return fmt.Errorf("Snapshot %s has mismatched encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SnapshotEncryptionKey)
		}

		attr = rs.Primary.Attributes["source_disk_encryption_key_sha256"]
		if found.SourceDiskEncryptionKey != nil && found.SourceDiskEncryptionKey.Sha256 != attr {
			return fmt.Errorf("Snapshot %s has mismatched source disk encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SourceDiskEncryptionKey.Sha256)
		} else if found.SourceDiskEncryptionKey == nil && attr != "" {
			return fmt.Errorf("Snapshot %s has mismatched source disk encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SourceDiskEncryptionKey)
		}

		attr = rs.Primary.Attributes["source_disk_encryption_key_raw"]
		if found.SourceDiskEncryptionKey != nil && found.SourceDiskEncryptionKey.RawKey != attr {
			return fmt.Errorf("Snapshot %s has mismatched source disk encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SourceDiskEncryptionKey.RawKey)
		} else if found.SourceDiskEncryptionKey == nil && attr != "" {
			return fmt.Errorf("Snapshot %s has mismatched source disk encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SourceDiskEncryptionKey)
		}

		attr = rs.Primary.Attributes["source_disk_id"]
		if found.SourceDiskId != attr {
			return fmt.Errorf("Snapshot %s has mismatched source disk id.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SourceDiskId)
		}

		attr = rs.Primary.Attributes["source_disk"]
		if found.SourceDisk != attr {
			return fmt.Errorf("Snapshot %s has mismatched source disk.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SourceDisk)
		}

		attr = rs.Primary.Attributes["self_link"]
		if found.SelfLink != attr {
			return fmt.Errorf("Snapshot %s has mismatched self link.\nTF State: %+v.\nGCP State: %+v",
				n, attr, found.SelfLink)
		}

		*snapshot = *found

		return nil
	}
}

func testAccCheckSnapshotEncryptionKey(n string, snapshot *compute.Snapshot) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		attr := rs.Primary.Attributes["snapshot_encryption_key_sha256"]
		if snapshot.SnapshotEncryptionKey == nil && attr != "" {
			return fmt.Errorf("Snapshot %s has mismatched encryption key.\nTF State: %+v\nGCP State: <empty>", n, attr)
		}

		if attr != snapshot.SnapshotEncryptionKey.Sha256 {
			return fmt.Errorf("Snapshot %s has mismatched encryption key.\nTF State: %+v.\nGCP State: %+v",
				n, attr, snapshot.SnapshotEncryptionKey.Sha256)
		}
		return nil
	}
}

func testAccComputeSnapshot_basic(snapshotName string, diskName string) string {
	return fmt.Sprintf(`
resource "google_compute_disk" "foobar" {
	name = "%s"
	image = "debian-8-jessie-v20160921"
	size = 10
	type = "pd-ssd"
	zone = "us-central1-a"
}

resource "google_compute_snapshot" "foobar" {
	name = "%s"
	disk = "${google_compute_disk.foobar.name}"
	zone = "us-central1-a"
}`, diskName, snapshotName)
}

func testAccComputeSnapshot_encryption(snapshotName string, diskName string) string {
	return fmt.Sprintf(`
resource "google_compute_disk" "foobar" {
	name = "%s"
	image = "debian-8-jessie-v20160921"
	size = 10
	type = "pd-ssd"
	zone = "us-central1-a"
	disk_encryption_key_raw = "SGVsbG8gZnJvbSBHb29nbGUgQ2xvdWQgUGxhdGZvcm0="
}
resource "google_compute_snapshot" "foobar" {
	name = "%s"
	disk = "${google_compute_disk.foobar.name}"
	zone = "us-central1-a"
	source_disk_encryption_key_raw = "SGVsbG8gZnJvbSBHb29nbGUgQ2xvdWQgUGxhdGZvcm0="
	snapshot_encryption_key_raw = "SGVsbG8gZnJvbSBHb29nbGUgQ2xvdWQgUGxhdGZvcm0="
}`, diskName, snapshotName)
}
