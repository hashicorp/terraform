package cloudfoundry

import (
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
	if ut != "" && ut != filepath.Base(filename) {
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
							ref, "auth-endpoint", "https://login.local.pcfdev.io"),
						resource.TestCheckResourceAttr(
							ref, "uaa-endpoint", "https://uaa.local.pcfdev.io"),
						resource.TestCheckResourceAttr(
							ref, "logging-endpoint", "wss://loggregator.local.pcfdev.io:443"),
						resource.TestCheckResourceAttr(
							ref, "doppler-endpoint", "wss://doppler.local.pcfdev.io:443"),
					),
				},
			},
		})
}
