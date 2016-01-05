package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPubsubTopicCreate(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPubsubTopicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccPubsubTopic,
				Check: resource.ComposeTestCheckFunc(
					testAccPubsubTopicExists(
						"google_pubsub_topic.foobar"),
				),
			},
		},
	})
}

func testAccCheckPubsubTopicDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "google_pubsub_topic" {
			continue
		}

		config := testAccProvider.Meta().(*Config)
		_, err := config.clientPubsub.Projects.Topics.Get(rs.Primary.ID).Do()
		if err != nil {
			fmt.Errorf("Topic still present")
		}
	}

	return nil
}

func testAccPubsubTopicExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		config := testAccProvider.Meta().(*Config)
		_, err := config.clientPubsub.Projects.Topics.Get(rs.Primary.ID).Do()
		if err != nil {
			fmt.Errorf("Topic still present")
		}

		return nil
	}
}

var testAccPubsubTopic = fmt.Sprintf(`
resource "google_pubsub_topic" "foobar" {
	name = "pstopic-test-%s"
}`, acctest.RandString(10))
