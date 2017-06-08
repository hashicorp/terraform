package azurerm

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMLoadBalancerNatRule_basic(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	natRuleName := fmt.Sprintf("NatRule-%d", ri)

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	natRule_id := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/acctestrg-%d/providers/Microsoft.Network/loadBalancers/arm-test-loadbalancer-%d/inboundNatRules/%s",
		subscriptionID, ri, ri, natRuleName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerNatRule_basic(ri, natRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRuleName, &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb_nat_rule.test", "id", natRule_id),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerNatRule_removal(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	natRuleName := fmt.Sprintf("NatRule-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerNatRule_basic(ri, natRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRuleName, &lb),
				),
			},
			{
				Config: testAccAzureRMLoadBalancerNatRule_removal(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleNotExists(natRuleName, &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerNatRule_update(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	natRuleName := fmt.Sprintf("NatRule-%d", ri)
	natRule2Name := fmt.Sprintf("NatRule-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerNatRule_multipleRules(ri, natRuleName, natRule2Name),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRuleName, &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRule2Name, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_nat_rule.test2", "frontend_port", "3390"),
					resource.TestCheckResourceAttr("azurerm_lb_nat_rule.test2", "backend_port", "3390"),
				),
			},
			{
				Config: testAccAzureRMLoadBalancerNatRule_multipleRulesUpdate(ri, natRuleName, natRule2Name),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRuleName, &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRule2Name, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_nat_rule.test2", "frontend_port", "3391"),
					resource.TestCheckResourceAttr("azurerm_lb_nat_rule.test2", "backend_port", "3391"),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerNatRule_reapply(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	natRuleName := fmt.Sprintf("NatRule-%d", ri)

	deleteNatRuleState := func(s *terraform.State) error {
		return s.Remove("azurerm_lb_nat_rule.test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerNatRule_basic(ri, natRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRuleName, &lb),
					deleteNatRuleState,
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAzureRMLoadBalancerNatRule_basic(ri, natRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRuleName, &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerNatRule_disappears(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	natRuleName := fmt.Sprintf("NatRule-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerNatRule_basic(ri, natRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerNatRuleExists(natRuleName, &lb),
					testCheckAzureRMLoadBalancerNatRuleDisappears(natRuleName, &lb),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckAzureRMLoadBalancerNatRuleExists(natRuleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerNatRuleByName(lb, natRuleName)
		if !exists {
			return fmt.Errorf("A NAT Rule with name %q cannot be found.", natRuleName)
		}

		return nil
	}
}

func testCheckAzureRMLoadBalancerNatRuleNotExists(natRuleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerNatRuleByName(lb, natRuleName)
		if exists {
			return fmt.Errorf("A NAT Rule with name %q has been found.", natRuleName)
		}

		return nil
	}
}

func testCheckAzureRMLoadBalancerNatRuleDisappears(natRuleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

		_, i, exists := findLoadBalancerNatRuleByName(lb, natRuleName)
		if !exists {
			return fmt.Errorf("A Nat Rule with name %q cannot be found.", natRuleName)
		}

		currentRules := *lb.LoadBalancerPropertiesFormat.InboundNatRules
		rules := append(currentRules[:i], currentRules[i+1:]...)
		lb.LoadBalancerPropertiesFormat.InboundNatRules = &rules

		id, err := parseAzureResourceID(*lb.ID)
		if err != nil {
			return err
		}

		_, error := conn.CreateOrUpdate(id.ResourceGroup, *lb.Name, *lb, make(chan struct{}))
		err = <-error
		if err != nil {
			return fmt.Errorf("Error Creating/Updating LoadBalancer %s", err)
		}

		_, err = conn.Get(id.ResourceGroup, *lb.Name, "")
		return err
	}
}

func testAccAzureRMLoadBalancerNatRule_basic(rInt int, natRuleName string) string {
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
}

resource "azurerm_lb_nat_rule" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  frontend_port = 3389
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}

`, rInt, rInt, rInt, rInt, natRuleName, rInt)
}

func testAccAzureRMLoadBalancerNatRule_removal(rInt int) string {
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
}
`, rInt, rInt, rInt, rInt)
}

func testAccAzureRMLoadBalancerNatRule_multipleRules(rInt int, natRuleName, natRule2Name string) string {
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
}

resource "azurerm_lb_nat_rule" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  frontend_port = 3389
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}

resource "azurerm_lb_nat_rule" "test2" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  frontend_port = 3390
  backend_port = 3390
  frontend_ip_configuration_name = "one-%d"
}

`, rInt, rInt, rInt, rInt, natRuleName, rInt, natRule2Name, rInt)
}

func testAccAzureRMLoadBalancerNatRule_multipleRulesUpdate(rInt int, natRuleName, natRule2Name string) string {
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
}

resource "azurerm_lb_nat_rule" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  frontend_port = 3389
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}

resource "azurerm_lb_nat_rule" "test2" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  frontend_port = 3391
  backend_port = 3391
  frontend_ip_configuration_name = "one-%d"
}

`, rInt, rInt, rInt, rInt, natRuleName, rInt, natRule2Name, rInt)
}
