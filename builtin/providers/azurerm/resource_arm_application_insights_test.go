package azurerm

import (
	"fmt"
	"net/http"

	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMApplicationInsightsApplicationType_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Web",
			ErrCount: 0,
		},
		{
			Value:    "Other",
			ErrCount: 0,
		},
		{
			Value:    "Random",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateEventHubNamespaceSku(tc.Value, "azurerm_eventhub_namespace")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Application Insights Application Type to trigger a validation error")
		}
	}
}

func TestAccAzureRMApplicationInsights_web(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_web, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApplicationInsightsExists("azurerm_application_insights.test"),
				),
			},
		},
	})
}

func TestAccAzureRMApplicationInsights_webWithTags(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_webWithTags, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApplicationInsightsExists("azurerm_application_insights.test"),
				),
			},
		},
	})
}

func TestAccAzureRMApplicationInsights_other(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_other, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApplicationInsightsExists("azurerm_application_insights.test"),
				),
			},
		},
	})
}

func TestAccAzureRMApplicationInsights_otherWithTags(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMApplicationInsights_otherWithTags, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMApplicationInsightsDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMApplicationInsightsExists("azurerm_application_insights.test"),
				),
			},
		},
	})
}

func testCheckAzureRMApplicationInsightsDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).applicationInsightsClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_application_insights" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Application Insights instance still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

func testCheckAzureRMApplicationInsightsExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		namespaceName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Application Insights instance: %s", namespaceName)
		}

		conn := testAccProvider.Meta().(*ArmClient).applicationInsightsClient

		resp, err := conn.Get(resourceGroup, namespaceName)
		if err != nil {
			return fmt.Errorf("Bad: Get on applicationInsightsClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Application Insights instance %q (resource group: %q) does not exist", namespaceName, resourceGroup)
		}

		return nil
	}
}

var testAccAzureRMApplicationInsights_web = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_application_insights" "test" {
  name                = "acctestaiweb-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  application_id      = "acctestaiweb-%d"
  application_type    = "web"
}
`

var testAccAzureRMApplicationInsights_webWithTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_application_insights" "test" {
  name                = "acctestaiweb-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  application_id      = "acctestaiweb-%d"
  application_type    = "web"

  tags {
    Purpose = "AcceptanceTests"
  }
}
`

var testAccAzureRMApplicationInsights_other = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}


resource "azurerm_application_insights" "test" {
  name                = "acctestaiweb-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  application_id      = "acctestaiweb-%d"
  application_type    = "other"
}
`

var testAccAzureRMApplicationInsights_otherWithTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}


resource "azurerm_application_insights" "test" {
  name                = "acctestaiweb-%d"
  location            = "${azurerm_resource_group.test.location}"
  resource_group_name = "${azurerm_resource_group.test.name}"
  application_id      = "acctestaiweb-%d"
  application_type    = "other"

  tags {
    Purpose = "AcceptanceTests"
  }
}
`
