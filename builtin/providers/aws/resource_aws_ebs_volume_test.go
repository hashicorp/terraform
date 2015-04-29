package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAwsEbsVolume(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsEbsVolumeConfig,
			},
		},
	})
}

const testAccAwsEbsVolumeConfig = `
resource "aws_ebs_volume" "test" {
	availability_zone = "us-east-1d"
	size = 1
}
`
