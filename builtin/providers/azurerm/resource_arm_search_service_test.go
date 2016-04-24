package azurerm

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/jen20/riviera/search"
)

func TestAccAzureRMSearchService_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMSearchService_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSearchServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSearchServiceExists("azurerm_search_service.test"),
					resource.TestCheckResourceAttr(
						"azurerm_search_service.test", "tags.#", "2"),
				),
			},
		},
	})
}

func TestAccAzureRMSearchService_updateReplicaCountAndTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMSearchService_basic, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMSearchService_updated, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMSearchServiceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSearchServiceExists("azurerm_search_service.test"),
					resource.TestCheckResourceAttr(
						"azurerm_search_service.test", "tags.#", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_search_service.test", "replica_count", "1"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMSearchServiceExists("azurerm_search_service.test"),
					resource.TestCheckResourceAttr(
						"azurerm_search_service.test", "tags.#", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_search_service.test", "replica_count", "2"),
				),
			},
		},
	})
}

func testCheckAzureRMSearchServiceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).rivieraClient

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &search.GetSearchService{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetSearchService: %s", err)
		}
		if !readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: GetSearchService: %s", readResponse.Error)
		}

		return nil
	}
}

func testCheckAzureRMSearchServiceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).rivieraClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_search_service" {
			continue
		}

		readRequest := conn.NewRequestForURI(rs.Primary.ID)
		readRequest.Command = &search.GetSearchService{}

		readResponse, err := readRequest.Execute()
		if err != nil {
			return fmt.Errorf("Bad: GetSearchService: %s", err)
		}

		if readResponse.IsSuccessful() {
			return fmt.Errorf("Bad: Search Service still exists: %s", readResponse.Error)
		}
	}

	return nil
}

var testAccAzureRMSearchService_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_search_service" "test" {
    name = "acctestsearchservice%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    sku = "standard"

    tags {
    	environment = "staging"
    	database = "test"
    }
}
`

var testAccAzureRMSearchService_updated = `
resource "azurerm_resource_group" "test" {
    name = "acctest_rg_%d"
    location = "West US"
}
resource "azurerm_search_service" "test" {
    name = "acctestsearchservice%d"
    resource_group_name = "${azurerm_resource_group.test.name}"
    location = "West US"
    sku = "standard"
    replica_count = 2

    tags {
    	environment = "production"
    }
}
`
