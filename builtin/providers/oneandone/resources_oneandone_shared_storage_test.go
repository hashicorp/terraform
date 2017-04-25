package oneandone

import (
	"fmt"
	"testing"

	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"os"
	"time"
)

func TestAccOneandoneSharedStorage_Basic(t *testing.T) {
	var storage oneandone.SharedStorage

	name := "test_storage"
	name_updated := "test1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDOneandoneSharedStorageDestroyCheck,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneSharedStorage_basic, name),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneSharedStorageExists("oneandone_shared_storage.storage", &storage),
					testAccCheckOneandoneSharedStorageAttributes("oneandone_shared_storage.storage", name),
					resource.TestCheckResourceAttr("oneandone_shared_storage.storage", "name", name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckOneandoneSharedStorage_basic, name_updated),
				Check: resource.ComposeTestCheckFunc(
					func(*terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
					testAccCheckOneandoneSharedStorageExists("oneandone_shared_storage.storage", &storage),
					testAccCheckOneandoneSharedStorageAttributes("oneandone_shared_storage.storage", name_updated),
					resource.TestCheckResourceAttr("oneandone_shared_storage.storage", "name", name_updated),
				),
			},
		},
	})
}

func testAccCheckDOneandoneSharedStorageDestroyCheck(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneandone_shared_storage" {
			continue
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		_, err := api.GetVPN(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("VPN still exists %s %s", rs.Primary.ID, err.Error())
		}
	}

	return nil
}
func testAccCheckOneandoneSharedStorageAttributes(n string, reverse_dns string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["name"] != reverse_dns {
			return fmt.Errorf("Bad name: expected %s : found %s ", reverse_dns, rs.Primary.Attributes["name"])
		}

		return nil
	}
}

func testAccCheckOneandoneSharedStorageExists(n string, storage *oneandone.SharedStorage) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		api := oneandone.New(os.Getenv("ONEANDONE_TOKEN"), oneandone.BaseUrl)

		found_storage, err := api.GetSharedStorage(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("Error occured while fetching SharedStorage: %s", rs.Primary.ID)
		}
		if found_storage.Id != rs.Primary.ID {
			return fmt.Errorf("Record not found")
		}
		storage = found_storage

		return nil
	}
}

const testAccCheckOneandoneSharedStorage_basic = `
resource "oneandone_shared_storage" "storage" {
	name = "%s"
	description = "ttt"
	size = 50
	datacenter = "GB"
}`
