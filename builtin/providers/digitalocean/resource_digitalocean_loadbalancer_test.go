package digitalocean

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccDigitalOceanLoadbalancer_Basic(t *testing.T) {
	var loadbalancer godo.LoadBalancer
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanLoadbalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanLoadbalancerConfig_basic(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanLoadbalancerExists("digitalocean_loadbalancer.foobar", &loadbalancer),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "name", fmt.Sprintf("loadbalancer-%d", rInt)),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "region", "nyc3"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_port", "80"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_port", "80"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.port", "22"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "droplet_ids.#", "1"),
				),
			},
		},
	})
}

func TestAccDigitalOceanLoadbalancer_Updated(t *testing.T) {
	var loadbalancer godo.LoadBalancer
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanLoadbalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanLoadbalancerConfig_basic(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanLoadbalancerExists("digitalocean_loadbalancer.foobar", &loadbalancer),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "name", fmt.Sprintf("loadbalancer-%d", rInt)),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "region", "nyc3"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_port", "80"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_port", "80"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.port", "22"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "droplet_ids.#", "1"),
				),
			},
			{
				Config: testAccCheckDigitalOceanLoadbalancerConfig_updated(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanLoadbalancerExists("digitalocean_loadbalancer.foobar", &loadbalancer),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "name", fmt.Sprintf("loadbalancer-%d", rInt)),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "region", "nyc3"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_port", "81"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_port", "81"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.port", "22"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "droplet_ids.#", "2"),
				),
			},
		},
	})
}

func TestAccDigitalOceanLoadbalancer_dropletTag(t *testing.T) {
	var loadbalancer godo.LoadBalancer
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDigitalOceanLoadbalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDigitalOceanLoadbalancerConfig_dropletTag(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDigitalOceanLoadbalancerExists("digitalocean_loadbalancer.foobar", &loadbalancer),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "name", fmt.Sprintf("loadbalancer-%d", rInt)),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "region", "nyc3"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_port", "80"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.entry_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_port", "80"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "forwarding_rule.0.target_protocol", "http"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.#", "1"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.port", "22"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "healthcheck.0.protocol", "tcp"),
					resource.TestCheckResourceAttr(
						"digitalocean_loadbalancer.foobar", "droplet_tag", "sample"),
				),
			},
		},
	})
}

func testAccCheckDigitalOceanLoadbalancerDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*godo.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "digitalocean_loadbalancer" {
			continue
		}

		_, _, err := client.LoadBalancers.Get(context.Background(), rs.Primary.ID)

		if err != nil && !strings.Contains(err.Error(), "404") {
			return fmt.Errorf(
				"Error waiting for loadbalancer (%s) to be destroyed: %s",
				rs.Primary.ID, err)
		}
	}

	return nil
}

func testAccCheckDigitalOceanLoadbalancerExists(n string, loadbalancer *godo.LoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Loadbalancer ID is set")
		}

		client := testAccProvider.Meta().(*godo.Client)

		lb, _, err := client.LoadBalancers.Get(context.Background(), rs.Primary.ID)

		if err != nil {
			return err
		}

		if lb.ID != rs.Primary.ID {
			return fmt.Errorf("Loabalancer not found")
		}

		*loadbalancer = *lb

		return nil
	}
}

func testAccCheckDigitalOceanLoadbalancerConfig_basic(rInt int) string {
	return fmt.Sprintf(`
resource "digitalocean_droplet" "foobar" {
  name      = "foo-%d"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
}

resource "digitalocean_loadbalancer" "foobar" {
  name = "loadbalancer-%d"
  region = "nyc3"

  forwarding_rule {
    entry_port = 80
    entry_protocol = "http"

    target_port = 80
    target_protocol = "http"
  }

  healthcheck {
    port = 22
    protocol = "tcp"
  }

  droplet_ids = ["${digitalocean_droplet.foobar.id}"]
}`, rInt, rInt)
}

func testAccCheckDigitalOceanLoadbalancerConfig_updated(rInt int) string {
	return fmt.Sprintf(`
resource "digitalocean_droplet" "foobar" {
  name      = "foo-%d"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
}

resource "digitalocean_droplet" "foo" {
  name      = "foo-%d"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
}

resource "digitalocean_loadbalancer" "foobar" {
  name = "loadbalancer-%d"
  region = "nyc3"

  forwarding_rule {
    entry_port = 81
    entry_protocol = "http"

    target_port = 81
    target_protocol = "http"
  }

  healthcheck {
    port = 22
    protocol = "tcp"
  }

  droplet_ids = ["${digitalocean_droplet.foobar.id}","${digitalocean_droplet.foo.id}"]
}`, rInt, rInt, rInt)
}

func testAccCheckDigitalOceanLoadbalancerConfig_dropletTag(rInt int) string {
	return fmt.Sprintf(`
resource "digitalocean_tag" "barbaz" {
  name = "sample"
}

resource "digitalocean_droplet" "foobar" {
  name      = "foo-%d"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
  tags = ["${digitalocean_tag.barbaz.id}"]
}

resource "digitalocean_loadbalancer" "foobar" {
  name = "loadbalancer-%d"
  region = "nyc3"

  forwarding_rule {
    entry_port = 80
    entry_protocol = "http"

    target_port = 80
    target_protocol = "http"
  }

  healthcheck {
    port = 22
    protocol = "tcp"
  }

  droplet_tag = "${digitalocean_tag.barbaz.name}"

  depends_on = ["digitalocean_droplet.foobar"]
}`, rInt, rInt)
}
