package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEbsSnapshotDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEbsSnapshotDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ebs_snapshot.snapshot"),
					resource.TestCheckResourceAttr("data.aws_ebs_snapshot.snapshot", "volume_size", "40"),
				),
			},
		},
	})
}

func TestAccAWSEbsSnapshotDataSource_multipleFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEbsSnapshotDataSourceConfigWithMultipleFilters,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ebs_snapshot.snapshot"),
					resource.TestCheckResourceAttr("data.aws_ebs_snapshot.snapshot", "volume_size", "10"),
				),
			},
		},
	})
}

func testAccCheckAwsEbsSnapshotDataSourceID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find snapshot data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Snapshot data source ID not set")
		}
		return nil
	}
}

const testAccCheckAwsEbsSnapshotDataSourceConfig = `
resource "aws_ebs_volume" "example" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 40
    tags {
        Name = "External Volume"
    }
}

resource "aws_ebs_snapshot" "snapshot" {
    volume_id = "${aws_ebs_volume.example.id}"
}

data "aws_ebs_snapshot" "snapshot" {
    most_recent = true
    snapshot_ids = ["${aws_ebs_snapshot.snapshot.id}"]
}
`

const testAccCheckAwsEbsSnapshotDataSourceConfigWithMultipleFilters = `
resource "aws_ebs_volume" "external1" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 10
    tags {
        Name = "External Volume 1"
    }
}

resource "aws_ebs_snapshot" "snapshot" {
    volume_id = "${aws_ebs_volume.external1.id}"
}

data "aws_ebs_snapshot" "snapshot" {
    most_recent = true
    snapshot_ids = ["${aws_ebs_snapshot.snapshot.id}"]
    filter {
	name = "volume-size"
	values = ["10"]
    }
}
`
