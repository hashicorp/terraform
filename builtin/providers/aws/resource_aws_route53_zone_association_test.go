package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

func TestAccAWSRoute53ZoneAssociation_basic(t *testing.T) {
	var zone route53.HostedZone

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53ZoneAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53ZoneAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53ZoneAssociationExists("aws_route53_zone_association.foobar", &zone),
				),
			},
		},
	})
}

func TestAccAWSRoute53ZoneAssociation_region(t *testing.T) {
	var zone route53.HostedZone

	// record the initialized providers so that we can use them to
	// check for the instances in each region
	var providers []*schema.Provider
	providerFactories := map[string]terraform.ResourceProviderFactory{
		"aws": func() (terraform.ResourceProvider, error) {
			p := Provider()
			providers = append(providers, p.(*schema.Provider))
			return p, nil
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckRoute53ZoneAssociationDestroyWithProviders(&providers),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53ZoneAssociationRegionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53ZoneAssociationExistsWithProviders("aws_route53_zone_association.foobar", &zone, &providers),
				),
			},
		},
	})
}

func testAccCheckRoute53ZoneAssociationDestroy(s *terraform.State) error {
	return testAccCheckRoute53ZoneAssociationDestroyWithProvider(s, testAccProvider)
}

func testAccCheckRoute53ZoneAssociationDestroyWithProviders(providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, provider := range *providers {
			if provider.Meta() == nil {
				continue
			}
			if err := testAccCheckRoute53ZoneAssociationDestroyWithProvider(s, provider); err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccCheckRoute53ZoneAssociationDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	conn := provider.Meta().(*AWSClient).r53conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_zone_association" {
			continue
		}

		zone_id, vpc_id := resourceAwsRoute53ZoneAssociationParseId(rs.Primary.ID)

		resp, err := conn.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone_id)})
		if err != nil {
			exists := false
			for _, vpc := range resp.VPCs {
				if vpc_id == *vpc.VPCId {
					exists = true
				}
			}
			if exists {
				return fmt.Errorf("VPC: %v is still associated to HostedZone: %v", vpc_id, zone_id)
			}
		}
	}
	return nil
}

func testAccCheckRoute53ZoneAssociationExists(n string, zone *route53.HostedZone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return testAccCheckRoute53ZoneAssociationExistsWithProvider(s, n, zone, testAccProvider)
	}
}

func testAccCheckRoute53ZoneAssociationExistsWithProviders(n string, zone *route53.HostedZone, providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, provider := range *providers {
			if provider.Meta() == nil {
				continue
			}
			if err := testAccCheckRoute53ZoneAssociationExistsWithProvider(s, n, zone, provider); err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccCheckRoute53ZoneAssociationExistsWithProvider(s *terraform.State, n string, zone *route53.HostedZone, provider *schema.Provider) error {
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return fmt.Errorf("Not found: %s", n)
	}

	if rs.Primary.ID == "" {
		return fmt.Errorf("No zone association ID is set")
	}

	zone_id, vpc_id := resourceAwsRoute53ZoneAssociationParseId(rs.Primary.ID)

	conn := provider.Meta().(*AWSClient).r53conn
	resp, err := conn.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone_id)})
	if err != nil {
		return fmt.Errorf("Hosted zone err: %v", err)
	}

	exists := false
	for _, vpc := range resp.VPCs {
		if vpc_id == *vpc.VPCId {
			exists = true
		}
	}
	if !exists {
		return fmt.Errorf("Hosted zone association not found")
	}

	*zone = *resp.HostedZone
	return nil
}

const testAccRoute53ZoneAssociationConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.6.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_vpc" "bar" {
	cidr_block = "10.7.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_route53_zone" "foo" {
	name = "foo.com"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route53_zone_association" "foobar" {
	zone_id = "${aws_route53_zone.foo.id}"
	vpc_id  = "${aws_vpc.bar.id}"
}
`

const testAccRoute53ZoneAssociationRegionConfig = `
provider "aws" {
	alias = "west"
	region = "us-west-2"
}

provider "aws" {
	alias = "east"
	region = "us-east-1"
}

resource "aws_vpc" "foo" {
	provider = "aws.west"
	cidr_block = "10.6.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_vpc" "bar" {
	provider = "aws.east"
	cidr_block = "10.7.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_route53_zone" "foo" {
	provider = "aws.west"
	name = "foo.com"
	vpc_id = "${aws_vpc.foo.id}"
	vpc_region = "us-west-2"
}

resource "aws_route53_zone_association" "foobar" {
	provider = "aws.west"
	zone_id = "${aws_route53_zone.foo.id}"
	vpc_id  = "${aws_vpc.bar.id}"
	vpc_region = "us-east-1"
}
`
