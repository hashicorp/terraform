package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSPartition_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsPartitionConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPartition("data.aws_partition.current"),
				),
			},
		},
	})
}

func testAccCheckAwsPartition(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find resource: %s", n)
		}

		expected := testAccProvider.Meta().(*AWSClient).partition
		if rs.Primary.Attributes["partition"] != expected {
			return fmt.Errorf("Incorrect Partition: expected %q, got %q", expected, rs.Primary.Attributes["partition"])
		}

		return nil
	}
}

const testAccCheckAwsPartitionConfig_basic = `
data "aws_partition" "current" { }
`
