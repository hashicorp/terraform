package ns1

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func TestAccNS1Zone_Basic(t *testing.T) {
	var zone dns.Zone
	name := fmt.Sprintf("terraform.acctest-%s.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccNS1Zone_basic, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1ZoneExists("ns1_zone.foobar", &zone),
					testAccCheckNS1ZoneAttributes(&zone, name),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "zone", name),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "ttl", "3600"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "refresh", "43200"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "retry", "7200"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "expiry", "1209600"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "nx_ttl", "3600"),
				),
			},
		},
	})
}

func TestAccNS1Zone_Updated(t *testing.T) {
	var zone dns.Zone
	name := fmt.Sprintf("terraform.acctest%s.com", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNS1ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccNS1Zone_basic, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1ZoneExists("ns1_zone.foobar", &zone),
					testAccCheckNS1ZoneAttributes(&zone, name),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testAccNS1Zone_updated, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNS1ZoneExists("ns1_zone.foobar", &zone),
					testAccCheckNS1ZoneAttributesUpdated(&zone, name),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "ttl", "3601"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "refresh", "43201"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "retry", "7201"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "expiry", "1209601"),
					resource.TestCheckResourceAttr("ns1_zone.foobar", "nx_ttl", "3601"),
				),
			},
		},
	})
}

func testAccCheckNS1ZoneExists(n string, zone *dns.Zone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*ns1.Client)

		foundZone, _, err := client.Zones.Get(rs.Primary.Attributes["zone"])

		if err != nil {
			return err
		}

		if foundZone.Zone != rs.Primary.Attributes["zone"] {
			return fmt.Errorf("Zone not found")
		}

		*zone = *foundZone

		return nil
	}
}

func testAccCheckNS1ZoneDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*ns1.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ns1_zone" {
			continue
		}

		zone, _, err := client.Zones.Get(rs.Primary.Attributes["zone"])

		if err == nil {
			return fmt.Errorf("Zone still exists: %#v: %#v", err, zone)
		}
	}

	return nil
}

func testAccCheckNS1ZoneAttributes(zone *dns.Zone, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Zone != name {
			return fmt.Errorf("Bad value zone.Zone: %s", zone.Zone)
		}

		if zone.TTL != 3600 {
			return fmt.Errorf("Bad value zone.TTL: %d", zone.TTL)
		}

		if zone.NxTTL != 3600 {
			return fmt.Errorf("Bad value zone.NxTTL: %d", zone.NxTTL)
		}

		return nil
	}
}

func testAccCheckNS1ZoneAttributesUpdated(zone *dns.Zone, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Zone != name {
			return fmt.Errorf("Bad value zone.Zone: %s", zone.Zone)
		}

		if zone.TTL != 3601 {
			return fmt.Errorf("Bad value zone.TTL: %d", zone.TTL)
		}

		if zone.NxTTL != 3601 {
			return fmt.Errorf("Bad value zone.NxTTL: %d", zone.NxTTL)
		}

		return nil
	}
}

const testAccNS1Zone_basic = `
resource "ns1_zone" "foobar" {
  zone = "%s"
  ttl = 3600
  refresh = 43200
  retry = 7200
  expiry = 1209600
  nx_ttl = 3600
}`

const testAccNS1Zone_updated = `
resource "ns1_zone" "foobar" {
  zone = "%s"
  ttl = 3601
  refresh = 43201
  retry = 7201
  expiry = 1209601
  nx_ttl = 3601
}`
