package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/terraform/helper/acctest"
)

func TestAccAWSWafRegionalWebAcl_basic(t *testing.T) {
	var v waf.WebACL
	wafAclName := fmt.Sprintf("wafacl%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalWebAclDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafRegionalWebAclConfig(wafAclName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalWebAclExists("aws_wafregional_web_acl.waf_acl", &v),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.4234791575.type", "ALLOW"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "name", wafAclName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "rules.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "metric_name", wafAclName),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalWebAcl_changeNameForceNew(t *testing.T) {
	var before, after waf.WebACL
	wafAclName := fmt.Sprintf("wafacl%s", acctest.RandString(5))
	wafAclNewName := fmt.Sprintf("wafacl%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalWebAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalWebAclConfig(wafAclName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalWebAclExists("aws_wafregional_web_acl.waf_acl", &before),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.4234791575.type", "ALLOW"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "name", wafAclName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "rules.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "metric_name", wafAclName),
				),
			},
			{
				Config: testAccAWSWafRegionalWebAclConfigChangeName(wafAclNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalWebAclExists("aws_wafregional_web_acl.waf_acl", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.4234791575.type", "ALLOW"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "name", wafAclNewName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "rules.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "metric_name", wafAclNewName),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalWebAcl_changeDefaultAction(t *testing.T) {
	var before, after waf.WebACL
	wafAclName := fmt.Sprintf("wafacl%s", acctest.RandString(5))
	wafAclNewName := fmt.Sprintf("wafacl%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalWebAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalWebAclConfig(wafAclName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalWebAclExists("aws_wafregional_web_acl.waf_acl", &before),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.4234791575.type", "ALLOW"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "name", wafAclName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "rules.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "metric_name", wafAclName),
				),
			},
			{
				Config: testAccAWSWafRegionalWebAclConfigDefaultAction(wafAclNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalWebAclExists("aws_wafregional_web_acl.waf_acl", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "default_action.2267395054.type", "BLOCK"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "name", wafAclNewName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "rules.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_web_acl.waf_acl", "metric_name", wafAclNewName),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalWebAcl_disappears(t *testing.T) {
	var v waf.WebACL
	wafAclName := fmt.Sprintf("wafacl%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalWebAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalWebAclConfig(wafAclName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalWebAclExists("aws_wafregional_web_acl.waf_acl", &v),
					testAccCheckAWSWafRegionalWebAclDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSWafRegionalWebAclDisappears(v *waf.WebACL) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn

		// ChangeToken
		var ct *waf.GetChangeTokenInput

		resp, err := conn.GetChangeToken(ct)
		if err != nil {
			return fmt.Errorf("Error getting change token: %s", err)
		}

		req := &waf.UpdateWebACLInput{
			ChangeToken: resp.ChangeToken,
			WebACLId:    v.WebACLId,
		}

		for _, ActivatedRule := range v.Rules {
			WebACLUpdate := &waf.WebACLUpdate{
				Action: aws.String("DELETE"),
				ActivatedRule: &waf.ActivatedRule{
					Priority: ActivatedRule.Priority,
					RuleId:   ActivatedRule.RuleId,
					Action:   ActivatedRule.Action,
				},
			}
			req.Updates = append(req.Updates, WebACLUpdate)
		}

		_, err = conn.UpdateWebACL(req)
		if err != nil {
			return fmt.Errorf("Error Updating WAF ACL: %s", err)
		}

		resp, err = conn.GetChangeToken(ct)
		if err != nil {
			return fmt.Errorf("Error getting change token for waf ACL: %s", err)
		}

		opts := &waf.DeleteWebACLInput{
			ChangeToken: resp.ChangeToken,
			WebACLId:    v.WebACLId,
		}
		if _, err := conn.DeleteWebACL(opts); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckAWSWafRegionalWebAclDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_wafregional_web_acl" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetWebACL(
			&waf.GetWebACLInput{
				WebACLId: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if *resp.WebACL.WebACLId == rs.Primary.ID {
				return fmt.Errorf("WebACL %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the WebACL is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "WAFNonexistentItemException" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSWafRegionalWebAclExists(n string, v *waf.WebACL) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WebACL ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetWebACL(&waf.GetWebACLInput{
			WebACLId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.WebACL.WebACLId == rs.Primary.ID {
			*v = *resp.WebACL
			return nil
		}

		return fmt.Errorf("WebACL (%s) not found", rs.Primary.ID)
	}
}

func testAccAWSWafRegionalWebAclConfig(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_wafregional_rule" "wafrule" {
  depends_on = ["aws_wafregional_ipset.ipset"]
  name = "%s"
  metric_name = "%s"
  predicates {
    data_id = "${aws_wafregional_ipset.ipset.id}"
    negated = false
    type = "IPMatch"
  }
}
resource "aws_wafregional_web_acl" "waf_acl" {
  depends_on = ["aws_wafregional_ipset.ipset", "aws_wafregional_rule.wafrule"]
  name = "%s"
  metric_name = "%s"
  default_action {
    type = "ALLOW"
  }
  rules {
    action {
       type = "BLOCK"
    }
    priority = 1 
    rule_id = "${aws_wafregional_rule.wafrule.id}"
  }
}`, name, name, name, name, name)
}

func testAccAWSWafRegionalWebAclConfigChangeName(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_wafregional_rule" "wafrule" {
  depends_on = ["aws_wafregional_ipset.ipset"]
  name = "%s"
  metric_name = "%s"
  predicates {
    data_id = "${aws_wafregional_ipset.ipset.id}"
    negated = false
    type = "IPMatch"
  }
}
resource "aws_wafregional_web_acl" "waf_acl" {
  depends_on = ["aws_wafregional_ipset.ipset", "aws_wafregional_rule.wafrule"]
  name = "%s"
  metric_name = "%s"
  default_action {
    type = "ALLOW"
  }
  rules {
    action {
       type = "BLOCK"
    }
    priority = 1 
    rule_id = "${aws_wafregional_rule.wafrule.id}"
  }
}`, name, name, name, name, name)
}

func testAccAWSWafRegionalWebAclConfigDefaultAction(name string) string {
	return fmt.Sprintf(`resource "aws_wafregional_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_wafregional_rule" "wafrule" {
  depends_on = ["aws_wafregional_ipset.ipset"]
  name = "%s"
  metric_name = "%s"
  predicates {
    data_id = "${aws_wafregional_ipset.ipset.id}"
    negated = false
    type = "IPMatch"
  }
}
resource "aws_wafregional_web_acl" "waf_acl" {
  depends_on = ["aws_wafregional_ipset.ipset", "aws_wafregional_rule.wafrule"]
  name = "%s"
  metric_name = "%s"
  default_action {
    type = "BLOCK"
  }
  rules {
    action {
       type = "BLOCK"
    }
    priority = 1 
    rule_id = "${aws_wafregional_rule.wafrule.id}"
  }
}`, name, name, name, name, name)
}
