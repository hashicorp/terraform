package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAmiDescription_natInstance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsAmiDescriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsAmiDescriptionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDescriptionID("aws_ami_description.nat_ami"),
					// Check attributes. Some attributes are tough to test - any not contained here should not be considered
					// stable and should not be used in interpolation. Exception to block_device_mappings which should both
					// show up consistently and break if certain references are not available. However modification of the
					// snapshot ID which is bound to happen on the NAT AMIs will cause testing to break consistently, so
					// deep inspection is not included, simply the count is checked.
					// Tags and product codes may need more testing, but I'm having a hard time finding images with
					// these attributes set.
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "architecture", "x86_64"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "block_device_mappings.#", "1"),
					resource.TestMatchResourceAttr("aws_ami_description.nat_ami", "creation_date", regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestMatchResourceAttr("aws_ami_description.nat_ami", "description", regexp.MustCompile("^Amazon Linux AMI")),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "hypervisor", "xen"),
					resource.TestMatchResourceAttr("aws_ami_description.nat_ami", "image_id", regexp.MustCompile("^ami-")),
					resource.TestMatchResourceAttr("aws_ami_description.nat_ami", "image_location", regexp.MustCompile("^amazon/")),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "image_owner_alias", "amazon"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "image_type", "machine"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "most_recent", "true"),
					resource.TestMatchResourceAttr("aws_ami_description.nat_ami", "name", regexp.MustCompile("^amzn-ami-vpc-nat")),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "owner_id", "137112412989"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "public", "true"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "product_codes.#", "0"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "root_device_name", "/dev/xvda"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "root_device_type", "ebs"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "sriov_net_support", "simple"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "state", "available"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "state_reason.code", "UNSET"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "state_reason.message", "UNSET"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "tags.#", "0"),
					resource.TestCheckResourceAttr("aws_ami_description.nat_ami", "virtualization_type", "hvm"),
				),
			},
		},
	})
}
func TestAccAWSAmiDescription_windowsInstance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsAmiDescriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsAmiDescriptionWindowsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDescriptionID("aws_ami_description.windows_ami"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "architecture", "x86_64"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "block_device_mappings.#", "27"),
					resource.TestMatchResourceAttr("aws_ami_description.windows_ami", "creation_date", regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestMatchResourceAttr("aws_ami_description.windows_ami", "description", regexp.MustCompile("^Microsoft Windows Server 2012")),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "hypervisor", "xen"),
					resource.TestMatchResourceAttr("aws_ami_description.windows_ami", "image_id", regexp.MustCompile("^ami-")),
					resource.TestMatchResourceAttr("aws_ami_description.windows_ami", "image_location", regexp.MustCompile("^amazon/")),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "image_owner_alias", "amazon"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "image_type", "machine"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "most_recent", "true"),
					resource.TestMatchResourceAttr("aws_ami_description.windows_ami", "name", regexp.MustCompile("^Windows_Server-2012-R2")),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "owner_id", "801119661308"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "platform", "windows"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "public", "true"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "product_codes.#", "0"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "root_device_name", "/dev/sda1"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "root_device_type", "ebs"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "sriov_net_support", "simple"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "state", "available"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "state_reason.code", "UNSET"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "state_reason.message", "UNSET"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "tags.#", "0"),
					resource.TestCheckResourceAttr("aws_ami_description.windows_ami", "virtualization_type", "hvm"),
				),
			},
		},
	})
}

func TestAccAWSAmiDescription_instanceStore(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsAmiDescriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsAmiDescriptionInstanceStoreConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDescriptionID("aws_ami_description.instance_store_ami"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "architecture", "x86_64"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "block_device_mappings.#", "0"),
					resource.TestMatchResourceAttr("aws_ami_description.instance_store_ami", "creation_date", regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "hypervisor", "xen"),
					resource.TestMatchResourceAttr("aws_ami_description.instance_store_ami", "image_id", regexp.MustCompile("^ami-")),
					resource.TestMatchResourceAttr("aws_ami_description.instance_store_ami", "image_location", regexp.MustCompile("images/hvm-instance/ubuntu-trusty-14.04-amd64-server")),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "image_type", "machine"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "most_recent", "true"),
					resource.TestMatchResourceAttr("aws_ami_description.instance_store_ami", "name", regexp.MustCompile("^ubuntu/images/hvm-instance/ubuntu-trusty-14.04-amd64-server")),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "owner_id", "099720109477"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "public", "true"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "product_codes.#", "0"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "root_device_type", "instance-store"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "sriov_net_support", "simple"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "state", "available"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "state_reason.code", "UNSET"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "state_reason.message", "UNSET"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "tags.#", "0"),
					resource.TestCheckResourceAttr("aws_ami_description.instance_store_ami", "virtualization_type", "hvm"),
				),
			},
		},
	})
}

func TestAccAWSAmiDescription_owners(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsAmiDescriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAwsAmiDescriptionOwnersConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsAmiDescriptionID("aws_ami_description.amazon_ami"),
				),
			},
		},
	})
}

func testAccCheckAwsAmiDescriptionDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAwsAmiDescriptionID(n string) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find AMI Description: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("AMI Description ID not set")
		}
		return nil
	}
}

// Using NAT AMIs for testing - I would expect with NAT gateways now a thing,
// that this will possibly be deprecated at some point in time. Other candidates
// for testing this after that may be Ubuntu's AMI's, or Amazon's regular
// Amazon Linux AMIs.
const testAccCheckAwsAmiDescriptionConfig = `
resource "aws_ami_description" "nat_ami" {
	most_recent = true
	filter {
		name = "owner-alias"
		values = ["amazon"]
	}
	filter {
		name = "name"
		values = ["amzn-ami-vpc-nat*"]
	}
	filter {
		name = "virtualization-type"
		values = ["hvm"]
	}
	filter {
		name = "root-device-type"
		values = ["ebs"]
	}
	filter {
		name = "block-device-mapping.volume-type"
		values = ["standard"]
	}
}
`

// Windows image test.
const testAccCheckAwsAmiDescriptionWindowsConfig = `
resource "aws_ami_description" "windows_ami" {
	most_recent = true
	filter {
		name = "owner-alias"
		values = ["amazon"]
	}
	filter {
		name = "name"
		values = ["Windows_Server-2012-R2*"]
	}
	filter {
		name = "virtualization-type"
		values = ["hvm"]
	}
	filter {
		name = "root-device-type"
		values = ["ebs"]
	}
	filter {
		name = "block-device-mapping.volume-type"
		values = ["gp2"]
	}
}
`

// Instance store test - using Ubuntu images
const testAccCheckAwsAmiDescriptionInstanceStoreConfig = `
resource "aws_ami_description" "instance_store_ami" {
	most_recent = true
	filter {
		name = "owner-id"
		values = ["099720109477"]
	}
	filter {
		name = "name"
		values = ["ubuntu/images/hvm-instance/ubuntu-trusty-14.04-amd64-server*"]
	}
	filter {
		name = "virtualization-type"
		values = ["hvm"]
	}
	filter {
		name = "root-device-type"
		values = ["instance-store"]
	}
}
`

// Testing owner parameter
const testAccCheckAwsAmiDescriptionOwnersConfig = `
resource "aws_ami_description" "amazon_ami" {
	most_recent = true
	owners = ["amazon"]
}
`
