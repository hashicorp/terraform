package spotinst

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

func TestAccSpotinstSubscription_Basic(t *testing.T) {
	var subscription spotinst.Subscription
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		//CheckDestroy: testAccCheckSpotinstSubscriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSpotinstSubscriptionConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstSubscriptionExists("spotinst_subscription.foo", &subscription), testAccCheckSpotinstSubscriptionAttributes(&subscription),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "protocol", "http"),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "endpoint", "http://endpoint.com"),
				),
			},
		},
	})
}

func TestAccSpotinstSubscription_Updated(t *testing.T) {
	var subscription spotinst.Subscription
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		//CheckDestroy: testAccCheckSpotinstSubscriptionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckSpotinstSubscriptionConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstSubscriptionExists("spotinst_subscription.foo", &subscription), testAccCheckSpotinstSubscriptionAttributes(&subscription),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "protocol", "http"),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "endpoint", "http://endpoint.com"),
				),
			},
			resource.TestStep{
				Config: testAccCheckSpotinstSubscriptionConfigNewValue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstSubscriptionExists("spotinst_subscription.foo", &subscription), testAccCheckSpotinstSubscriptionAttributesUpdated(&subscription),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "protocol", "https"),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "endpoint", "https://endpoint.com"),
				),
			},
		},
	})
}

func testAccCheckSpotinstSubscriptionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*spotinst.Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "spotinst_subscription" {
			continue
		}

		_, _, err := client.Subscription.Get(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Subscription still exists")
		}
	}

	return nil
}

func testAccCheckSpotinstSubscriptionAttributes(subscription *spotinst.Subscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *subscription.Protocol != "http" {
			return fmt.Errorf("Bad content: %v", subscription.Protocol)
		}

		if *subscription.Endpoint != "http://endpoint.com" {
			return fmt.Errorf("Bad content: %v", subscription.Endpoint)
		}

		return nil
	}
}

func testAccCheckSpotinstSubscriptionAttributesUpdated(subscription *spotinst.Subscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *subscription.Protocol != "https" {
			return fmt.Errorf("Bad content: %v", subscription.Protocol)
		}

		if *subscription.Endpoint != "https://endpoint.com" {
			return fmt.Errorf("Bad content: %v", subscription.Endpoint)
		}

		return nil
	}
}

func testAccCheckSpotinstSubscriptionExists(n string, subscription *spotinst.Subscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No resource ID is set")
		}

		client := testAccProvider.Meta().(*spotinst.Client)
		foundSubscriptions, _, err := client.Subscription.Get(rs.Primary.ID)

		if err != nil {
			return err
		}

		if *foundSubscriptions[0].ID != rs.Primary.Attributes["id"] {
			return fmt.Errorf("Subscription not found: %+v,\n %+v\n", foundSubscriptions[0], rs.Primary.Attributes)
		}

		*subscription = *foundSubscriptions[0]

		return nil
	}
}

const testAccCheckSpotinstSubscriptionConfigBasic = `
resource "spotinst_subscription" "foo" {
	resource_id = "sig-foo"
	event_type = "aws_ec2_instance_launch"
	protocol = "http"
	endpoint = "http://endpoint.com"
	format = {
		instance_id = "%instance-id%"
		tags = "foo,baz,baz"
	}
}`

const testAccCheckSpotinstSubscriptionConfigNewValue = `
resource "spotinst_subscription" "foo" {
	resource_id = "sig-foo"
	event_type = "aws_ec2_instance_launch"
	protocol = "https"
	endpoint = "https://endpoint.com"
	format = {
		instance_id = "%instance-id%"
		tags = "foo,baz,baz"
	}
}`
