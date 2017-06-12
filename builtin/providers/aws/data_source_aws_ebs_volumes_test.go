package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSEbsVolumesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEbsVolumesDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsVolumesDataSourceID("data.aws_ebs_volumes.ebs_volumes"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.#", "1"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.0.volume_type", "gp2"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "ids.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSEbsVolumesDataSource_standard(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEbsVolumesDataSourceConfigStandard,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsVolumesDataSourceID("data.aws_ebs_volumes.ebs_volumes"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.#", "1"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.0.volume_type", "standard"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.0.tags.3542208501.value", "Standard Volume"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "ids.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSEbsVolumesDataSource_multipleFilters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsEbsVolumesDataSourceConfigWithMultipleFilters,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsEbsVolumesDataSourceID("data.aws_ebs_volumes.ebs_volumes"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.#", "2"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "ids.#", "2"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.0.size", "10"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.0.volume_type", "gp2"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.0.tags.614424690.value", "MyTag"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.1.size", "10"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.1.volume_type", "gp2"),
					resource.TestCheckResourceAttr("data.aws_ebs_volumes.ebs_volumes", "volumes.1.tags.614424690.value", "MyTag"),
				),
			},
		},
	})
}

func testAccCheckAwsEbsVolumesDataSourceID(n string) resource.TestCheckFunc {
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

const testAccCheckAwsEbsVolumesDataSourceConfig = `
resource "aws_ebs_volume" "example" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 40
    tags {
        Name = "External Volume"
    }
}

data "aws_ebs_volumes" "ebs_volumes" {
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

const testAccCheckAwsEbsVolumesDataSourceConfigStandard = `
resource "aws_ebs_volume" "example" {
    availability_zone = "us-west-2a"
    type = "standard"
    size = 10
    tags {
        Name = "Standard Volume"
    }
}

data "aws_ebs_volumes" "ebs_volumes" {
    filter {
      name = "tag:Name"
      values = ["Standard Volume"]
    }
    filter {
      name = "volume-type"
      values = ["${aws_ebs_volume.example.type}"]
    }
}
`

const testAccCheckAwsEbsVolumesDataSourceConfigWithMultipleFilters = `
resource "aws_ebs_volume" "external1" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 10
    tags {
        Name = "External Volume 1"
        Group = "MyTag"
    }
}

resource "aws_ebs_volume" "external2" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 10
    tags {
        Name = "External Volume 2"
        Group = "MyTag"
    }
}

resource "aws_ebs_volume" "external3" {
    availability_zone = "us-west-2a"
    type = "gp2"
    size = 11
    tags {
        Name = "External Volume 3"
        Group = "MyTag2"
    }
}

data "aws_ebs_volumes" "ebs_volumes" {
    filter {
			name = "tag:Group"
			values = ["MyTag"]
    }
    filter {
			name = "volume-type"
			values = ["${aws_ebs_volume.external3.type}"]
    }
}
`
