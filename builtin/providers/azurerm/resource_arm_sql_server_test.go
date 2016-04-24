package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/sql"
)

func TestAccAzureRMSqlServer_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSqlServer_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlServerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlServerExists("azurerm_sql_server.test"),
				),
			},
		},
	})
}

func TestAccAzureRMSqlServer_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMSqlServer_withTags, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMSqlServer_withTagsUpdated, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlServerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlServerExists("azurerm_sql_server.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_server.test", "tags.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlServerExists("azurerm_sql_server.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_server.test", "tags.#", "1"),
				),
			},
		},
	})
}

func testCheckAzureRMSqlServerExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetServer{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetServer: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetServer: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMSqlServerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_sql_server" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetServer{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetServer: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: SQL Server still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMSqlServer_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"
}
`

var testAccAzureRMSqlServer_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"

    tags {
    	environment = "staging"
    	database = "test"
    }
}
`

var testAccAzureRMSqlServer_withTagsUpdated = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_sql_server" "test" {
    name = "acctestsqlserver%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    version = "12.0"
    administrator_login = "mradministrator"
    administrator_login_password = "thisIsDog11"

    tags {
    	environment = "production"
    }
}
`
