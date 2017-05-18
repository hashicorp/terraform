package vault

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestResourceAuth(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			r.TestStep{
				Config: testResourceAuth_initialConfig,
				Check:  testResourceAuth_initialCheck,
			},
			r.TestStep{
				Config: testResourceAuth_updateConfig,
				Check:  testResourceAuth_updateCheck,
			},
		},
	})
}

var testResourceAuth_initialConfig = `

resource "vault_auth_backend" "test" {
	type = "github"
}

`

func testResourceAuth_initialCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_auth_backend.test"]
	if resourceState == nil {
		return fmt.Errorf("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return fmt.Errorf("resource has no primary instance")
	}

	name := instanceState.ID

	if name != instanceState.Attributes["type"] {
		return fmt.Errorf("id doesn't match name")
	}

	if name != "github" {
		return fmt.Errorf("unexpected auth name %s", name)
	}

	client := testProvider.Meta().(*api.Client)
	auths, err := client.Sys().ListAuth()

	if err != nil {
		return fmt.Errorf("error reading back auth: %s", err)
	}

	found := false
	for _, auth := range auths {
		if auth.Type == name {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("could not find auth backend %s in %+v", name, auths)
	}

	return nil
}

var testResourceAuth_updateConfig = `

resource "vault_auth_backend" "test" {
	type = "ldap"
}

`

func testResourceAuth_updateCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_auth_backend.test"]
	if resourceState == nil {
		return fmt.Errorf("resource not found in state")
	}

	instanceState := resourceState.Primary
	if instanceState == nil {
		return fmt.Errorf("resource has no primary instance")
	}

	name := instanceState.ID

	if name != instanceState.Attributes["type"] {
		return fmt.Errorf("id doesn't match name")
	}

	if name != "ldap" {
		return fmt.Errorf("unexpected auth name")
	}

	client := testProvider.Meta().(*api.Client)
	auths, err := client.Sys().ListAuth()

	if err != nil {
		return fmt.Errorf("error reading back auth: %s", err)
	}

	found := false
	for _, auth := range auths {
		if auth.Type == name {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("could not find auth backend %s in %+v", name, auths)
	}

	return nil
}
