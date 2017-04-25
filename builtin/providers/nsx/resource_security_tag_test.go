package nsx


import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/sky-uk/gonsx"
	"github.com/sky-uk/gonsx/api/securitytag"
	"fmt"
)

func TestAccNSXSecurityTag_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNSXSecurityTagDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckNSXSecurityTagConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNSXSecurityTagExists("nsx_security_tag.foo"),
					resource.TestCheckResourceAttr(
						"nsx_security_tag.foo", "name", "foo"),
					resource.TestCheckResourceAttr(
						"nsx_security_tag.foo", "description", "foo"),
				),
			},
			resource.TestStep{
				Config: testAccCheckNSXSecurityTagConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNSXSecurityTagExists("nsx_security_tag.foo"),
					resource.TestCheckResourceAttr(
						"nsx_security_tag.foo", "name", "bar"),
					resource.TestCheckResourceAttr(
						"nsx_security_tag.foo", "description", "bar"),
				),
			},
		},
	})
}


func testAccCheckNSXSecurityTagDestroy(s *terraform.State) error {
	nsxclient := testAccProvider.Meta().(*gonsx.NSXClient)
	var name string
	for _, r := range s.RootModule().Resources {
		if r.Type != "nsx_security_tag" {
			continue
		}

		if name, ok := r.Primary.Attributes["name"]; ok && name == "" {
			return nil
		}
		
		api := securitytag.NewGetAll()
		err := nsxclient.Do(api)

		if err != nil {
			return err
		}

		_, err = getSingleSecurityTag(name, nsxclient)

		if err == nil {
			return fmt.Errorf("Team still exists")
		}
	}
	return nil
}

func testAccCheckNSXSecurityTagExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var name string
		nsxclient := testAccProvider.Meta().(*gonsx.NSXClient)
		for _, r := range s.RootModule().Resources {

			if name, ok := r.Primary.Attributes["name"]; ok && name == "" {
				return nil
			}
			api := securitytag.NewGetAll()
			err := nsxclient.Do(api)

			if err != nil {
				return err
			}

			_, err = getSingleSecurityTag(name, nsxclient)

			if err != nil {
				return fmt.Errorf("Received an error retrieving security tag with %s name: %s", err, name)
			}
		}
		return nil
	}
}

const testAccCheckNSXSecurityTagConfig = `
resource "nsx_security_tag" "foo" {
  name = "foo"
  desc = "foo"
}`


const testAccCheckNSXSecurityTagConfigUpdated = `
resource "nsx_security_tag" "foo" {
  name = "bar"
  desc = "bar"
}
`