package ultradns

import (
	"fmt"
	"testing"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccUltradnsDirpool(t *testing.T) {
	var record udnssdk.RRSet
	domain := "ultradns.phinze.com"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDirpoolCheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testCfgDirpoolMinimal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_dirpool.it", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "name", "test-dirpool-minimal"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "type", "A"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "ttl", "300"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "description", "Minimal directional pool"),
					// hashRdatas(): 10.1.0.1 -> 463398947
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.463398947.host", "10.1.0.1"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.463398947.all_non_configured", "true"),
					// Generated
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "id", "test-dirpool-minimal.ultradns.phinze.com"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "hostname", "test-dirpool-minimal.ultradns.phinze.com."),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(testCfgDirpoolMaximal, domain),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUltradnsRecordExists("ultradns_dirpool.it", &record),
					// Specified
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "zone", domain),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "name", "test-dirpool-maximal"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "type", "A"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "ttl", "300"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "description", "Description of pool"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "conflict_resolve", "GEO"),

					// hashRdatas(): 10.1.1.1 -> 442270228
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.442270228.host", "10.1.1.1"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.442270228.all_non_configured", "true"),
					// hashRdatas(): 10.1.1.2 -> 2203440046
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.2203440046.host", "10.1.1.2"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.2203440046.geo_info.0.name", "North America"),
					// hashRdatas(): 10.1.1.3 -> 4099072824
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.4099072824.host", "10.1.1.3"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "rdata.4099072824.ip_info.0.name", "some Ips"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "no_response.0.geo_info.0.name", "nrGeo"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "no_response.0.ip_info.0.name", "nrIP"),
					// Generated
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "id", "test-dirpool-maximal.ultradns.phinze.com"),
					resource.TestCheckResourceAttr("ultradns_dirpool.it", "hostname", "test-dirpool-maximal.ultradns.phinze.com."),
				),
			},
		},
	})
}

func testAccDirpoolCheckDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*udnssdk.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "ultradns_dirpool" {
			continue
		}

		k := udnssdk.RRSetKey{
			Zone: rs.Primary.Attributes["zone"],
			Name: rs.Primary.Attributes["name"],
			Type: rs.Primary.Attributes["type"],
		}

		_, err := client.RRSets.Select(k)

		if err == nil {
			return fmt.Errorf("Record still exists")
		}
	}

	return nil
}

const testCfgDirpoolMinimal = `
resource "ultradns_dirpool" "it" {
  zone        = "%s"
  name        = "test-dirpool-minimal"
  type        = "A"
  ttl         = 300
  description = "Minimal directional pool"

  rdata {
    host = "10.1.0.1"
    all_non_configured = true
  }
}
`

const testCfgDirpoolMaximal = `
resource "ultradns_dirpool" "it" {
  zone        = "%s"
  name        = "test-dirpool-maximal"
  type        = "A"
  ttl         = 300
  description = "Description of pool"

  conflict_resolve = "GEO"

  rdata {
    host               = "10.1.1.1"
    all_non_configured = true
  }

  rdata {
    host = "10.1.1.2"

    geo_info {
      name = "North America"

      codes = [
        "US-OK",
        "US-DC",
        "US-MA",
      ]
    }
  }

  rdata {
    host = "10.1.1.3"

    ip_info {
      name = "some Ips"

      ips {
        start = "200.20.0.1"
        end   = "200.20.0.10"
      }

      ips {
        cidr = "20.20.20.0/24"
      }

      ips {
        address = "50.60.70.80"
      }
    }
  }

#   rdata {
#     host = "10.1.1.4"
#
#     geo_info {
#       name             = "accountGeoGroup"
#       is_account_level = true
#     }
#
#     ip_info {
#       name             = "accountIPGroup"
#       is_account_level = true
#     }
#   }

  no_response {
    geo_info {
      name = "nrGeo"

      codes = [
        "Z4",
      ]
    }

    ip_info {
      name = "nrIP"

      ips {
        address = "197.231.41.3"
      }
    }
  }
}
`
