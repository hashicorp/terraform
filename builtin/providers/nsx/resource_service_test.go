package nsx

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/sky-uk/gonsx"
	"github.com/sky-uk/gonsx/api/service"
	"testing"
)

func TestAccNSXService_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNSXServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckNSXServiceConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckNSXServiceExists("nsx_service.foo"),
					resource.TestCheckResourceAttr(
						"nsx_service.foo", "name", "foo_service"),
					resource.TestCheckResourceAttr(
						"nsx_service.foo", "description", "foo service description"),
				),
			},
			resource.TestStep{
				Config: testAccCheckNSXServiceConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testCheckNSXServiceExists("nsx_service.foo"),
					resource.TestCheckResourceAttr(
						"nsx_service.foo", "name", "bar_service"),
					resource.TestCheckResourceAttr(
						"nsx_service.foo", "description", "bar service description"),
					resource.TestCheckResourceAttr(
						"nsx_service.foo", "protocol", "UDP"),
					resource.TestCheckResourceAttr(
						"nsx_service.foo", "ports", "81"),
				),
			},
		},
	})
}

func testAccCheckNSXServiceDestroy(s *terraform.State) error {
	nsxclient := testAccProvider.Meta().(*gonsx.NSXClient)
	var name, scopeid string
	for _, r := range s.RootModule().Resources {
		if r.Type != "nsx_service" {
			continue
		}

		if name, ok := r.Primary.Attributes["name"]; ok && name == "" {
			return nil
		}

		if scopeid, ok := r.Primary.Attributes["scopeid"]; ok && scopeid == "" {
			return nil
		}

		api := service.NewGetAll(scopeid)
		err := nsxclient.Do(api)

		if err != nil {
			return err
		}

		_, err = getSingleService(scopeid, name, nsxclient)

		if err == nil {
			return fmt.Errorf("Team still exists")
		}
	}
	return nil
}

func testCheckNSXServiceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		nsxClient := testAccProvider.Meta().(*gonsx.NSXClient)
		serviceScopeID := rs.Primary.Attributes["scopeid"]
		serviceName := rs.Primary.Attributes["name"]

		_, err := getSingleService(serviceScopeID, serviceName, nsxClient)
		if err != nil {
			return fmt.Errorf("Received an error retrieving service with name: %s, %s", serviceName, err)
		}

		return nil
	}
}

func testCheckNSXServiceNotExist(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Service ID is set")
		}

		nsxClient := testAccProvider.Meta().(*gonsx.NSXClient)
		serviceScopeID := rs.Primary.Attributes["scopeid"]
		serviceName := rs.Primary.Attributes["name"]

		_, err := getSingleService(serviceScopeID, serviceName, nsxClient)
		if err == nil {
			return fmt.Errorf("Service resource with name '%s' should not exist, but found the resource.", serviceName)
		}
		return nil

	}
}

const testAccCheckNSXServiceConfig = `
resource "nsx_service" "foo" {
  name = "foo_service"
  scopeid = "globalroot-0"
  description = "foo service description"
  protocol = "TCP"
  ports = "80"
}`

const testAccCheckNSXServiceConfigUpdated = `
resource "nsx_service" "foo" {
  name = "bar_service"
  scopeid = "globalroot-0"
  description = "bar service description"
  protocol = "UDP"
  ports = "81"
}`
