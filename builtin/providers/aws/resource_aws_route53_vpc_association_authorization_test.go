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

func TestAccAWSRoute53VpcAssociationAuthorization_basic(t *testing.T) {
	var zone route53.HostedZone

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53VPCAssociationAuthorizationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53VPCAssociationAuthorizationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53VPCAssociationAuthorizationExists("aws_route53_vpc_association_authorization.peer", &zone),
				),
			},
		},
	})
}

func TestAccAWSRoute53VPCAssociationAuthorization_region(t *testing.T) {
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
		CheckDestroy:      testAccCheckRoute53VPCAssociationAuthorizationDestroyWithProviders(&providers),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53ZoneAssociationRegionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53VPCAssociationAuthorizationExistsWithProviders("aws_route53_vpc_association_authorization.peer", &zone, &providers),
				),
			},
		},
	})
}

func testAccCheckRoute53VPCAssociationAuthorizationDestroy(s *terraform.State) error {
	return testAccCheckRoute53VPCAssociationAuthorizationDestroyWithProvider(s, testAccProvider)
}

func testAccCheckRoute53VPCAssociationAuthorizationDestroyWithProviders(providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, provider := range *providers {
			if provider.Meta() == nil {
				continue
			}
			if err := testAccCheckRoute53VPCAssociationAuthorizationDestroyWithProvider(s, provider); err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccCheckRoute53VPCAssociationAuthorizationDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	conn := provider.Meta().(*AWSClient).r53conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_vpc_association_authorization" {
			continue
		}

		zone_id, vpc_id := resourceAwsRoute53ZoneAssociationParseId(rs.Primary.ID)

		req := route53.ListVPCAssociationAuthorizationsInput{HostedZoneId: aws.String(zone_id)}
		res, err := conn.ListVPCAssociationAuthorizations(&req)
		if err != nil {
			return err
		}

		exists := false
		for _, vpc := range res.VPCs {
			if vpc_id == *vpc.VPCId {
				exists = true
			}
		}

		if exists {
			return fmt.Errorf("VPC association authorization for zone %v with %v still exists", zone_id, vpc_id)
		}
	}
	return nil
}

func testAccCheckRoute53VPCAssociationAuthorizationExists(n string, zone *route53.HostedZone) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return testAccCheckRoute53VPCAssociationAuthorizationExistsWithProvider(s, n, zone, testAccProvider)
	}
}

func testAccCheckRoute53VPCAssociationAuthorizationExistsWithProviders(n string, zone *route53.HostedZone, providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, provider := range *providers {
			if provider.Meta() == nil {
				continue
			}
			if err := testAccCheckRoute53VPCAssociationAuthorizationExistsWithProvider(s, n, zone, provider); err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccCheckRoute53VPCAssociationAuthorizationExistsWithProvider(s *terraform.State, n string, zone *route53.HostedZone, provider *schema.Provider) error {
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return fmt.Errorf("Not found: %s", n)
	}

	if rs.Primary.ID == "" {
		return fmt.Errorf("No VPC association authorization ID is set")
	}

	zone_id, vpc_id := resourceAwsRoute53ZoneAssociationParseId(rs.Primary.ID)
	conn := provider.Meta().(*AWSClient).r53conn

	req := route53.ListVPCAssociationAuthorizationsInput{HostedZoneId: aws.String(zone_id)}
	res, err := conn.ListVPCAssociationAuthorizations(&req)
	if err != nil {
		return err
	}

	exists := false
	for _, vpc := range res.VPCs {
		if vpc_id == *vpc.VPCId {
			exists = true
		}
	}

	if !exists {
		return fmt.Errorf("VPC association authorization not found")
	}

	return nil
}

const testAccRoute53VPCAssociationAuthorizationConfig = `
provider "aws" {
    region = "us-west-2"
    // Requester's credentials.
}

provider "aws" {
    alias = "peer"
    region = "us-west-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.6.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_vpc" "peer" {
    provider = "aws.peer"
	cidr_block = "10.7.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_route53_zone" "foo" {
	name = "foo.com"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route53_vpc_association_authorization" "peer" {
    zone_id = "${aws_route53_zone.foo.id}"
    vpc_id  = "${aws_vpc.peer.id}"
}

resource "aws_route53_zone_association" "foobar" {
	zone_id = "${aws_route53_zone.foo.id}"
	vpc_id  = "${aws_vpc.peer.id}"
}
`

const testAccRoute53VPCAssociationAuthorizationRegionConfig = `
provider "aws" {
    region = "us-west-2"
    // Requester's credentials.
}

provider "aws" {
    alias = "peer"
    region = "us-east-2"
}

resource "aws_vpc" "foo" {
	cidr_block = "10.6.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_vpc" "peer" {
    provider = "aws.peer"
	cidr_block = "10.7.0.0/16"
	enable_dns_hostnames = true
	enable_dns_support = true
}

resource "aws_route53_zone" "foo" {
	name = "foo.com"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_route53_vpc_association_authorization" "peer" {
    zone_id = "${aws_route53_zone.foo.id}"
    vpc_id  = "${aws_vpc.peer.id}"
	region  = "us-east-2"
}

resource "aws_route53_zone_association" "foobar" {
	zone_id = "${aws_route53_zone.foo.id}"
	vpc_id  = "${aws_vpc.peer.id}"
	region  = "us-east-2"
}
`
