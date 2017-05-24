package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDataSourceAwsEfsFileSystem(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAwsEfsFileSystemConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAwsEfsFileSystemCheck("data.aws_efs_file_system.by_creation_token"),
					testAccDataSourceAwsEfsFileSystemCheck("data.aws_efs_file_system.by_id"),
				),
			},
		},
	})
}

func testAccDataSourceAwsEfsFileSystemCheck(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("root module has no resource called %s", name)
		}

		efsRs, ok := s.RootModule().Resources["aws_efs_file_system.test"]
		if !ok {
			return fmt.Errorf("can't find aws_efs_file_system.test in state")
		}

		attr := rs.Primary.Attributes

		if attr["creation_token"] != efsRs.Primary.Attributes["creation_token"] {
			return fmt.Errorf(
				"creation_token is %s; want %s",
				attr["creation_token"],
				efsRs.Primary.Attributes["creation_token"],
			)
		}

		if attr["id"] != efsRs.Primary.Attributes["id"] {
			return fmt.Errorf(
				"file_system_id is %s; want %s",
				attr["id"],
				efsRs.Primary.Attributes["id"],
			)
		}

		return nil
	}
}

const testAccDataSourceAwsEfsFileSystemConfig = `
resource "aws_efs_file_system" "test" {}

data "aws_efs_file_system" "by_creation_token" {
  creation_token = "${aws_efs_file_system.test.creation_token}"
}

data "aws_efs_file_system" "by_id" {
  file_system_id = "${aws_efs_file_system.test.id}"
}
`
