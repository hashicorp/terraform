package aws

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

func TestAccAWSRoute53DelegationSet_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53DelegationSetConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53DelegationSetExists("aws_route53_delegation_set.test"),
				),
			},
		},
	})
}

func TestAccAWSRoute53DelegationSet_withZones(t *testing.T) {
	var zone route53.GetHostedZoneOutput

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRoute53ZoneDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRoute53DelegationSetWithZonesConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoute53DelegationSetExists("aws_route53_delegation_set.main"),
					testAccCheckRoute53ZoneExists("aws_route53_zone.primary", &zone),
					testAccCheckRoute53ZoneExists("aws_route53_zone.secondary", &zone),
					testAccCheckRoute53NameServersMatch("aws_route53_delegation_set.main", "aws_route53_zone.primary"),
					testAccCheckRoute53NameServersMatch("aws_route53_delegation_set.main", "aws_route53_zone.secondary"),
				),
			},
		},
	})
}

func testAccCheckRoute53DelegationSetDestroy(s *terraform.State, provider *schema.Provider) error {
	conn := provider.Meta().(*AWSClient).r53conn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_route53_delegation_set" {
			continue
		}

		_, err := conn.GetReusableDelegationSet(&route53.GetReusableDelegationSetInput{Id: aws.String(rs.Primary.ID)})
		if err == nil {
			return fmt.Errorf("Delegation set still exists")
		}
	}
	return nil
}

func testAccCheckRoute53DelegationSetExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).r53conn
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No delegation set ID is set")
		}

		out, err := conn.GetReusableDelegationSet(&route53.GetReusableDelegationSetInput{
			Id: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("Delegation set does not exist: %#v", rs.Primary.ID)
		}

		setID := cleanDelegationSetId(*out.DelegationSet.Id)
		if setID != rs.Primary.ID {
			return fmt.Errorf("Delegation set ID does not match:\nExpected: %#v\nReturned: %#v", rs.Primary.ID, setID)
		}

		return nil
	}
}

func testAccCheckRoute53NameServersMatch(delegationSetName, zoneName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).r53conn

		delegationSetLocal, ok := s.RootModule().Resources[delegationSetName]
		if !ok {
			return fmt.Errorf("Not found: %s", delegationSetName)
		}
		delegationSet, err := conn.GetReusableDelegationSet(&route53.GetReusableDelegationSetInput{
			Id: aws.String(delegationSetLocal.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("Delegation set does not exist: %#v", delegationSetLocal.Primary.ID)
		}

		hostedZoneLocal, ok := s.RootModule().Resources[zoneName]
		if !ok {
			return fmt.Errorf("Not found: %s", zoneName)
		}
		hostedZone, err := conn.GetHostedZone(&route53.GetHostedZoneInput{
			Id: aws.String(hostedZoneLocal.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("Delegation set does not exist: %#v", hostedZoneLocal.Primary.ID)
		}

		if !reflect.DeepEqual(delegationSet.DelegationSet.NameServers, hostedZone.DelegationSet.NameServers) {
			return fmt.Errorf("Name servers do not match:\nDelegation Set: %#v\nHosted Zone:%#v",
				delegationSet.DelegationSet.NameServers, hostedZone.DelegationSet.NameServers)
		}

		return nil
	}
}

const testAccRoute53DelegationSetConfig = `
resource "aws_route53_delegation_set" "test" {
	reference_name = "test"
}
`

const testAccRoute53DelegationSetWithZonesConfig = `
resource "aws_route53_delegation_set" "main" {
    reference_name = "main"
}

resource "aws_route53_zone" "primary" {
    name = "hashicorp.com"
    delegation_set_id = "${aws_route53_delegation_set.main.id}"
}

resource "aws_route53_zone" "secondary" {
    name = "terraform.io"
    delegation_set_id = "${aws_route53_delegation_set.main.id}"
}
`
