package vault

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestResourcePolicy(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			r.TestStep{
				Config: testResourcePolicy_initialConfig,
				Check:  testResourcePolicy_initialCheck,
			},
			r.TestStep{
				Config: testResourcePolicy_updateConfig,
				Check:  testResourcePolicy_updateCheck,
			},
		},
	})
}

var testResourcePolicy_initialConfig = `

resource "vault_policy" "test" {
	name = "dev-team"
	policy = <<EOT
path "secret/*" {
	policy = "read"
}
EOT
}

`

func testResourcePolicy_initialCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_policy.test"]
	if resourceState == nil {
		return fmt.Errorf("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return fmt.Errorf("resource has no primary instance")
	}

	name := instanceState.ID

	if name != instanceState.Attributes["name"] {
		return fmt.Errorf("id doesn't match name")
	}

	if name != "dev-team" {
		return fmt.Errorf("unexpected policy name")
	}

	client := testProvider.Meta().(*api.Client)
	policy, err := client.Sys().GetPolicy(name)
	if err != nil {
		return fmt.Errorf("error reading back policy: %s", err)
	}

	if got, want := policy, "path \"secret/*\" {\n\tpolicy = \"read\"\n}\n"; got != want {
		return fmt.Errorf("policy data is %q; want %q", got, want)
	}

	return nil
}

var testResourcePolicy_updateConfig = `

resource "vault_policy" "test" {
	name = "dev-team"
	policy = <<EOT
path "secret/*" {
	policy = "write"
}
EOT
}

`

func testResourcePolicy_updateCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_policy.test"]
	instanceState := resourceState.Primary

	name := instanceState.ID

	client := testProvider.Meta().(*api.Client)

	if name != instanceState.Attributes["name"] {
		return fmt.Errorf("id doesn't match name")
	}

	if name != "dev-team" {
		return fmt.Errorf("unexpected policy name")
	}

	policy, err := client.Sys().GetPolicy(name)
	if err != nil {
		return fmt.Errorf("error reading back policy: %s", err)
	}

	if got, want := policy, "path \"secret/*\" {\n\tpolicy = \"write\"\n}\n"; got != want {
		return fmt.Errorf("policy data is %q; want %q", got, want)
	}

	return nil
}
