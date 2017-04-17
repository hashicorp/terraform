package vault

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/hashicorp/vault/api"
)

func TestResourceGenericSecret(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			r.TestStep{
				Config: testResourceGenericSecret_initialConfig,
				Check:  testResourceGenericSecret_initialCheck,
			},
			r.TestStep{
				Config: testResourceGenericSecret_updateConfig,
				Check:  testResourceGenericSecret_updateCheck,
			},
		},
	})
}

var testResourceGenericSecret_initialConfig = `

resource "vault_generic_secret" "test" {
    path = "secret/foo"
    allow_read = true
    data_json = <<EOT
{
    "zip": "zap"
}
EOT
}

`

func testResourceGenericSecret_initialCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_generic_secret.test"]
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

	client := testProvider.Meta().(*api.Client)
	secret, err := client.Logical().Read(path)
	if err != nil {
		return fmt.Errorf("error reading back secret: %s", err)
	}

	if got, want := secret.Data["zip"], "zap"; got != want {
		return fmt.Errorf("'zip' data is %q; want %q", got, want)
	}

	return nil
}

var testResourceGenericSecret_updateConfig = `

resource "vault_generic_secret" "test" {
    path = "secret/foo"
    allow_read = true
    data_json = <<EOT
{
    "zip": "zoop"
}
EOT
}

`

func testResourceGenericSecret_updateCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_generic_secret.test"]
	instanceState := resourceState.Primary

	path := instanceState.ID

	client := testProvider.Meta().(*api.Client)
	secret, err := client.Logical().Read(path)
	if err != nil {
		return fmt.Errorf("error reading back secret: %s", err)
	}

	if got, want := secret.Data["zip"], "zoop"; got != want {
		return fmt.Errorf("'zip' data is %q; want %q", got, want)
	}

	return nil
}
