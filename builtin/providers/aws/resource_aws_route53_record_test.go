package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/route53"
)

func TestCleanRecordName(t *testing.T) {
	cases := []struct {
		Input, Output string
	}{
		{"www.nonexample.com", "www.nonexample.com"},
		{"\\052.nonexample.com", "*.nonexample.com"},
		{"nonexample.com", "nonexample.com"},
	}

	for _, tc := range cases {
		actual := cleanRecordName(tc.Input)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

func TestExpandRecordName(t *testing.T) {
	cases := []struct {
		Input, Output string
	}{
		{"www", "www.nonexample.com"},
		{"dev.www", "dev.www.nonexample.com"},
		{"*", "*.nonexample.com"},
		{"nonexample.com", "nonexample.com"},
		{"test.nonexample.com", "test.nonexample.com"},
		{"test.nonexample.com.", "test.nonexample.com"},
	}

	zone_name := "nonexample.com"
	for _, tc := range cases {
		actual := expandRecordName(tc.Input, zone_name)
		if actual != tc.Output {
			t.Fatalf("input: %s\noutput: %s", tc.Input, actual)
		}
	}
}

func TestAccRoute53Record(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53RecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53RecordConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53RecordExists("aws_route53_record.default"),
				),
			},
		},
	})
}

func TestAccRoute53Record_txtSupport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53RecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53RecordConfigTXT,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53RecordExists("aws_route53_record.default"),
				),
			},
		},
	})
}

func TestAccRoute53Record_generatesSuffix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53RecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53RecordConfigSuffix,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53RecordExists("aws_route53_record.default"),
				),
			},
		},
	})
}

func TestAccRoute53Record_wildcard(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53RecordDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53WildCardRecordConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53RecordExists("aws_route53_record.wildcard"),
				),
			},

			// Cause a change, which will trigger a refresh
			resource.TestStep{
				Config: testAccRoute53WildCardRecordConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53RecordExists("aws_route53_record.wildcard"),
				),
			},
		},
	})
}

func testAccCheckRoute53RecordDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).r53conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_record" {
			continue
		}

		parts := strings.Split(rs.Primary.ID, "_")
		zone := parts[0]
		name := parts[1]
		rType := parts[2]

		lopts := &route53.ListResourceRecordSetsInput{
			HostedZoneID:    aws.String(cleanZoneID(zone)),
			StartRecordName: aws.String(name),
			StartRecordType: aws.String(rType),
		}

		resp, err := conn.ListResourceRecordSets(lopts)
		if err != nil {
			return err
		}
		if len(resp.ResourceRecordSets) == 0 {
			return nil
		}
		rec := resp.ResourceRecordSets[0]
		if FQDN(*rec.Name) == FQDN(name) && *rec.Type == rType {
			return fmt.Errorf("Record still exists: %#v", rec)
		}
	}
	return nil
}

func testAccCheckRoute53RecordExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).r53conn
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No hosted zone ID is set")
		}

		parts := strings.Split(rs.Primary.ID, "_")
		zone := parts[0]
		name := parts[1]
		rType := parts[2]

		en := expandRecordName(name, "notexample.com")

		lopts := &route53.ListResourceRecordSetsInput{
			HostedZoneID:    aws.String(cleanZoneID(zone)),
			StartRecordName: aws.String(en),
			StartRecordType: aws.String(rType),
		}

		resp, err := conn.ListResourceRecordSets(lopts)
		if err != nil {
			return err
		}
		if len(resp.ResourceRecordSets) == 0 {
			return fmt.Errorf("Record does not exist")
		}
		// rec := resp.ResourceRecordSets[0]
		for _, rec := range resp.ResourceRecordSets {
			recName := cleanRecordName(*rec.Name)
			if FQDN(recName) == FQDN(en) && *rec.Type == rType {
				return nil
			}
		}
		return fmt.Errorf("Record does not exist: %#v", rs.Primary.ID)
	}
}

const testAccRoute53RecordConfig = `
resource "aws_route53_zone" "main" {
	name = "notexample.com"
}

resource "aws_route53_record" "default" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "www.notexample.com"
	type = "A"
	ttl = "30"
	records = ["127.0.0.1", "127.0.0.27"]
}
`

const testAccRoute53RecordConfigSuffix = `
resource "aws_route53_zone" "main" {
	name = "notexample.com"
}

resource "aws_route53_record" "default" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "subdomain"
	type = "A"
	ttl = "30"
	records = ["127.0.0.1", "127.0.0.27"]
}
`

const testAccRoute53WildCardRecordConfig = `
resource "aws_route53_zone" "main" {
    name = "notexample.com"
}

resource "aws_route53_record" "default" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "subdomain"
	type = "A"
	ttl = "30"
	records = ["127.0.0.1", "127.0.0.27"]
}

resource "aws_route53_record" "wildcard" {
    zone_id = "${aws_route53_zone.main.zone_id}"
    name = "*.notexample.com"
    type = "A"
    ttl = "30"
    records = ["127.0.0.1"]
}
`

const testAccRoute53WildCardRecordConfigUpdate = `
resource "aws_route53_zone" "main" {
    name = "notexample.com"
}

resource "aws_route53_record" "default" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "subdomain"
	type = "A"
	ttl = "30"
	records = ["127.0.0.1", "127.0.0.27"]
}

resource "aws_route53_record" "wildcard" {
    zone_id = "${aws_route53_zone.main.zone_id}"
    name = "*.notexample.com"
    type = "A"
    ttl = "60"
    records = ["127.0.0.1"]
}
`
const testAccRoute53RecordConfigTXT = `
resource "aws_route53_zone" "main" {
	name = "notexample.com"
}

resource "aws_route53_record" "default" {
	zone_id = "/hostedzone/${aws_route53_zone.main.zone_id}"
	name = "subdomain"
	type = "TXT"
	ttl = "30"
	records = ["lalalala"]
}
`
