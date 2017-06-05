package vault

import (
	"fmt"
	"testing"

	r "github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/vault/api"
)

func TestResourceMount(t *testing.T) {
	r.Test(t, r.TestCase{
		Providers: testProviders,
		PreCheck:  func() { testAccPreCheck(t) },
		Steps: []r.TestStep{
			{
				Config: testResourceMount_initialConfig,
				Check:  testResourceMount_initialCheck,
			},
			{
				Config: testResourceMount_updateConfig,
				Check:  testResourceMount_updateCheck,
			},
		},
	})
}

var testResourceMount_initialConfig = `

resource "vault_mount" "test" {
	path = "example"
	type = "generic"
	description = "Example mount for testing"
	default_lease_ttl_seconds = 3600
	max_lease_ttl_seconds = 36000
}

`

func testResourceMount_initialCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_mount.test"]
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

	if path != "example" {
		return fmt.Errorf("unexpected path value")
	}

	mount, err := findMount(path)
	if err != nil {
		return fmt.Errorf("error reading back mount: %s", err)
	}

	if wanted := "Example mount for testing"; mount.Description != wanted {
		return fmt.Errorf("description is %v; wanted %v", mount.Description, wanted)
	}

	if wanted := "generic"; mount.Type != wanted {
		return fmt.Errorf("type is %v; wanted %v", mount.Description, wanted)
	}

	if wanted := 3600; mount.Config.DefaultLeaseTTL != wanted {
		return fmt.Errorf("default lease ttl is %v; wanted %v", mount.Description, wanted)
	}

	if wanted := 36000; mount.Config.MaxLeaseTTL != wanted {
		return fmt.Errorf("max lease ttl is %v; wanted %v", mount.Description, wanted)
	}

	return nil
}

var testResourceMount_updateConfig = `

resource "vault_mount" "test" {
	path = "remountingExample"
	type = "generic"
	description = "Example mount for testing"
	default_lease_ttl_seconds = 7200
	max_lease_ttl_seconds = 72000
}

`

func testResourceMount_updateCheck(s *terraform.State) error {
	resourceState := s.Modules[0].Resources["vault_mount.test"]
	instanceState := resourceState.Primary

	path := instanceState.ID

	if path != instanceState.Attributes["path"] {
		return fmt.Errorf("id doesn't match path")
	}

	if path != "remountingExample" {
		return fmt.Errorf("unexpected path value")
	}

	mount, err := findMount(path)
	if err != nil {
		return fmt.Errorf("error reading back mount: %s", err)
	}

	if wanted := "Example mount for testing"; mount.Description != wanted {
		return fmt.Errorf("description is %v; wanted %v", mount.Description, wanted)
	}

	if wanted := "generic"; mount.Type != wanted {
		return fmt.Errorf("type is %v; wanted %v", mount.Description, wanted)
	}

	if wanted := 7200; mount.Config.DefaultLeaseTTL != wanted {
		return fmt.Errorf("default lease ttl is %v; wanted %v", mount.Description, wanted)
	}

	if wanted := 72000; mount.Config.MaxLeaseTTL != wanted {
		return fmt.Errorf("max lease ttl is %v; wanted %v", mount.Description, wanted)
	}

	return nil
}

func findMount(path string) (*api.MountOutput, error) {
	client := testProvider.Meta().(*api.Client)

	path = path + "/"

	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return nil, err
	}

	if mounts[path] != nil {
		return mounts[path], nil
	}

	return nil, fmt.Errorf("Unable to find mount %s in Vault; current list: %v", path, mounts)
}
