package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMCdnProfileSKU_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Standard_Verizon",
			ErrCount: 0,
		},
		{
			Value:    "Premium_Verizon",
			ErrCount: 0,
		},
		{
			Value:    "Standard_Akamai",
			ErrCount: 0,
		},
		{
			Value:    "STANDARD_AKAMAI",
			ErrCount: 0,
		},
		{
			Value:    "standard_akamai",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateCdnProfileSku(tc.Value, "azurerm_cdn_profile")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM CDN Profile SKU to trigger a validation error")
		}
	}
}

func TestAccAzureRMCdnProfile_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMCdnProfile_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMCdnProfileDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMCdnProfileExists("azurerm_cdn_profile.test"),
				),
			},
		},
	})
}

func TestAccAzureRMCdnProfile_withTags(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMCdnProfile_withTags, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMCdnProfile_withTagsUpdate, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMCdnProfileDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMCdnProfileExists("azurerm_cdn_profile.test"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_profile.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_profile.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_profile.test", "tags.cost_center", "MSFT"),
				),
			},

			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMCdnProfileExists("azurerm_cdn_profile.test"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_profile.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_cdn_profile.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func TestAccAzureRMCdnProfile_NonStandardCasing(t *testing.T) {

	ri := acctest.RandInt()
	config := testAccAzureRMCdnProfileNonStandardCasing(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMCdnProfileDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMCdnProfileExists("azurerm_cdn_profile.test"),
				),
			},

			resource.TestStep{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testCheckAzureRMCdnProfileExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for cdn profile: %s", name)
		}

		conn := testAccProvider.Meta().(*ArmClient).cdnProfilesClient

		resp, err := conn.Get(resourceGroup, name)
		if err != nil {
			return fmt.Errorf("Bad: Get on cdnProfilesClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: CDN Profile %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMCdnProfileDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).cdnProfilesClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_cdn_profile" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("CDN Profile still exists:\n%#v", resp.ProfileProperties)
		}
	}

	return nil
}

var testAccAzureRMCdnProfile_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_cdn_profile" "test" {
    name = "acctestcdnprof%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "Standard_Verizon"
}
`

var testAccAzureRMCdnProfile_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_cdn_profile" "test" {
    name = "acctestcdnprof%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "Standard_Verizon"

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMCdnProfile_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_cdn_profile" "test" {
    name = "acctestcdnprof%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "Standard_Verizon"

    tags {
	environment = "staging"
    }
}
`

func testAccAzureRMCdnProfileNonStandardCasing(ri int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_cdn_profile" "test" {
    name = "acctestcdnprof%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    sku = "standard_verizon"
}
`, ri, ri)
}
