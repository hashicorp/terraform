package cloudfoundry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

const infoDataResource = `

data "cf_info" "info" {}
`

func TestAccDataSourceInfo_normal(t *testing.T) {

	_, filename, _, _ := runtime.Caller(0)
	ut := os.Getenv("UNIT_TEST")
	if !testAccEnvironmentSet() || (len(ut) > 0 && ut != filepath.Base(filename)) {
		fmt.Printf("Skipping tests in '%s'.\n", filepath.Base(filename))
		return
	}

	ref := "data.cf_info.info"

	resource.Test(t,
		resource.TestCase{
			PreCheck:  func() { testAccPreCheck(t) },
			Providers: testAccProviders,
			Steps: []resource.TestStep{

				resource.TestStep{
					Config: infoDataResource,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(
							ref, "auth_endpoint", "https://login.local.pcfdev.io"),
						resource.TestCheckResourceAttr(
							ref, "uaa_endpoint", "https://uaa.local.pcfdev.io"),
						resource.TestCheckResourceAttr(
							ref, "logging_endpoint", "wss://loggregator.local.pcfdev.io:443"),
						resource.TestCheckResourceAttr(
							ref, "doppler_endpoint", "wss://doppler.local.pcfdev.io:443"),
					),
				},
			},
		})
}
