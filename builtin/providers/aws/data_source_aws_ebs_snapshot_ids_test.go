package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceAwsEbsSnapshotIds_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEbsSnapshotIdsConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ebs_snapshot_ids.test"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsEbsSnapshotIds_empty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEbsSnapshotIdsConfig_empty,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ebs_snapshot_ids.empty"),
					resource.TestCheckResourceAttr("data.aws_ebs_snapshot_ids.empty", "ids.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceAwsEbsSnapshotIdsConfig_basic = `
resource "aws_ebs_volume" "test" {
    availability_zone = "us-west-2a"
    size              = 40
}

resource "aws_ebs_snapshot" "test" {
    volume_id = "${aws_ebs_volume.test.id}"
}

data "aws_ebs_snapshot_ids" "test" {
    owners = ["self"]
}
`

const testAccDataSourceAwsEbsSnapshotIdsConfig_empty = `
data "aws_ebs_snapshot_ids" "empty" {
    owners = ["000000000000"]
}
`
