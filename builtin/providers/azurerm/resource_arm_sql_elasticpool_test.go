package azurerm

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"net/http"
	"testing"
)

func TestAccAzureRMSqlElasticPool_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSqlElasticPool_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlElasticPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlElasticPoolExists("azurerm_sql_elasticpool.test"),
				),
			},
		},
	})
}

func TestAccAzureRMSqlElasticPool_resizeDtu(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMSqlElasticPool_basic, ri)
	postConfig := fmt.Sprintf(testAccAzureRMSqlElasticPool_resizedDtu, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSqlElasticPoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlElasticPoolExists("azurerm_sql_elasticpool.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_elasticpool.test", "dtu", "50"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_elasticpool.test", "pool_size", "5000"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSqlElasticPoolExists("azurerm_sql_elasticpool.test"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_elasticpool.test", "dtu", "100"),
					resource.TestCheckResourceAttr(
						"azurerm_sql_elasticpool.test", "pool_size", "10000"),
				),
			},
		},
	})
}

func testCheckAzureRMSqlElasticPoolExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ressource, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		resourceGroup, serverName, name, err := parseArmSqlElasticPoolId(ressource.Primary.ID)
		if err != nil {
			return err
		}

		conn := testAccProvider.Meta().(*ArmClient).sqlElasticPoolsClient

		resp, err := conn.Get(resourceGroup, serverName, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on sqlElasticPoolsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: SQL Elastic Pool %q on server: %q (resource group: %q) does not exist", name, serverName, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMSqlElasticPoolDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).sqlElasticPoolsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_sql_elasticpool" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		serverName := rs.Primary.Attributes["server_name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, serverName, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("SQL Elastic Pool still exists:\n%#v", resp.ElasticPoolProperties)
		}
	}

	return nil
}

var testAccAzureRMSqlElasticPool_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctest-%[1]d"
    location = "West US"
}

resource "azurerm_sql_server" "test" {
    name = "acctest%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    version = "12.0"
    administrator_login = "4dm1n157r470r"
    administrator_login_password = "4-v3ry-53cr37-p455w0rd"
}

resource "azurerm_sql_elasticpool" "test" {
    name = "acctest-pool-%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    server_name = "${azurerm_sql_server.test.name}"
    edition = "Basic"
    dtu = 50
    pool_size = 5000
}
`

var testAccAzureRMSqlElasticPool_resizedDtu = `
resource "azurerm_resource_group" "test" {
    name = "acctest-%[1]d"
    location = "West US"
}

resource "azurerm_sql_server" "test" {
    name = "acctest%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    version = "12.0"
    administrator_login = "4dm1n157r470r"
    administrator_login_password = "4-v3ry-53cr37-p455w0rd"
}

resource "azurerm_sql_elasticpool" "test" {
    name = "acctest-pool-%[1]d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    server_name = "${azurerm_sql_server.test.name}"
    edition = "Basic"
    dtu = 100
    pool_size = 10000
}
`
