package triton

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/joyent/gosdc/cloudapi"
)

func TestAccTritonKey_basic(t *testing.T) {
	keyName := fmt.Sprintf("acctest-%d", acctest.RandInt())
	config := fmt.Sprintf(testAccTritonKey_basic, keyName, testAccTritonKey_basicMaterial)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckTritonKeyDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckTritonKeyExists("triton_key.test"),
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
		},
	})
}

func testCheckTritonKeyExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		conn := testAccProvider.Meta().(*cloudapi.Client)

		rule, err := conn.GetKey(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Bad: Check Key Exists: %s", err)
		}

		if rule == nil {
			return fmt.Errorf("Bad: Key %q does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testCheckTritonKeyDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*cloudapi.Client)

	return resource.Retry(1*time.Minute, func() *resource.RetryError {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "triton_key" {
				continue
			}

			resp, err := conn.GetKey(rs.Primary.ID)
			if err != nil {
				return nil
			}

			if resp != nil {
				return resource.RetryableError(fmt.Errorf("Bad: Key %q still exists", rs.Primary.ID))
			}
		}

		return nil
	})
}

var testAccTritonKey_basic = `
resource "triton_key" "test" {
    name = "%s"
    key = "%s"
}
`

const testAccTritonKey_basicMaterial = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDL18KJIe8N7FxcgOMtabo10qZEDyYUSlOpsh/EYrugQCQHMKuNytog1lhFNZNk4LGNAz5L8/btG9+/axY/PfundbjR3SXt0hupAGQIVHuygWTr7foj5iGhckrEM+r3eMCXqoCnIFLhDZLDcq/zN2MxNbqDKcWSYmc8ul9dZWuiQpKOL+0nNXjhYA8Ewu+07kVAtsZD0WfvnAUjxmYb3rB15eBWk7gLxHrOPfZpeDSvOOX2bmzikpLn+L5NKrJsLrzO6hU/rpxD4OTHLULcsnIts3lYH8hShU8uY5ry94PBzdix++se3pUGvNSe967fKlHw3Ymh9nE/LJDQnzTNyFMj James@jn-mpb13`
