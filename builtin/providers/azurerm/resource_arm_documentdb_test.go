package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMDocumentDbName_validation(t *testing.T) {
	str := acctest.RandString(50)
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "ab",
			ErrCount: 1,
		},
		{
			Value:    "abc",
			ErrCount: 0,
		},
		{
			Value:    str,
			ErrCount: 0,
		},
		{
			Value:    str + "a",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAzureRmDocumentDbName(tc.Value, "azurerm_documentdb")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM DocumentDB Name to trigger a validation error for '%s'", tc.Value)
		}
	}
}

func TestAccAzureRMDocumentDbMaxIntervalInSeconds_validation(t *testing.T) {
	cases := []struct {
		Value    int
		ErrCount int
	}{
		{
			Value:    0,
			ErrCount: 1,
		},
		{
			Value:    1,
			ErrCount: 0,
		},
		{
			Value:    99,
			ErrCount: 0,
		},
		{
			Value:    100,
			ErrCount: 0,
		},
		{
			Value:    101,
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateAzureRmDocumentDbMaxIntervalInSeconds(tc.Value, "azurerm_documentdb")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM DocumentDB Max Interval in Seconds to trigger a validation error")
		}
	}
}

// TODO: validation for max staleness

func TestAccAzureRMDocumentDb_standard(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDocumentDb_standard, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDocumentDbDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDocumentDbExists("azurerm_documentdb.test"),
				),
			},
		},
	})
}

func TestAccAzureRMDocumentDb_standardGeoReplicated(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMDocumentDb_standardGeoReplicated, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMDocumentDbDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMDocumentDbExists("azurerm_documentdb.test"),
				),
			},
		},
	})
}

func testCheckAzureRMDocumentDbDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).documentDbClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_documentdb" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("DocumentDB instance still exists:\n%#v", resp)
		}
	}

	return nil
}

func testCheckAzureRMDocumentDbExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for DocumentDB instance: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).documentDbClient

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on documentDbClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: DocumentDB instance %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMDocumentDb_standard = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_documentdb" "test" {
  name                = "acctest-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  offer_type          = "Standard"

  consistency_policy {
    consistency_level       = "Eventual"
    max_interval_in_seconds = 100
    max_staleness           = 30
  }

  failover_policy {
    location = "${azurerm_resource_group.test.location}"
    priority = 0
  }
}
`

var testAccAzureRMDocumentDb_standardGeoReplicated = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_documentdb" "test" {
  name                = "acctest-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  offer_type          = "Standard"

  consistency_policy {
    consistency_level       = "Eventual"
    max_interval_in_seconds = 100
    max_staleness           = 30
  }

  failover_policy {
    location = "${azurerm_resource_group.test.location}"
    priority = 0
  }

  failover_policy {
    location = "West Europe"
    priority = 1
  }
}
`
