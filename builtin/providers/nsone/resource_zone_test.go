package nsone

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
)

func TestAccZone_basic(t *testing.T) {
	var zone nsone.Zone
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccZoneBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneState("zone", "terraform-test-zone.io"),
					testAccCheckZoneExists("nsone_zone.it", &zone),
					testAccCheckZoneTTL(&zone, 3600),
					testAccCheckZoneRefresh(&zone, 43200),
					testAccCheckZoneRetry(&zone, 7200),
					testAccCheckZoneExpiry(&zone, 1209600),
					testAccCheckZoneNXTTL(&zone, 3600),
				),
			},
		},
	})
}

func TestAccZone_updated(t *testing.T) {
	var zone nsone.Zone
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccZoneBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneState("zone", "terraform-test-zone.io"),
					testAccCheckZoneExists("nsone_zone.it", &zone),
					testAccCheckZoneTTL(&zone, 3600),
					testAccCheckZoneRefresh(&zone, 43200),
					testAccCheckZoneRetry(&zone, 7200),
					testAccCheckZoneExpiry(&zone, 1209600),
					testAccCheckZoneNXTTL(&zone, 3600),
				),
			},
			resource.TestStep{
				Config: testAccZoneUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneState("zone", "terraform-test-zone.io"),
					testAccCheckZoneExists("nsone_zone.it", &zone),
					testAccCheckZoneTTL(&zone, 10800),
					testAccCheckZoneRefresh(&zone, 3600),
					testAccCheckZoneRetry(&zone, 300),
					testAccCheckZoneExpiry(&zone, 2592000),
					testAccCheckZoneNXTTL(&zone, 3601),
				),
			},
		},
	})
}

func testAccCheckZoneState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_zone.it"]
		if !ok {
			return fmt.Errorf("Not found: %s", "nsone_zone.it")
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		p := rs.Primary
		if p.Attributes[key] != value {
			return fmt.Errorf(
				"%s != %s (actual: %s)", key, value, p.Attributes[key])
		}

		return nil
	}
}

func testAccCheckZoneExists(n string, zone *nsone.Zone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("NoID is set")
		}

		client := testAccProvider.Meta().(*nsone.APIClient)

		foundZone, err := client.GetZone(rs.Primary.Attributes["zone"])

		p := rs.Primary

		if err != nil {
			return err
		}

		if foundZone.Id != p.Attributes["id"] {
			return fmt.Errorf("Zone not found")
		}

		*zone = *foundZone

		return nil
	}
}

func testAccCheckZoneDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*nsone.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "nsone_zone" {
			continue
		}

		zone, err := client.GetZone(rs.Primary.Attributes["zone"])

		if err == nil {
			return fmt.Errorf("Record still exists: %#v: %#v", err, zone)
		}
	}

	return nil
}

func testAccCheckZoneTTL(zone *nsone.Zone, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Ttl != expected {
			return fmt.Errorf("TTL: got: %d want: %d", zone.Ttl, expected)
		}
		return nil
	}
}
func testAccCheckZoneRefresh(zone *nsone.Zone, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Refresh != expected {
			return fmt.Errorf("Refresh: got: %d want: %d", zone.Refresh, expected)
		}
		return nil
	}
}
func testAccCheckZoneRetry(zone *nsone.Zone, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Retry != expected {
			return fmt.Errorf("Retry: got: %d want: %d", zone.Retry, expected)
		}
		return nil
	}
}
func testAccCheckZoneExpiry(zone *nsone.Zone, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Expiry != expected {
			return fmt.Errorf("Expiry: got: %d want: %d", zone.Expiry, expected)
		}
		return nil
	}
}
func testAccCheckZoneNXTTL(zone *nsone.Zone, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Nx_ttl != expected {
			return fmt.Errorf("NXTTL: got: %d want: %d", zone.Nx_ttl, expected)
		}
		return nil
	}
}

const testAccZoneBasic = `
resource "nsone_zone" "it" {
  zone = "terraform-test-zone.io"
}
`

const testAccZoneUpdated = `
resource "nsone_zone" "it" {
  zone    = "terraform-test-zone.io"
  ttl     = 10800
  refresh = 3600
  retry   = 300
  expiry  = 2592000
  nx_ttl  = 3601
  # link    = "1.2.3.4.in-addr.arpa" # TODO
  # primary = "1.2.3.4.in-addr.arpa" # TODO
}
`
