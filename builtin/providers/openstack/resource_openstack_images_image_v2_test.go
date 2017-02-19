package openstack

import (
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"strings"
)

func TestAccImagesImageV2_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImagesImageV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccImagesImageV2_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImagesImageV2Exists(t, "openstack_images_image_v2.foo"),
				),
			},
		},
	})
}

func TestAccImagesImageV2_with_tags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckImagesImageV2Destroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccImagesImageV2_with_tags,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckImagesImageV2HasTags(t, "openstack_images_image_v2.foo"),
				),
			},
		},
	})
}

func testAccCheckImagesImageV2Destroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	imageClient, err := config.imageV2Client(OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("(testAccCheckImagesImageV2Destroy) Error creating OpenStack Image: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "openstack_images_image_v2" {
			continue
		}

		_, err := images.Get(imageClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("Image still exists")
		}
	}

	return nil
}

func testAccCheckImagesImageV2Exists(t *testing.T, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		imageClient, err := config.imageV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckImagesImageV2Destroy) Error creating OpenStack Image: %s", err)
		}

		found, err := images.Get(imageClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Image not found")
		}

		return nil
	}
}

func testAccCheckImagesImageV2HasTags(t *testing.T, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := testAccProvider.Meta().(*Config)
		imageClient, err := config.imageV2Client(OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("(testAccCheckImagesImageV2Destroy) Error creating OpenStack Image: %s", err)
		}

		found, err := images.Get(imageClient, rs.Primary.ID).Extract()
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Image not found")
		}

		tags := strings.Join(found.Tags, "")
		if tags != "foobar" && tags != "barfoo" {
			return fmt.Errorf("Image tags are %#v and should be \"foo, bar\"", found.Tags)
		}
		return nil
	}
}

var testAccImagesImageV2_basic = `
  resource "openstack_images_image_v2" "foo" {
      name   = "Rancher TerraformAccTest"
      image_source_url = "https://releases.rancher.com/os/latest/rancheros-openstack.img"
      container_format = "bare"
      disk_format = "qcow2"
  }`

var testAccImagesImageV2_with_tags = `
  resource "openstack_images_image_v2" "foo" {
      name   = "Rancher TerraformAccTest"
      image_source_url = "https://releases.rancher.com/os/latest/rancheros-openstack.img"
      container_format = "bare"
      disk_format = "qcow2"
      tags = ["foo","bar"]
  }`
