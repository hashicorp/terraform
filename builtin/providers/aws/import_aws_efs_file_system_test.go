package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSEFSFileSystem_importBasic(t *testing.T) {
	resourceName := "aws_efs_file_system.foo-with-tags"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEfsFileSystemDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEFSFileSystemConfigWithTags,
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"reference_name", "creation_token"},
			},
		},
	})
}
