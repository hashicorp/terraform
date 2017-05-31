package ad

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"gopkg.in/ldap.v2"
	"testing"
)

func TestAccAdComputer_Basic(t *testing.T) {
	computer_name := "code1"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAdComputerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckAdComputerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAdComputerExists("ad_resourceComputer.test", computer_name),
					resource.TestCheckResourceAttr(
						"ad_resourceComputer.test", "computer_name", "code1"),
					resource.TestCheckResourceAttr(
						"ad_resourceComputer.test", "domain", "terraform.local"),
				),
			},
		},
	})
}

func testAccCheckAdComputerDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ldap.Conn)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ad_resourceComputer" {
			continue
		}

		searchRequest := ldap.NewSearchRequest(
			"cn=code1,cn=Computers,dc=terraform,dc=local", // The base dn to search
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			"(&(objectClass=Computer))", // The filter to apply
			[]string{"dn", "cn"},        // A list attributes to retrieve
			nil,
		)
		_, err := client.Search(searchRequest)

		if err == nil {
			return fmt.Errorf("Computer still exists")
		}
	}

	return nil
}

func testAccCheckAdComputerExists(n, computer_name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No AD Computer ID is set")
		}
		client := testAccProvider.Meta().(*ldap.Conn)
		dn := "cn=" + computer_name + ",cn=Computers,dc=terraform,dc=local"
		searchRequest := ldap.NewSearchRequest(
			dn, // The base dn to search
			ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
			"(&(objectClass=Computer))", // The filter to apply
			[]string{"dn", "cn"},        // A list attributes to retrieve
			nil,
		)
		_, err := client.Search(searchRequest)

		if err != nil {
			return err
		}
		return nil
	}
}

const testAccCheckAdComputerConfig_basic = `
resource "ad_resourceComputer" "test"{
	domain = "terraform.local"
	computer_name = "code1"
}`
