package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceAzureRMLoadBalancerPrivateIpAddressAllocation_validation(t *testing.T) {
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
		_, errors := validateLoadBalancerPrivateIpAddressAllocation(tc.Value, "azurerm_lb")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM LoadBalancer private_ip_address_allocation to trigger a validation error")
		}
	}
}

func TestAccAzureRMLoadBalancer_basic(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancer_basic(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancer_frontEndConfig(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancer_frontEndConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "frontend_ip_configuration.#", "2"),
				),
			},
			{
				Config: testAccAzureRMLoadBalancer_frontEndConfigRemoval(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "frontend_ip_configuration.#", "1"),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancer_tags(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancer_basic(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.Environment", "production"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.Purpose", "AcceptanceTests"),
				),
			},
			{
				Config: testAccAzureRMLoadBalancer_updatedTags(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"azurerm_lb.test", "tags.Purpose", "AcceptanceTests"),
				),
			},
		},
	})
}

func testCheckAzureRMLoadBalancerExists(name string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		loadbalancerName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for loadbalancer: %s", loadbalancerName)
		}

		conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

		resp, err := conn.Get(resourceGroup, loadbalancerName, "")
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("Bad: LoadBalancer %q (resource group: %q) does not exist", loadbalancerName, resourceGroup)
			}

			return fmt.Errorf("Bad: Get on loadBalancerClient: %s", err)
		}

		*lb = resp

		return nil
	}
}

func testCheckAzureRMLoadBalancerDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_lb" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name, "")

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("LoadBalancer still exists:\n%#v", resp.LoadBalancerPropertiesFormat)
		}
	}

	return nil
}

func testAccAzureRMLoadBalancer_basic(rInt int) string {
	return fmt.Sprintf(`

resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    tags {
    	Environment = "production"
    	Purpose = "AcceptanceTests"
    }

}`, rInt, rInt)
}

func testAccAzureRMLoadBalancer_updatedTags(rInt int) string {
	return fmt.Sprintf(`

resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    tags {
    	Purpose = "AcceptanceTests"
    }

}`, rInt, rInt)
}

func testAccAzureRMLoadBalancer_frontEndConfig(rInt int) string {
	return fmt.Sprintf(`

resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "test-ip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_public_ip" "test1" {
    name = "another-test-ip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    frontend_ip_configuration {
      name = "one-%d"
      public_ip_address_id = "${azurerm_public_ip.test.id}"
    }

    frontend_ip_configuration {
      name = "two-%d"
      public_ip_address_id = "${azurerm_public_ip.test1.id}"
    }
}`, rInt, rInt, rInt, rInt, rInt, rInt)
}

func testAccAzureRMLoadBalancer_frontEndConfigRemoval(rInt int) string {
	return fmt.Sprintf(`

resource "azurerm_resource_group" "test" {
    name = "acctestrg-%d"
    location = "West US"
}

resource "azurerm_public_ip" "test" {
    name = "test-ip-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"
    public_ip_address_allocation = "static"
}

resource "azurerm_lb" "test" {
    name = "arm-test-loadbalancer-%d"
    location = "West US"
    resource_group_name = "${azurerm_resource_group.test.name}"

    frontend_ip_configuration {
      name = "one-%d"
      public_ip_address_id = "${azurerm_public_ip.test.id}"
    }
}`, rInt, rInt, rInt, rInt)
}
