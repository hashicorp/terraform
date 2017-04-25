package aws

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSAMILaunchPermission_Basic(t *testing.T) {
	imageID := ""
	accountID := os.Getenv("AWS_ACCOUNT_ID")

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
				Config: testAccAWSAMILaunchPermissionConfig(accountID, true),
				Check: r.ComposeTestCheckFunc(
					testCheckResourceGetAttr("aws_ami_copy.test", "id", &imageID),
					testAccAWSAMILaunchPermissionExists(accountID, &imageID),
				),
			},
			// Drop just launch permission to test destruction
			r.TestStep{
				Config: testAccAWSAMILaunchPermissionConfig(accountID, false),
				Check: r.ComposeTestCheckFunc(
					testAccAWSAMILaunchPermissionDestroyed(accountID, &imageID),
				),
			},
			// Re-add everything so we can test when AMI disappears
			r.TestStep{
				Config: testAccAWSAMILaunchPermissionConfig(accountID, true),
				Check: r.ComposeTestCheckFunc(
					testCheckResourceGetAttr("aws_ami_copy.test", "id", &imageID),
					testAccAWSAMILaunchPermissionExists(accountID, &imageID),
				),
			},
			// Here we delete the AMI to verify the follow-on refresh after this step
			// should not error.
			r.TestStep{
				Config: testAccAWSAMILaunchPermissionConfig(accountID, true),
				Check: r.ComposeTestCheckFunc(
					testAccAWSAMIDisappears(&imageID),
				),
				ExpectNonEmptyPlan: true,
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

func testAccAWSAMILaunchPermissionExists(accountID string, imageID *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasLaunchPermission(conn, *imageID, accountID); err != nil {
			return err
		} else if !has {
			return fmt.Errorf("launch permission does not exist for '%s' on '%s'", accountID, *imageID)
		}
		return nil
	}
}

func testAccAWSAMILaunchPermissionDestroyed(accountID string, imageID *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		if has, err := hasLaunchPermission(conn, *imageID, accountID); err != nil {
			return err
		} else if has {
			return fmt.Errorf("launch permission still exists for '%s' on '%s'", accountID, *imageID)
		}
		return nil
	}
}

// testAccAWSAMIDisappears is technically a "test check function" but really it
// exists to perform a side effect of deleting an AMI out from under a resource
// so we can test that Terraform will react properly
func testAccAWSAMIDisappears(imageID *string) r.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).ec2conn
		req := &ec2.DeregisterImageInput{
			ImageId: aws.String(*imageID),
		}

		_, err := conn.DeregisterImage(req)
		if err != nil {
			return err
		}

		if err := resourceAwsAmiWaitForDestroy(*imageID, conn); err != nil {
			return err
		}
		return nil
	}
}

func testAccAWSAMILaunchPermissionConfig(accountID string, includeLaunchPermission bool) string {
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
`, accountID)
}
