package pass

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourcePassword(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			r.TestStep{
				Config: testResourcePassword_initialConfig,
				Check:  testResourcePassword_initialCheck,
			},
			r.TestStep{
				Config: testResourcePassword_updateConfig,
				Check:  testResourcePassword_updateCheck,
			},
		},
	})
}

var testResourcePassword_initialConfig = `

resource "pass_password" "test" {
    path = "secret/foo"
    data = <<EOT
{"zip": "zap"}
EOT
}

`

func testResourcePassword_initialCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["pass_password.test"]
	if resourceState == nil {
		return fmt.Errorf("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return fmt.Errorf("resource has no primary instance")
	}

	path := instanceState.ID

	if path != instanceState.Attributes["path"] {
		return fmt.Errorf("id doesn't match path")
	}
	if path != "secret/foo" {
		return fmt.Errorf("unexpected secret path")
	}

	return nil
}

var testResourcePassword_updateConfig = `

resource "pass_password" "test" {
    path = "secret/foo"
    data = <<EOT
{"zip": "zoop"}
EOT
}

`

func testResourcePassword_updateCheck(s *terraform.State) error {
	return nil
}
