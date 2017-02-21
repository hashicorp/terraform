package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEbsVolumeDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEbsVolumeDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsVolumeDataSourceID("data.aws_ebs_volume.ebs_volume"),
					resource.TestCheckResourceAttr("data.aws_ebs_volume.ebs_volume", "size", "40"),
				),
			},
		},
	})
}

func TestAccAWSEbsVolumeDataSource_multipleFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEbsVolumeDataSourceConfigWithMultipleFilters,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsVolumeDataSourceID("data.aws_ebs_volume.ebs_volume"),
					resource.TestCheckResourceAttr("data.aws_ebs_volume.ebs_volume", "size", "10"),
					resource.TestCheckResourceAttr("data.aws_ebs_volume.ebs_volume", "volume_type", "gp2"),
				),
			},
		},
	})
}

func testAccCheckAwsEbsVolumeDataSourceID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find Volume data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Volume data source ID not set")
		}
		return nil
	}
}

const testAccCheckAwsEbsVolumeDataSourceConfig = `
resource "aws_ebs_volume" "example" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 40
    tags {
        Name = "External Volume"
    }
}

data "aws_ebs_volume" "ebs_volume" {
    most_recent = true
    filter {
	name = "tag:Name"
	values = ["External Volume"]
    }
    filter {
	name = "volume-type"
	values = ["${aws_ebs_volume.example.type}"]
    }
}
`

const testAccCheckAwsEbsVolumeDataSourceConfigWithMultipleFilters = `
resource "aws_ebs_volume" "external1" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 10
    tags {
        Name = "External Volume 1"
    }
}

data "aws_ebs_volume" "ebs_volume" {
    most_recent = true
    filter {
	name = "tag:Name"
	values = ["External Volume 1"]
    }
    filter {
	name = "size"
	values = ["${aws_ebs_volume.external1.size}"]
    }
    filter {
	name = "volume-type"
	values = ["${aws_ebs_volume.external1.type}"]
    }
}
`
