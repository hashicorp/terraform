package aws

import (
	"fmt"
	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"testing"
)

func TestAccAWSAMILaunchPermission_Basic(t *testing.T) {
	image_id := ""
	account_id := os.Getenv("AWS_ACCOUNT_ID")

	r.Test(t, r.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if os.Getenv("AWS_ACCOUNT_ID") == "" {
				t.Fatal("AWS_ACCOUNT_ID must be set")
			}
		},
		Providers: testAccProviders,
		Steps: []r.TestStep{
			// Scaffold everything
			r.TestStep{
				Config: testAccAWSAMILaunchPermissionConfig(account_id, true),
				Check: r.ComposeTestCheckFunc(
					testCheckResourceGetAttr("aws_ami_copy.test", "id", &image_id),
					testAccAWSAMILaunchPermissionExists(account_id, &image_id),
				),
			},
			// Drop just launch permission to test destruction
			r.TestStep{
				Config: testAccAWSAMILaunchPermissionConfig(account_id, false),
				Check: r.ComposeTestCheckFunc(
					testAccAWSAMILaunchPermissionDestroyed(account_id, &image_id),
				),
			},
		},
	})
}

func testCheckResourceGetAttr(name, key string, value *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		*value = is.Attributes[key]
		return nil
	}
}

func testAccAWSAMILaunchPermissionExists(account_id string, image_id *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasLaunchPermission(conn, *image_id, account_id); err != nil {
			return err
		} else if !has {
			return fmt.Errorf("launch permission does not exist for '%s' on '%s'", account_id, *image_id)
		}
		return nil
	}
}

func testAccAWSAMILaunchPermissionDestroyed(account_id string, image_id *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasLaunchPermission(conn, *image_id, account_id); err != nil {
			return err
		} else if has {
			return fmt.Errorf("launch permission still exists for '%s' on '%s'", account_id, *image_id)
		}
		return nil
	}
}

func testAccAWSAMILaunchPermissionConfig(account_id string, includeLaunchPermission bool) string {
	base := `
resource "aws_ami_copy" "test" {
  name = "launch-permission-test"
  description = "Launch Permission Test Copy"
  source_ami_id = "ami-7172b611"
  source_ami_region = "us-west-2"
}
`

	if !includeLaunchPermission {
		return base
	}

	return base + fmt.Sprintf(`
resource "aws_ami_launch_permission" "self-test" {
    image_id   = "${aws_ami_copy.test.id}"
    account_id = "%s"
}
`, account_id)
}
