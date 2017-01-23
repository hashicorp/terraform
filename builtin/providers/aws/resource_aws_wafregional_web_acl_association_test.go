package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/aws/aws-sdk-go/service/wafregional"
)

func TestAccAWSWafRegionalWebAclAssociation_basic(t *testing.T) {
	var webAcl waf.WebACL

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckWafRegionalWebAclAssociationDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckWafRegionalWebAclAssociationConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWafRegionalWebAclAssociationExists("aws_wafregional_web_acl_association.foo", &webAcl),
				),
			},
		},
	})
}

func testAccCheckWafRegionalWebAclAssociationDestroy(s *terraform.State) error {
	return testAccCheckWafRegionalWebAclAssociationDestroyWithProvider(s, testAccProvider)
}

func testAccCheckWafRegionalWebAclAssociationDestroyWithProvider(s *terraform.State, provider *schema.Provider) error {
	conn := provider.Meta().(*AWSClient).wafregionalconn
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_wafregional_web_acl_association" {
			continue
		}

		web_acl_id, resource_arn := resourceAwsWafRegionalWebAclAssociationParseId(rs.Primary.ID)

		resp, err := conn.ListResourcesForWebACL(&wafregional.ListResourcesForWebACLInput{WebACLId: aws.String(web_acl_id)})
		if err != nil {
			found := false
			for _, list_resource_arn := range resp.ResourceArns {
				if resource_arn == *list_resource_arn {
					found = true
					break
				}
			}
			if found {
				return fmt.Errorf("WebACL: %v is still associated to resource: %v", web_acl_id, resource_arn)
			}
		}
	}
	return nil
}

func testAccCheckWafRegionalWebAclAssociationExists(n string, webAcl *waf.WebACL) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return testAccCheckWafRegionalWebAclAssociationExistsWithProvider(s, n, webAcl, testAccProvider)
	}
}

func testAccCheckWafRegionalWebAclAssociationExistsWithProvider(s *terraform.State, n string, webAcl *waf.WebACL, provider *schema.Provider) error {
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return fmt.Errorf("Not found: %s", n)
	}

	if rs.Primary.ID == "" {
		return fmt.Errorf("No WebACL association ID is set")
	}

	web_acl_id, resource_arn := resourceAwsWafRegionalWebAclAssociationParseId(rs.Primary.ID)

	conn := provider.Meta().(*AWSClient).wafregionalconn
	resp, err := conn.ListResourcesForWebACL(&wafregional.ListResourcesForWebACLInput{WebACLId: aws.String(web_acl_id)})
	if err != nil {
		return fmt.Errorf("List Web ACL err: %v", err)
	}

	found := false
	for _, list_resource_arn := range resp.ResourceArns {
		if resource_arn == *list_resource_arn {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Web ACL association not found")
	}

	return nil
}

const testAccCheckWafRegionalWebAclAssociationConfig = `
resource "aws_wafregional_ipset" "foo" {
  name = "foo"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_wafregional_rule" "foo" {
  depends_on = ["aws_wafregional_ipset.foo"]
  name = "foo"
  metric_name = "foo"
  predicates {
    data_id = "${aws_wafregional_ipset.foo.id}"
    negated = false
    type = "IPMatch"
  }
}

resource "aws_wafregional_web_acl" "foo" {
  name = "foo"
  metric_name = "foo"
  default_action {
    type = "ALLOW"
  }
	rules {
	 action {
			type = "COUNT"
	 }
	 priority = 100
	 rule_id = "${aws_wafregional_rule.foo.id}"
 }
}

resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	vpc_id = "${aws_vpc.foo.id}"
	cidr_block = "10.1.1.0/24"
}

resource "aws_subnet" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
	cidr_block = "10.1.2.0/24"
}

resource "aws_alb" "foo" {
    subnets = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}

resource "aws_wafregional_web_acl_association" "foo" {
    depends_on = ["aws_alb.foo", "aws_wafregional_web_acl.foo"]
    resource_arn = "${aws_alb.foo.arn}"
    web_acl_id = "${aws_wafregional_web_acl.foo.id}"
}
`
