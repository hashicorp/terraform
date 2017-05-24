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
			{
				Config: testAccCheckSpotinstSubscriptionConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstSubscriptionExists("spotinst_subscription.foo", &subscription),
					testAccCheckSpotinstSubscriptionAttributes(&subscription),
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
			{
				Config: testAccCheckSpotinstSubscriptionConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstSubscriptionExists("spotinst_subscription.foo", &subscription),
					testAccCheckSpotinstSubscriptionAttributes(&subscription),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "protocol", "http"),
					resource.TestCheckResourceAttr("spotinst_subscription.foo", "endpoint", "http://endpoint.com"),
				),
			},
			{
				Config: testAccCheckSpotinstSubscriptionConfigNewValue,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSpotinstSubscriptionExists("spotinst_subscription.foo", &subscription),
					testAccCheckSpotinstSubscriptionAttributesUpdated(&subscription),
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
		input := &spotinst.ReadSubscriptionInput{ID: spotinst.String(rs.Primary.ID)}
		if _, err := client.SubscriptionService.Read(input); err == nil {
			return fmt.Errorf("Subscription still exists")
		}
	}
	return nil
}

func testAccCheckSpotinstSubscriptionAttributes(subscription *spotinst.Subscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if p := spotinst.StringValue(subscription.Protocol); p != "http" {
			return fmt.Errorf("Bad content: %s", p)
		}
		if e := spotinst.StringValue(subscription.Endpoint); e != "http://endpoint.com" {
			return fmt.Errorf("Bad content: %s", e)
		}
		return nil
	}
}

func testAccCheckSpotinstSubscriptionAttributesUpdated(subscription *spotinst.Subscription) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if p := spotinst.StringValue(subscription.Protocol); p != "https" {
			return fmt.Errorf("Bad content: %s", p)
		}
		if e := spotinst.StringValue(subscription.Endpoint); e != "https://endpoint.com" {
			return fmt.Errorf("Bad content: %s", e)
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
		input := &spotinst.ReadSubscriptionInput{ID: spotinst.String(rs.Primary.ID)}
		resp, err := client.SubscriptionService.Read(input)
		if err != nil {
			return err
		}
		if spotinst.StringValue(resp.Subscription.ID) != rs.Primary.Attributes["id"] {
			return fmt.Errorf("Subscription not found: %+v,\n %+v\n", resp.Subscription, rs.Primary.Attributes)
		}
		*subscription = *resp.Subscription
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
