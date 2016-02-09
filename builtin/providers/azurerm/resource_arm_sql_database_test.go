package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/sql"
)

func TestResourceAzureRMSqlDatabaseEdition_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Basic",
			ErrCount: 0,
		},
		{
			Value:    "Standard",
			ErrCount: 0,
		},
		{
			Value:    "Premium",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmSqlDatabaseEdition(tc.Value, "azurerm_sql_database")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM SQL Database edition to trigger a validation error")
		}
	}
}

func TestAccAzureRMSqlDatabase_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSqlDatabase_basic, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlDatabaseDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
				),
			},
		},
	})
}

func TestAccAzureRMSqlDatabase_withTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMSqlDatabase_withTags, ri, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMSqlDatabase_withTagsUpdate, ri, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlDatabaseDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_database.test", "tags.#", "2"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlDatabaseExists("azurerm_sql_database.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_database.test", "tags.#", "1"),
				),
			},
		},
	})
}

func testCheckAzureRMSqlDatabaseExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetDatabase{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetDatabase: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetDatabase: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMSqlDatabaseDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_sql_database" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &sql.GetDatabase{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetDatabase: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: SQL Database still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMSqlDatabase_basic = `
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

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "West US"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    requested_service_objective_name = "S0"
}
`

var testAccAzureRMSqlDatabase_withTags = `
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

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "West US"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    requested_service_objective_name = "S0"

    tags {
    	environment = "staging"
    	database = "test"
    }
}
`

var testAccAzureRMSqlDatabase_withTagsUpdate = `
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

resource "azurerm_sql_database" "test" {
    name = "acctestdb%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    server_name = "${azurerm_sql_server.test.name}"
    location = "West US"
    edition = "Standard"
    collation = "SQL_Latin1_General_CP1_CI_AS"
    max_size_bytes = "1073741824"
    requested_service_objective_name = "S0"

    tags {
    	environment = "production"
    }
}
`
