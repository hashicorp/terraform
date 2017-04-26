package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/satori/uuid"
)

func TestAccDataSourceAwsAmiIds_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsAmiIdsConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDataSourceID("data.aws_ami_ids.ubuntu"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsAmiIds_sorted(t *testing.T) {
	uuid := uuid.NewV4().String()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsAmiIdsConfig_sorted1(uuid),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("aws_ami_from_instance.a", "id"),
					resource.TestCheckResourceAttrSet("aws_ami_from_instance.b", "id"),
				),
			},
			{
				Config: testAccDataSourceAwsAmiIdsConfig_sorted2(uuid),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsSnapshotDataSourceID("data.aws_ami_ids.test"),
					resource.TestCheckResourceAttr("data.aws_ami_ids.test", "ids.#", "2"),
					resource.TestCheckResourceAttrPair(
						"data.aws_ami_ids.test", "ids.0",
						"aws_ami_from_instance.b", "id"),
					resource.TestCheckResourceAttrPair(
						"data.aws_ami_ids.test", "ids.1",
						"aws_ami_from_instance.a", "id"),
				),
			},
		},
	})
}

func TestAccDataSourceAwsAmiIds_empty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsAmiIdsConfig_empty,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDataSourceID("data.aws_ami_ids.empty"),
					resource.TestCheckResourceAttr("data.aws_ami_ids.empty", "ids.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceAwsAmiIdsConfig_basic = `
data "aws_ami_ids" "ubuntu" {
    owners = ["099720109477"]

    filter {
        name   = "name"
        values = ["ubuntu/images/ubuntu-*-*-amd64-server-*"]
    }
}
`

func testAccDataSourceAwsAmiIdsConfig_sorted1(uuid string) string {
	return fmt.Sprintf(`
resource "aws_instance" "test" {
    ami           = "ami-efd0428f"
    instance_type = "m3.medium"

    count = 2
}

resource "aws_ami_from_instance" "a" {
    name                    = "tf-test-%s-a"
    source_instance_id      = "${aws_instance.test.*.id[0]}"
    snapshot_without_reboot = true
}

resource "aws_ami_from_instance" "b" {
    name                    = "tf-test-%s-b"
    source_instance_id      = "${aws_instance.test.*.id[1]}"
    snapshot_without_reboot = true

    // We want to ensure that 'aws_ami_from_instance.a.creation_date' is less
    // than 'aws_ami_from_instance.b.creation_date' so that we can ensure that
    // the images are being sorted correctly.
    depends_on = ["aws_ami_from_instance.a"]
}
`, uuid, uuid)
}

func testAccDataSourceAwsAmiIdsConfig_sorted2(uuid string) string {
	return testAccDataSourceAwsAmiIdsConfig_sorted1(uuid) + fmt.Sprintf(`
data "aws_ami_ids" "test" {
  owners     = ["self"]
  name_regex = "^tf-test-%s-"
}
`, uuid)
}

const testAccDataSourceAwsAmiIdsConfig_empty = `
data "aws_ami_ids" "empty" {
  filter {
    name   = "name"
    values = []
  }
}
`
