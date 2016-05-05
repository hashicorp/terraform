package cloudstack

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func TestAccCloudStackLoadBalancerRule_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackLoadBalancerRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", nil),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "roundrobin"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "80"),
				),
			},
		},
	})
}

func TestAccCloudStackLoadBalancerRule_update(t *testing.T) {
	var id string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackLoadBalancerRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", &id),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "roundrobin"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "80"),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", &id),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb-update"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "leastconn"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "80"),
				),
			},
		},
	})
}

func TestAccCloudStackLoadBalancerRule_forceNew(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackLoadBalancerRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", nil),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "roundrobin"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "80"),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_forcenew,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", nil),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb-update"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "leastconn"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "443"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "443"),
				),
			},
		},
	})
}

func TestAccCloudStackLoadBalancerRule_vpc(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackLoadBalancerRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_vpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", nil),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "roundrobin"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "80"),
				),
			},
		},
	})
}

func TestAccCloudStackLoadBalancerRule_vpcUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudStackLoadBalancerRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_vpc,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", nil),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "roundrobin"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "80"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "80"),
				),
			},

			resource.TestStep{
				Config: testAccCloudStackLoadBalancerRule_vpc_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudStackLoadBalancerRuleExist("cloudstack_loadbalancer_rule.foo", nil),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "name", "terraform-lb-update"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "algorithm", "leastconn"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "public_port", "443"),
					resource.TestCheckResourceAttr(
						"cloudstack_loadbalancer_rule.foo", "private_port", "443"),
				),
			},
		},
	})
}

func testAccCheckCloudStackLoadBalancerRuleExist(n string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No loadbalancer rule ID is set")
		}

		if id != nil {
			if *id != "" && *id != rs.Primary.ID {
				return fmt.Errorf("Resource ID has changed!")
			}

			*id = rs.Primary.ID
		}

		cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)
		_, count, err := cs.LoadBalancer.GetLoadBalancerRuleByID(rs.Primary.ID)

		if err != nil {
			return err
		}

		if count == 0 {
			return fmt.Errorf("Loadbalancer rule %s not found", n)
		}

		return nil
	}
}

func testAccCheckCloudStackLoadBalancerRuleDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cloudstack.CloudStackClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cloudstack_loadbalancer_rule" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Loadbalancer rule ID is set")
		}

		for k, id := range rs.Primary.Attributes {
			if !strings.Contains(k, "uuid") {
				continue
			}

			_, _, err := cs.LoadBalancer.GetLoadBalancerRuleByID(id)
			if err == nil {
				return fmt.Errorf("Loadbalancer rule %s still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

var testAccCloudStackLoadBalancerRule_basic = fmt.Sprintf(`
resource "cloudstack_instance" "foobar1" {
  name = "terraform-server1"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_loadbalancer_rule" "foo" {
  name = "terraform-lb"
  ip_address_id = "%s"
  algorithm = "roundrobin"
  public_port = 80
  private_port = 80
  member_ids = ["${cloudstack_instance.foobar1.id}"]
}
`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_PUBLIC_IPADDRESS)

var testAccCloudStackLoadBalancerRule_update = fmt.Sprintf(`
resource "cloudstack_instance" "foobar1" {
  name = "terraform-server1"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_loadbalancer_rule" "foo" {
  name = "terraform-lb-update"
  ip_address_id = "%s"
  algorithm = "leastconn"
  public_port = 80
  private_port = 80
  member_ids = ["${cloudstack_instance.foobar1.id}"]
}
`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_PUBLIC_IPADDRESS)

var testAccCloudStackLoadBalancerRule_forcenew = fmt.Sprintf(`
resource "cloudstack_instance" "foobar1" {
  name = "terraform-server1"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cloudstack_loadbalancer_rule" "foo" {
  name = "terraform-lb-update"
  ip_address_id = "%s"
  algorithm = "leastconn"
  public_port = 443
  private_port = 443
  member_ids = ["${cloudstack_instance.foobar1.id}"]
}
`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_PUBLIC_IPADDRESS)

var testAccCloudStackLoadBalancerRule_vpc = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
	name = "terraform-vpc"
	cidr = "%s"
	vpc_offering = "%s"
	zone = "%s"
}

resource "cloudstack_network" "foo" {
  name = "terraform-network"
  cidr = "%s"
  network_offering = "%s"
  vpc_id = "${cloudstack_vpc.foobar.id}"
  zone = "${cloudstack_vpc.foobar.zone}"
}

resource "cloudstack_ipaddress" "foo" {
  vpc_id = "${cloudstack_vpc.foobar.id}"
}

resource "cloudstack_instance" "foobar1" {
  name = "terraform-server1"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "${cloudstack_network.foo.id}"
  template = "%s"
  zone = "${cloudstack_network.foo.zone}"
  expunge = true
}

resource "cloudstack_loadbalancer_rule" "foo" {
  name = "terraform-lb"
  ip_address_id = "${cloudstack_ipaddress.foo.id}"
  algorithm = "roundrobin"
  network_id = "${cloudstack_network.foo.id}"
  public_port = 80
  private_port = 80
  member_ids = ["${cloudstack_instance.foobar1.id}"]
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_VPC_NETWORK_CIDR,
	CLOUDSTACK_VPC_NETWORK_OFFERING,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_TEMPLATE)

var testAccCloudStackLoadBalancerRule_vpc_update = fmt.Sprintf(`
resource "cloudstack_vpc" "foobar" {
  name = "terraform-vpc"
  cidr = "%s"
  vpc_offering = "%s"
  zone = "%s"
}

resource "cloudstack_network" "foo" {
  name = "terraform-network"
  cidr = "%s"
  network_offering = "%s"
  vpc_id = "${cloudstack_vpc.foobar.id}"
  zone = "${cloudstack_vpc.foobar.zone}"
}

resource "cloudstack_ipaddress" "foo" {
  vpc_id = "${cloudstack_vpc.foobar.id}"
}

resource "cloudstack_instance" "foobar1" {
  name = "terraform-server1"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "${cloudstack_network.foo.id}"
  template = "%s"
  zone = "${cloudstack_network.foo.zone}"
  expunge = true
}

resource "cloudstack_instance" "foobar2" {
  name = "terraform-server2"
  display_name = "terraform"
  service_offering= "%s"
  network_id = "${cloudstack_network.foo.id}"
  template = "%s"
  zone = "${cloudstack_network.foo.zone}"
  expunge = true
}

resource "cloudstack_loadbalancer_rule" "foo" {
  name = "terraform-lb-update"
  ip_address_id = "${cloudstack_ipaddress.foo.id}"
  algorithm = "leastconn"
  network_id = "${cloudstack_network.foo.id}"
  public_port = 443
  private_port = 443
  member_ids = ["${cloudstack_instance.foobar1.id}", "${cloudstack_instance.foobar2.id}"]
}`,
	CLOUDSTACK_VPC_CIDR_1,
	CLOUDSTACK_VPC_OFFERING,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_VPC_NETWORK_CIDR,
	CLOUDSTACK_VPC_NETWORK_OFFERING,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_TEMPLATE)
