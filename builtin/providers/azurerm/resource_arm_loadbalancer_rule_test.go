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

func TestResourceAzureRMLoadBalancerRuleNameLabel_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "-word",
			ErrCount: 1,
		},
		{
			Value:    "testing-",
			ErrCount: 1,
		},
		{
			Value:    "test#test",
			ErrCount: 1,
		},
		{
			Value:    acctest.RandStringFromCharSet(81, "abcdedfed"),
			ErrCount: 1,
		},
		{
			Value:    "test.rule",
			ErrCount: 0,
		},
		{
			Value:    "test_rule",
			ErrCount: 0,
		},
		{
			Value:    "test-rule",
			ErrCount: 0,
		},
		{
			Value:    "TestRule",
			ErrCount: 0,
		},
		{
			Value:    "Test123Rule",
			ErrCount: 0,
		},
		{
			Value:    "TestRule",
			ErrCount: 0,
		},
	}

	for _, tc := range cases {
		_, errors := validateArmLoadBalancerRuleName(tc.Value, "azurerm_lb_rule")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM LoadBalancer Rule Name Label to trigger a validation error")
		}
	}
}

func TestAccAzureRMLoadBalancerRule_basic(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	lbRuleName := fmt.Sprintf("LbRule-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	lbRule_id := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/acctestrg-%d/providers/Microsoft.Network/loadBalancers/arm-test-loadbalancer-%d/loadBalancingRules/%s",
		subscriptionID, ri, ri, lbRuleName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerRule_basic(ri, lbRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb_rule.test", "id", lbRule_id),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerRule_removal(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	lbRuleName := fmt.Sprintf("LbRule-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerRule_basic(ri, lbRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
				),
			},
			{
				Config: testAccAzureRMLoadBalancerRule_removal(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleNotExists(lbRuleName, &lb),
				),
			},
		},
	})
}

// https://github.com/hashicorp/terraform/issues/9424
func TestAccAzureRMLoadBalancerRule_inconsistentReads(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	backendPoolName := fmt.Sprintf("LbPool-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	lbRuleName := fmt.Sprintf("LbRule-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	probeName := fmt.Sprintf("LbProbe-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerRule_inconsistentRead(ri, backendPoolName, probeName, lbRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerBackEndAddressPoolExists(backendPoolName, &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerRule_update(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	lbRuleName := fmt.Sprintf("LbRule-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))
	lbRule2Name := fmt.Sprintf("LbRule-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	lbRuleID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/acctestrg-%d/providers/Microsoft.Network/loadBalancers/arm-test-loadbalancer-%d/loadBalancingRules/%s",
		subscriptionID, ri, ri, lbRuleName)

	lbRule2ID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/acctestrg-%d/providers/Microsoft.Network/loadBalancers/arm-test-loadbalancer-%d/loadBalancingRules/%s",
		subscriptionID, ri, ri, lbRule2Name)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerRule_multipleRules(ri, lbRuleName, lbRule2Name),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRule2Name, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test", "id", lbRuleID),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test2", "id", lbRule2ID),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test2", "frontend_port", "3390"),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test2", "backend_port", "3390"),
				),
			},
			{
				Config: testAccAzureRMLoadBalancerRule_multipleRulesUpdate(ri, lbRuleName, lbRule2Name),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRule2Name, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test", "id", lbRuleID),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test2", "id", lbRule2ID),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test2", "frontend_port", "3391"),
					resource.TestCheckResourceAttr("azurerm_lb_rule.test2", "backend_port", "3391"),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerRule_reapply(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	lbRuleName := fmt.Sprintf("LbRule-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))

	deleteRuleState := func(s *terraform.State) error {
		return s.Remove("azurerm_lb_rule.test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerRule_basic(ri, lbRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
					deleteRuleState,
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAzureRMLoadBalancerRule_basic(ri, lbRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerRule_disappears(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	lbRuleName := fmt.Sprintf("LbRule-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerRule_basic(ri, lbRuleName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerRuleExists(lbRuleName, &lb),
					testCheckAzureRMLoadBalancerRuleDisappears(lbRuleName, &lb),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckAzureRMLoadBalancerRuleExists(lbRuleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerRuleByName(lb, lbRuleName)
		if !exists {
			return fmt.Errorf("A LoadBalancer Rule with name %q cannot be found.", lbRuleName)
		}

		return nil
	}
}

func testCheckAzureRMLoadBalancerRuleNotExists(lbRuleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerRuleByName(lb, lbRuleName)
		if exists {
			return fmt.Errorf("A LoadBalancer Rule with name %q has been found.", lbRuleName)
		}

		return nil
	}
}

func testCheckAzureRMLoadBalancerRuleDisappears(ruleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

		_, i, exists := findLoadBalancerRuleByName(lb, ruleName)
		if !exists {
			return fmt.Errorf("A Rule with name %q cannot be found.", ruleName)
		}

		currentRules := *lb.LoadBalancerPropertiesFormat.LoadBalancingRules
		rules := append(currentRules[:i], currentRules[i+1:]...)
		lb.LoadBalancerPropertiesFormat.LoadBalancingRules = &rules

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

func testAccAzureRMLoadBalancerRule_basic(rInt int, lbRuleName string) string {
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

resource "azurerm_lb_rule" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  frontend_port = 3389
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}

`, rInt, rInt, rInt, rInt, lbRuleName, rInt)
}

func testAccAzureRMLoadBalancerRule_removal(rInt int) string {
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

// https://github.com/hashicorp/terraform/issues/9424
func testAccAzureRMLoadBalancerRule_inconsistentRead(rInt int, backendPoolName, probeName, lbRuleName string) string {
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

resource "azurerm_lb_backend_address_pool" "teset" {
  name                = "%s"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id     = "${azurerm_lb.test.id}"
}

resource "azurerm_lb_probe" "test" {
  name                = "%s"
  location            = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id     = "${azurerm_lb.test.id}"
  protocol            = "Tcp"
  port                = 443
}

resource "azurerm_lb_rule" "test" {
  name = "%s"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  protocol = "Tcp"
  frontend_port = 3389
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}
`, rInt, rInt, rInt, rInt, backendPoolName, probeName, lbRuleName, rInt)
}

func testAccAzureRMLoadBalancerRule_multipleRules(rInt int, lbRuleName, lbRule2Name string) string {
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

resource "azurerm_lb_rule" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Udp"
  frontend_port = 3389
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}

resource "azurerm_lb_rule" "test2" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Udp"
  frontend_port = 3390
  backend_port = 3390
  frontend_ip_configuration_name = "one-%d"
}

`, rInt, rInt, rInt, rInt, lbRuleName, rInt, lbRule2Name, rInt)
}

func testAccAzureRMLoadBalancerRule_multipleRulesUpdate(rInt int, lbRuleName, lbRule2Name string) string {
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

resource "azurerm_lb_rule" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Udp"
  frontend_port = 3389
  backend_port = 3389
  frontend_ip_configuration_name = "one-%d"
}

resource "azurerm_lb_rule" "test2" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Udp"
  frontend_port = 3391
  backend_port = 3391
  frontend_ip_configuration_name = "one-%d"
}

`, rInt, rInt, rInt, rInt, lbRuleName, rInt, lbRule2Name, rInt)
}
