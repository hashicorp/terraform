package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/route53"
)

func TestCleanPrefix(t *testing.T) {
	cases := []struct {
		Input, Prefix, Output string
	}{
		{"/hostedzone/foo", "/hostedzone/", "foo"},
		{"/change/foo", "/change/", "foo"},
		{"/bar", "/test", "/bar"},
	}

	for _, tc := range cases {
		actual := cleanPrefix(tc.Input, tc.Prefix)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

func TestCleanZoneID(t *testing.T) {
	cases := []struct {
		Input, Output string
	}{
		{"/hostedzone/foo", "foo"},
		{"/change/foo", "/change/foo"},
		{"/bar", "/bar"},
	}

	for _, tc := range cases {
		actual := cleanZoneID(tc.Input)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

func TestCleanChangeID(t *testing.T) {
	cases := []struct {
		Input, Output string
	}{
		{"/hostedzone/foo", "/hostedzone/foo"},
		{"/change/foo", "foo"},
		{"/bar", "/bar"},
	}

	for _, tc := range cases {
		actual := cleanChangeID(tc.Input)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

func TestAccRoute53Zone(t *testing.T) {
	var zone route53.HostedZone
	var td route53.ResourceTagSet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53ZoneConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53ZoneExists("aws_route53_zone.main", &zone),
					testAccLoadTagsR53(&zone, &td),
					testAccCheckTagsR53(&td.Tags, "foo", "bar"),
				),
			},
		},
	})
}

func testAccCheckRoute53ZoneDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).r53conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_zone" {
			continue
		}

		_, err := conn.GetHostedZone(&route53.GetHostedZoneInput{ID: aws.String(rs.Primary.ID)})
		if err == nil {
			return fmt.Errorf("Hosted zone still exists")
		}
	}
	return nil
}

func testAccCheckRoute53ZoneExists(n string, zone *route53.HostedZone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No hosted zone ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).r53conn
		resp, err := conn.GetHostedZone(&route53.GetHostedZoneInput{ID: aws.String(rs.Primary.ID)})
		if err != nil {
			return fmt.Errorf("Hosted zone err: %v", err)
		}
		*zone = *resp.HostedZone
		return nil
	}
}

func testAccLoadTagsR53(zone *route53.HostedZone, td *route53.ResourceTagSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).r53conn

		zone := cleanZoneID(*zone.ID)
		req := &route53.ListTagsForResourceInput{
			ResourceID:   aws.String(zone),
			ResourceType: aws.String("hostedzone"),
		}

		resp, err := conn.ListTagsForResource(req)
		if err != nil {
			return err
		}

		if resp.ResourceTagSet != nil {
			*td = *resp.ResourceTagSet
		}

		return nil
	}
}

const testAccRoute53ZoneConfig = `
resource "aws_route53_zone" "main" {
	name = "hashicorp.com"

	tags {
		foo = "bar"
		Name = "tf-route53-tag-test"
	}
}
`
