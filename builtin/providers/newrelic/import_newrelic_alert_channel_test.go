package newrelic

import (
	"testing"

	"github.com/r3labs/terraform/helper/acctest"
	"github.com/r3labs/terraform/helper/resource"
)

func TestAccNewRelicAlertChannel_import(t *testing.T) {
	resourceName := "newrelic_alert_channel.foo"
	rName := acctest.RandString(5)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNewRelicAlertChannelDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckNewRelicAlertChannelConfig(rName),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
