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
					testAccCheckZoneState("zone", "terraform.io"),
					testAccCheckZoneExists("nsone_zone.foobar", &zone),
					testAccCheckZoneAttributes(&zone),
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
					testAccCheckZoneState("zone", "terraform.io"),
					testAccCheckZoneExists("nsone_zone.foobar", &zone),
					testAccCheckZoneAttributes(&zone),
				),
			},
			resource.TestStep{
				Config: testAccZoneUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckZoneState("zone", "terraform.io"),
					testAccCheckZoneExists("nsone_zone.foobar", &zone),
					testAccCheckZoneAttributesUpdated(&zone),
				),
			},
		},
	})
}

func testAccCheckZoneState(key, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["nsone_zone.foobar"]
		if !ok {
			return fmt.Errorf("Not found: %s", "nsone_zone.foobar")
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

		_, err := client.GetZone(rs.Primary.Attributes["zone"])

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

func testAccCheckZoneAttributes(zone *nsone.Zone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Ttl != 3600 {
			return fmt.Errorf("Bad value : %d", zone.Ttl)
		}

		if zone.Nx_ttl != 3600 {
			return fmt.Errorf("Bad value : %d", zone.Nx_ttl)
		}

		return nil
	}
}

func testAccCheckZoneAttributesUpdated(zone *nsone.Zone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if zone.Ttl != 3601 {
			return fmt.Errorf("Bad value : %d", zone.Ttl)
		}

		if zone.Nx_ttl != 3601 {
			return fmt.Errorf("Bad value : %d", zone.Nx_ttl)
		}

		return nil
	}
}

const testAccZoneBasic = `
resource "nsone_zone" "foobar" {
	zone = "terraform.io"
	hostmaster = "hostmaster@nsone.net"
	ttl = "3600"
	nx_ttl = "3600"
}`

const testAccZoneUpdated = `
resource "nsone_zone" "foobar" {
	zone = "terraform.io"
	hostmaster = "hostmaster@nsone.net"
	ttl = "3601"
	nx_ttl = "3601"
}`
