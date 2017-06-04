package cloudflare

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccCloudFlareRecord_importBasic(t *testing.T) {
	domain := os.Getenv("CLOUDFLARE_DOMAIN")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFlareRecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckCloudFlareRecordConfigBasic, domain),
			},
			{
				ResourceName:            "cloudflare_record.foobar",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config_vars"},
			},
		},
	})
}
