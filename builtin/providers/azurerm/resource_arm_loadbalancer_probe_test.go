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

func TestAccAzureRMLoadBalancerProbe_basic(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	probeName := fmt.Sprintf("probe-%d", ri)

	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	probe_id := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/acctestrg-%d/providers/Microsoft.Network/loadBalancers/arm-test-loadbalancer-%d/probes/%s",
		subscriptionID, ri, ri, probeName)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerProbe_basic(ri, probeName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
					resource.TestCheckResourceAttr(
						"azurerm_lb_probe.test", "id", probe_id),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerProbe_removal(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	probeName := fmt.Sprintf("probe-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerProbe_basic(ri, probeName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
				),
			},
			{
				Config: testAccAzureRMLoadBalancerProbe_removal(ri),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeNotExists(probeName, &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerProbe_update(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	probeName := fmt.Sprintf("probe-%d", ri)
	probe2Name := fmt.Sprintf("probe-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerProbe_multipleProbes(ri, probeName, probe2Name),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
					testCheckAzureRMLoadBalancerProbeExists(probe2Name, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_probe.test2", "port", "80"),
				),
			},
			{
				Config: testAccAzureRMLoadBalancerProbe_multipleProbesUpdate(ri, probeName, probe2Name),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
					testCheckAzureRMLoadBalancerProbeExists(probe2Name, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_probe.test2", "port", "8080"),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerProbe_updateProtocol(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	probeName := fmt.Sprintf("probe-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerProbe_updateProtocolBefore(ri, probeName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_probe.test", "protocol", "Http"),
				),
			},
			{
				Config: testAccAzureRMLoadBalancerProbe_updateProtocolAfter(ri, probeName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
					resource.TestCheckResourceAttr("azurerm_lb_probe.test", "protocol", "Tcp"),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerProbe_reapply(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	probeName := fmt.Sprintf("probe-%d", ri)

	deleteProbeState := func(s *terraform.State) error {
		return s.Remove("azurerm_lb_probe.test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerProbe_basic(ri, probeName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
					deleteProbeState,
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccAzureRMLoadBalancerProbe_basic(ri, probeName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
				),
			},
		},
	})
}

func TestAccAzureRMLoadBalancerProbe_disappears(t *testing.T) {
	var lb network.LoadBalancer
	ri := acctest.RandInt()
	probeName := fmt.Sprintf("probe-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureRMLoadBalancerProbe_basic(ri, probeName),
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMLoadBalancerExists("azurerm_lb.test", &lb),
					testCheckAzureRMLoadBalancerProbeExists(probeName, &lb),
					testCheckAzureRMLoadBalancerProbeDisappears(probeName, &lb),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testCheckAzureRMLoadBalancerProbeExists(natRuleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerProbeByName(lb, natRuleName)
		if !exists {
			return fmt.Errorf("A Probe with name %q cannot be found.", natRuleName)
		}

		return nil
	}
}

func testCheckAzureRMLoadBalancerProbeNotExists(natRuleName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, exists := findLoadBalancerProbeByName(lb, natRuleName)
		if exists {
			return fmt.Errorf("A Probe with name %q has been found.", natRuleName)
		}

		return nil
	}
}

func testCheckAzureRMLoadBalancerProbeDisappears(addressPoolName string, lb *network.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*ArmClient).loadBalancerClient

		_, i, exists := findLoadBalancerProbeByName(lb, addressPoolName)
		if !exists {
			return fmt.Errorf("A Probe with name %q cannot be found.", addressPoolName)
		}

		currentProbes := *lb.LoadBalancerPropertiesFormat.Probes
		probes := append(currentProbes[:i], currentProbes[i+1:]...)
		lb.LoadBalancerPropertiesFormat.Probes = &probes

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

func testAccAzureRMLoadBalancerProbe_basic(rInt int, probeName string) string {
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

resource "azurerm_lb_probe" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  port = 22
}
`, rInt, rInt, rInt, rInt, probeName)
}

func testAccAzureRMLoadBalancerProbe_removal(rInt int) string {
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

func testAccAzureRMLoadBalancerProbe_multipleProbes(rInt int, probeName, probe2Name string) string {
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

resource "azurerm_lb_probe" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  port = 22
}

resource "azurerm_lb_probe" "test2" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  port = 80
}
`, rInt, rInt, rInt, rInt, probeName, probe2Name)
}

func testAccAzureRMLoadBalancerProbe_multipleProbesUpdate(rInt int, probeName, probe2Name string) string {
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

resource "azurerm_lb_probe" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  port = 22
}

resource "azurerm_lb_probe" "test2" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  port = 8080
}
`, rInt, rInt, rInt, rInt, probeName, probe2Name)
}

func testAccAzureRMLoadBalancerProbe_updateProtocolBefore(rInt int, probeName string) string {
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

resource "azurerm_lb_probe" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Http"
  request_path = "/"
  port = 80
}
`, rInt, rInt, rInt, rInt, probeName)
}

func testAccAzureRMLoadBalancerProbe_updateProtocolAfter(rInt int, probeName string) string {
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

resource "azurerm_lb_probe" "test" {
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  loadbalancer_id = "${azurerm_lb.test.id}"
  name = "%s"
  protocol = "Tcp"
  port = 80
}
`, rInt, rInt, rInt, rInt, probeName)
}
