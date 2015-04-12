package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/hashicorp/aws-sdk-go/aws"
	route53 "github.com/hashicorp/aws-sdk-go/gen/route53"
)

func TestAccRoute53AliasTarget_toELB(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53AliasTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53AliasTargetELBConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53AliasTargetExists("aws_route53_alias_target.default"),
				),
			},
		},
	})
}

func TestAccRoute53AliasTarget_toRecord(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53AliasTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53AliasTargetRecordConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53AliasTargetExists("aws_route53_alias_target.default"),
				),
			},
		},
	})
}

func TestAccRoute53AliasTarget_wildcardToRecord(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53AliasTargetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53WildCardAliasTargetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53AliasTargetExists("aws_route53_alias_target.wildcard_to_record"),
				),
			},

			// Cause a change, which will trigger a refresh
			resource.TestStep{
				Config: testAccRoute53WildCardAliasTargetConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53AliasTargetExists("aws_route53_alias_target.wildcard_to_record"),
				),
			},
		},
	})
}

func testAccCheckRoute53AliasTargetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).r53conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_alias_target" {
			continue
		}

		parts := strings.Split(rs.Primary.ID, "_")
		zone := parts[0]
		name := parts[1]
		rType := parts[2]

		lopts := &route53.ListResourceRecordSetsRequest{
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

func testAccCheckRoute53AliasTargetExists(n string) resource.TestCheckFunc {
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

		lopts := &route53.ListResourceRecordSetsRequest{
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

		for _, rec := range resp.ResourceRecordSets {
			recName := cleanRecordName(*rec.Name)
			if FQDN(recName) == FQDN(en) && *rec.Type == rType {
				return nil
			}
		}
		return fmt.Errorf("Record does not exist: %#v", rs.Primary.ID)
	}
}

const testAccRoute53AliasTargetELBConfig = `
resource "aws_vpc" "main" {
    cidr_block = "10.0.0.0/16"
}

resource "aws_internet_gateway" "main" {
    vpc_id = "${aws_vpc.main.id}"
}

resource "aws_subnet" "main" {
    vpc_id = "${aws_vpc.main.id}"
    cidr_block = "10.0.1.0/24"
}

resource "aws_elb" "default" {
	name = "terraform-example-elb"
	subnets = ["${aws_subnet.main.id}"]
	listener {
		instance_port = 80
		instance_protocol = "TCP"
		lb_port = 80
		lb_protocol = "TCP"
	}
}

resource "aws_route53_zone" "main" {
	name = "notexample.com"
}

resource "aws_route53_alias_target" "default" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "www"
	type = "A"
	target = "${aws_elb.default.dns_name}"
	target_zone_id = "${aws_elb.default.hosted_zone_id}"
}
`

const testAccRoute53AliasTargetRecordConfig = `
resource "aws_route53_zone" "main" {
    name = "notexample.com"
}

resource "aws_route53_record" "subdomain_one" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "subdomain-one"
	type = "A"
	ttl = 60
	records = ["127.0.0.1"]
}

resource "aws_route53_alias_target" "default" {
    zone_id = "${aws_route53_zone.main.zone_id}"
    name = "sub.notexample.com"
    type = "A"
		target = "${aws_route53_record.subdomain_one.name}"
		target_zone_id = "${aws_route53_record.subdomain_one.zone_id}"
}
`

const testAccRoute53WildCardAliasTargetConfig = `
resource "aws_route53_zone" "main" {
    name = "notexample.com"
}

resource "aws_route53_record" "subdomain_one" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "subdomain-one"
	type = "A"
	ttl = 60
	records = ["127.0.0.1"]
}

resource "aws_route53_alias_target" "wildcard_to_record" {
    zone_id = "${aws_route53_zone.main.zone_id}"
    name = "*.notexample.com"
    type = "A"
		target = "${aws_route53_record.subdomain_one.name}"
		target_zone_id = "${aws_route53_record.subdomain_one.zone_id}"
}
`

const testAccRoute53WildCardAliasTargetConfigUpdate = `
resource "aws_route53_zone" "main" {
    name = "notexample.com"
}

resource "aws_route53_record" "subdomain_two" {
	zone_id = "${aws_route53_zone.main.zone_id}"
	name = "subdomain-two"
	type = "A"
	ttl = 60
	records = ["127.0.0.1"]
}

resource "aws_route53_alias_target" "wildcard_to_record" {
    zone_id = "${aws_route53_zone.main.zone_id}"
    name = "*.notexample.com"
    type = "A"
		target = "${aws_route53_record.subdomain_two.name}"
		target_zone_id = "${aws_route53_record.subdomain_two.zone_id}"
}
`
