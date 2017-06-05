package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMPublicIpAllocation_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Random",
			ErrCount: 1,
		},
		{
			Value:    "Static",
			ErrCount: 0,
		},
		{
			Value:    "Dynamic",
			ErrCount: 0,
		},
		{
			Value:    "STATIC",
			ErrCount: 0,
		},
		{
			Value:    "static",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validatePublicIpAllocation(tc.Value, "azurerm_public_ip")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Public IP allocation to trigger a validation error")
		}
	}
}

func TestResourceAzureRMPublicIpDomainNameLabel_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting123",
			ErrCount: 1,
		},
		{
			Value:    "testing123!",
			ErrCount: 1,
		},
		{
			Value:    "testing123-",
			ErrCount: 1,
		},
		{
			Value:    acctest.RandString(80),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validatePublicIpDomainNameLabel(tc.Value, "azurerm_public_ip")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Public IP Domain Name Label to trigger a validation error")
		}
	}
}

func TestAccAzureRMPublicIpStatic_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVPublicIpStatic_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
				),
			},
		},
	})
}

func TestAccAzureRMPublicIpStatic_disappears(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVPublicIpStatic_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
					testCheckAzureRMPublicIpDisappears("azurerm_public_ip.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureRMPublicIpStatic_idleTimeout(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVPublicIpStatic_idleTimeout, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
					resource.TestCheckResourceAttr(
						"azurerm_public_ip.test",
						"idle_timeout_in_minutes",
						"30",
					),
				),
			},
		},
	})
}

func TestAccAzureRMPublicIpStatic_withTags(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVPublicIpStatic_withTags, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVPublicIpStatic_withTagsUpdate, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
					resource.TestCheckResourceAttr(
						"azurerm_public_ip.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_public_ip.test", "tags.environment", "Production"),
					resource.TestCheckResourceAttr(
						"azurerm_public_ip.test", "tags.cost_center", "MSFT"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
					resource.TestCheckResourceAttr(
						"azurerm_public_ip.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_public_ip.test", "tags.environment", "staging"),
				),
			},
		},
	})
}

func TestAccAzureRMPublicIpStatic_update(t *testing.T) {

	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAzureRMVPublicIpStatic_basic, ri, ri)
	postConfig := fmt.Sprintf(testAccAzureRMVPublicIpStatic_update, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
				),
			},

			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
					resource.TestCheckResourceAttr(
						"azurerm_public_ip.test", "domain_name_label", "mylabel01"),
				),
			},
		},
	})
}

func TestAccAzureRMPublicIpDynamic_basic(t *testing.T) {

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMVPublicIpDynamic_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMPublicIpDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMPublicIpExists("azurerm_public_ip.test"),
				),
			},
		},
	})
}

func testCheckAzureRMPublicIpExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		availSetName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for public ip: %s", availSetName)
		}

		conn := testAccProvider.Meta().(*ArmClient).publicIPClient

		resp, err := conn.Get(resourceGroup, availSetName, "")
		if err != nil {
			return fmt.Errorf("Bad: Get on publicIPClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Public IP %q (resource group: %q) does not exist", name, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMPublicIpDisappears(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		publicIpName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for public ip: %s", publicIpName)
		}

		conn := testAccProvider.Meta().(*ArmClient).publicIPClient

		_, error := conn.Delete(resourceGroup, publicIpName, make(chan struct{}))
		err := <-error
		if err != nil {
			return fmt.Errorf("Bad: Delete on publicIPClient: %s", err)
		}

		return nil
	}
}

func testCheckAzureRMPublicIpDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).publicIPClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_public_ip" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Public IP still exists:\n%#v", resp.PublicIPAddressPropertiesFormat)
		}
	}

	return nil
}

var testAccAzureRMVPublicIpStatic_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}
`

var testAccAzureRMVPublicIpStatic_update = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
    domain_name_label = "mylabel01"
}
`

var testAccAzureRMVPublicIpStatic_idleTimeout = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
    idle_timeout_in_minutes = 30
}
`

var testAccAzureRMVPublicIpDynamic_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "dynamic"
}
`

var testAccAzureRMVPublicIpStatic_withTags = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"

    tags {
	environment = "Production"
	cost_center = "MSFT"
    }
}
`

var testAccAzureRMVPublicIpStatic_withTagsUpdate = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_public_ip" "test" {
    name = "acctestpublicip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"

    tags {
	environment = "staging"
    }
}
`
