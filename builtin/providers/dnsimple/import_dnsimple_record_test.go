package dnsimple

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDnsimpleRecord_import(t *testing.T) {
	resourceName := "dnsimple_record.foobar"
	domain := os.Getenv("DNSIMPLE_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDNSimpleRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckDNSimpleRecordConfig_import, domain),
			},
			resource.TestStep{
				ResourceName:        resourceName,
				ImportState:         true,
				ImportStateVerify:   true,
				ImportStateIdPrefix: fmt.Sprintf("%s_", domain),
			},
		},
	})
}

const testAccCheckDNSimpleRecordConfig_import = `
resource "dnsimple_record" "foobar" {
	domain = "%s"

	name = "terraform"
	value = "192.168.0.10"
	type = "A"
	ttl = 3600
}`
