package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
	// "github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	// "github.com/hashicorp/terraform/helper/schema"
)

const testAccAWSNetworkAclConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.2.0.0/16"
}

resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
}
`

// NetworkAclId   string                  `xml:"networkAclId"`
// VpcId          string                  `xml:"vpcId"`
// Default        string                  `xml:"default"`
// EntrySet       []NetworkAclEntry       `xml:"entrySet>item"`
// AssociationSet []NetworkAclAssociation `xml:"AssociationSet>item"`
// Tags           []Tag                   `xml:"tagSet>item"`

// type NetworkAclEntry struct {
// 	RuleNumber int       `xml:"ruleNumber"`
// 	Protocol   string    `xml:"protocol"`
// 	RuleAction string    `xml:"ruleAction"`
// 	Egress     bool      `xml:"egress"`
// 	CidrBlock  string    `xml:"cidrBlock"`
// 	IcmpCode   IcmpCode  `xml:"icmpTypeCode"`
// 	PortRange  PortRange `xml:"portRange"`
// }

func TestAccAWSNetworkAclsSneha(t *testing.T) {
	fmt.Printf("%s\n", "i am inside")
	var networkAcl ec2.NetworkAcl

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSNetworkAclConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSNetworkAclExists("aws_network_acl.bar", &networkAcl),
				),
			},
		},
	})
}

func testAccCheckAWSNetworkAclDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_network" {
			continue
		}

		// Retrieve the network acl
		resp, err := conn.NetworkAcls([]string{rs.Primary.ID}, ec2.NewFilter())
		if err == nil {
			if len(resp.NetworkAcls) > 0 && resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
				return fmt.Errorf("Network Acl (%s) still exists.", rs.Primary.ID)
			}

			return nil
		}

		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return err
		}
		// Confirm error code is what we want
		if ec2err.Code != "InvalidNetworkAclID.NotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSNetworkAclExists(n string, networkAcl *ec2.NetworkAcl) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Security Group is set")
		}
		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		resp, err := conn.NetworkAcls([]string{rs.Primary.ID}, nil)
		if err != nil {
			return err
		}

		if len(resp.NetworkAcls) > 0 && resp.NetworkAcls[0].NetworkAclId == rs.Primary.ID {
			*networkAcl = resp.NetworkAcls[0]
			return nil
		}

		return fmt.Errorf("Network Acls not found")
	}
}
