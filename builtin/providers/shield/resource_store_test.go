package shield

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"testing"
)

func TestShieldStore_basic(t *testing.T) {
	var store Store

	testShieldStoreConfig := fmt.Sprintf(`
		resource "shield_store" "test_store" {
		  name = "Test-Store"
		  summary = "Terraform Test Store"
		  plugin = "fs"
		  endpoint = "{\"base_dir\": \"/backup_test\"}"
		}
	`)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testShieldStoreDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testShieldStoreConfig,
				Check: resource.ComposeTestCheckFunc(
					testShieldCheckStoreExists("shield_store.test_store", &store),
				),
			},
		},
	})
}

func testShieldStoreDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ShieldClient)
	rs, ok := s.RootModule().Resources["shield_store.test_store"]
	if !ok {
		return fmt.Errorf("Not found %s", "shield_store.test_store")
	}

	response, err := client.Get(fmt.Sprintf("v1/store/%s", rs.Primary.Attributes["uuid"]))

	if err != nil {
		return err
	}

	if response.StatusCode != 404 {
		return fmt.Errorf("Store still exists")
	}

	return nil
}

func testShieldCheckStoreExists(n string, store *Store) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Store UUID is set")
		}
		return nil
	}
}
