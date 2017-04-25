package alicloud

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAlicloudImagesDataSource_images(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudImagesDataSourceImagesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_images.multi_image"),

					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.#", "2"),

					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.architecture", "x86_64"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.disk_device_mappings.#", "0"),
					resource.TestMatchResourceAttr("data.alicloud_images.multi_image", "images.0.creation_time", regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestMatchResourceAttr("data.alicloud_images.multi_image", "images.0.image_id", regexp.MustCompile("^centos_6\\w{1,5}[64]{1}.")),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.image_owner_alias", "system"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.os_type", "linux"),
					resource.TestMatchResourceAttr("data.alicloud_images.multi_image", "images.0.name", regexp.MustCompile("^centos_6[a-zA-Z0-9_]{1,5}[64]{1}.")),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.progress", "100%"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.state", "Available"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.status", "Available"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.usage", "instance"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.0.tags.%", "0"),

					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.architecture", "i386"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.disk_device_mappings.#", "0"),
					resource.TestMatchResourceAttr("data.alicloud_images.multi_image", "images.1.creation_time", regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestMatchResourceAttr("data.alicloud_images.multi_image", "images.1.image_id", regexp.MustCompile("^centos_6[a-zA-Z0-9_]{1,5}[32]{1}.")),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.image_owner_alias", "system"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.os_type", "linux"),
					resource.TestMatchResourceAttr("data.alicloud_images.multi_image", "images.1.name", regexp.MustCompile("^centos_6\\w{1,5}[32]{1}.")),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.progress", "100%"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.state", "Available"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.status", "Available"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.usage", "instance"),
					resource.TestCheckResourceAttr("data.alicloud_images.multi_image", "images.1.tags.%", "0"),
				),
			},
		},
	})
}

func TestAccAlicloudImagesDataSource_owners(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudImagesDataSourceOwnersConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_images.owners_filtered_image"),
				),
			},
		},
	})
}

func TestAccAlicloudImagesDataSource_ownersEmpty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudImagesDataSourceEmptyOwnersConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_images.empty_owners_filtered_image"),
					resource.TestCheckResourceAttr("data.alicloud_images.empty_owners_filtered_image", "most_recent", "true"),
				),
			},
		},
	})
}

func TestAccAlicloudImagesDataSource_nameRegexFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudImagesDataSourceNameRegexConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_images.name_regex_filtered_image"),
					resource.TestMatchResourceAttr("data.alicloud_images.name_regex_filtered_image", "images.0.image_id", regexp.MustCompile("^centos_")),
				),
			},
		},
	})
}

func TestAccAlicloudImagesDataSource_imageNotInFirstPage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAlicloudImagesDataSourceImageNotInFirstPageConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudDataSourceID("data.alicloud_images.name_regex_filtered_image"),
					resource.TestMatchResourceAttr("data.alicloud_images.name_regex_filtered_image", "images.0.image_id", regexp.MustCompile("^ubuntu_14")),
				),
			},
		},
	})
}

// Instance store test - using centos images
const testAccCheckAlicloudImagesDataSourceImagesConfig = `
data "alicloud_images" "multi_image" {
	owners = "system"
	name_regex = "^centos_6"
}
`

// Testing owner parameter
const testAccCheckAlicloudImagesDataSourceOwnersConfig = `
data "alicloud_images" "owners_filtered_image" {
	most_recent = true
	owners = "system"
}
`

const testAccCheckAlicloudImagesDataSourceEmptyOwnersConfig = `
data "alicloud_images" "empty_owners_filtered_image" {
	most_recent = true
	owners = ""
}
`

// Testing name_regex parameter
const testAccCheckAlicloudImagesDataSourceNameRegexConfig = `
data "alicloud_images" "name_regex_filtered_image" {
	most_recent = true
	owners = "system"
	name_regex = "^centos_6\\w{1,5}[64]{1}.*"
}
`

// Testing image not in first page response
const testAccCheckAlicloudImagesDataSourceImageNotInFirstPageConfig = `
data "alicloud_images" "name_regex_filtered_image" {
	most_recent = true
	owners = "system"
	name_regex = "^ubuntu_14.*_64"
}
`
