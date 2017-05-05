package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPubsubSubscriptionCreate(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPubsubSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPubsubSubscription,
				Check: resource.ComposeTestCheckFunc(
					testAccPubsubSubscriptionExists(
						"google_pubsub_subscription.foobar_sub"),
					resource.TestCheckResourceAttrSet("google_pubsub_subscription.foobar_sub", "path"),
				),
			},
		},
	})
}

func testAccCheckPubsubSubscriptionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_pubsub_subscription" {
			continue
		}

		config := testAccProvider.Meta().(*Config)
		sub, _ := config.clientPubsub.Projects.Subscriptions.Get(rs.Primary.ID).Do()
		if sub != nil {
			return fmt.Errorf("Subscription still present")
		}
	}

	return nil
}

func testAccPubsubSubscriptionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		config := testAccProvider.Meta().(*Config)
		_, err := config.clientPubsub.Projects.Subscriptions.Get(rs.Primary.ID).Do()
		if err != nil {
			return fmt.Errorf("Subscription does not exist")
		}

		return nil
	}
}

var testAccPubsubSubscription = fmt.Sprintf(`
resource "google_pubsub_topic" "foobar_sub" {
	name = "pssub-test-%s"
}

resource "google_pubsub_subscription" "foobar_sub" {
	name                 = "pssub-test-%s"
	topic                = "${google_pubsub_topic.foobar_sub.name}"
	ack_deadline_seconds = 20
}`, acctest.RandString(10), acctest.RandString(10))
