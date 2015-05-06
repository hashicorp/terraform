package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/route53"
)

func TestAccRoute53ZoneAssociation(t *testing.T) {
	var zone route53.HostedZone

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53ZoneAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53ZoneAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53ZoneAssociationExists("aws_route53_zone_association.main", &zone),
				),
			},
		},
	})
}

func testAccCheckRoute53ZoneAssociationDestroy(s *terraform.State) error {
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

func testAccCheckRoute53ZoneAssociationExists(n string, zone *route53.HostedZone) resource.TestCheckFunc {
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

		exists := false
		for i := range resp.VPCs {
			if rs.Primary.Meta["vpc_id"] == *resp.VPCs[i].VPCID {
				exists = true
			}
		}
		if !exists {
			return fmt.Errorf("Hosted zone association not found")
		}

		*zone = *resp.HostedZone
		return nil
	}
}

const testAccRoute53ZoneAssociationConfig = `
resource "aws_vpc" "mosakos" {
	cidr_block = "10.6.0.0/16"

	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_route53_zone" "main" {
	name = "mosakos.com"

	tags {
		foo  = "bar"
		Name = "tf-route53-tag-test"
	}
}

resource "aws_route53_zone_association" "main" {
	vpc_id  = "${aws_vpc.mosakos.id}"
	zone_id = "${aws_route53_zone.main.id}"
	region  = "us-west-2"
}
`
