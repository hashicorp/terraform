package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/awslabs/aws-sdk-go/aws"
	awsr53 "github.com/awslabs/aws-sdk-go/gen/route53"
)

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

		lopts := &awsr53.ListResourceRecordSetsRequest{
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

		lopts := &awsr53.ListResourceRecordSetsRequest{
			HostedZoneID:    aws.String(cleanZoneID(zone)),
			StartRecordName: aws.String(name),
			StartRecordType: aws.String(rType),
		}

		resp, err := conn.ListResourceRecordSets(lopts)
		if err != nil {
			return err
		}
		if len(resp.ResourceRecordSets) == 0 {
			return fmt.Errorf("Record does not exist")
		}
		rec := resp.ResourceRecordSets[0]
		if FQDN(*rec.Name) == FQDN(name) && *rec.Type == rType {
			return nil
		}
		return fmt.Errorf("Record does not exist: %#v", rec)
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
