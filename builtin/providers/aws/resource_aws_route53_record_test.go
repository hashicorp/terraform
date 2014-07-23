package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/route53"
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

func testAccCheckRoute53RecordDestroy(s *terraform.State) error {
	conn := testAccProvider.route53
	for _, rs := range s.Resources {
		if rs.Type != "aws_route53_record" {
			continue
		}

		parts := strings.Split(rs.ID, "_")
		zone := parts[0]
		name := parts[1]
		rType := parts[2]

		lopts := &route53.ListOpts{Name: name, Type: rType}
		resp, err := conn.ListResourceRecordSets(zone, lopts)
		if err != nil {
			return err
		}
		if len(resp.Records) == 0 {
			return nil
		}
		rec := resp.Records[0]
		if route53.FQDN(rec.Name) == route53.FQDN(name) && rec.Type == rType {
			return fmt.Errorf("Record still exists: %#v", rec)
		}
	}
	return nil
}

func testAccCheckRoute53RecordExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.route53
		rs, ok := s.Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.ID == "" {
			return fmt.Errorf("No hosted zone ID is set")
		}

		parts := strings.Split(rs.ID, "_")
		zone := parts[0]
		name := parts[1]
		rType := parts[2]

		lopts := &route53.ListOpts{Name: name, Type: rType}
		resp, err := conn.ListResourceRecordSets(zone, lopts)
		if err != nil {
			return err
		}
		if len(resp.Records) == 0 {
			return fmt.Errorf("Record does not exist")
		}
		rec := resp.Records[0]
		if route53.FQDN(rec.Name) == route53.FQDN(name) && rec.Type == rType {
			return nil
		}
		return fmt.Errorf("Record does not exist: %#v", rec)
	}
}

const testAccRoute53RecordConfig = `
resource "aws_route53_zone" "main" {
	name = "hashicorp.com"
}

resource "aws_route53_record" "default" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "www.hashicorp.com"
	type = "A"
	ttl = "30"
	records = ["127.0.0.1"]
}
`
