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

func TestAccAWSWafRule_basic(t *testing.T) {
	var v waf.Rule
	wafRuleName := fmt.Sprintf("wafrule%s", acctest.RandString(5))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRuleDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafRuleConfig(wafRuleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleExists("aws_waf_rule.wafrule", &v),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "name", wafRuleName),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "predicates.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "metric_name", wafRuleName),
				),
			},
		},
	})
}

func TestAccAWSWafRule_changeNameForceNew(t *testing.T) {
	var before, after waf.Rule
	wafRuleName := fmt.Sprintf("wafrule%s", acctest.RandString(5))
	wafRuleNewName := fmt.Sprintf("wafrulenew%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafIPSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleConfig(wafRuleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleExists("aws_waf_rule.wafrule", &before),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "name", wafRuleName),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "predicates.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "metric_name", wafRuleName),
				),
			},
			{
				Config: testAccAWSWafRuleConfigChangeName(wafRuleNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleExists("aws_waf_rule.wafrule", &after),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "name", wafRuleNewName),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "predicates.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_rule.wafrule", "metric_name", wafRuleNewName),
				),
			},
		},
	})
}

func TestAccAWSWafRule_disappears(t *testing.T) {
	var v waf.Rule
	wafRuleName := fmt.Sprintf("wafrule%s", acctest.RandString(5))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRuleConfig(wafRuleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRuleExists("aws_waf_rule.wafrule", &v),
					testAccCheckAWSWafRuleDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSWafRuleDisappears(v *waf.Rule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafconn

		// ChangeToken
		var ct *waf.GetChangeTokenInput

		resp, err := conn.GetChangeToken(ct)
		if err != nil {
			return fmt.Errorf("Error getting change token: %s", err)
		}

		req := &waf.UpdateRuleInput{
			ChangeToken: resp.ChangeToken,
			RuleId:      v.RuleId,
		}

		for _, Predicate := range v.Predicates {
			Predicate := &waf.RuleUpdate{
				Action: aws.String("DELETE"),
				Predicate: &waf.Predicate{
					Negated: Predicate.Negated,
					Type:    Predicate.Type,
					DataId:  Predicate.DataId,
				},
			}
			req.Updates = append(req.Updates, Predicate)
		}

		_, err = conn.UpdateRule(req)
		if err != nil {
			return fmt.Errorf("Error Updating WAF Rule: %s", err)
		}

		resp, err = conn.GetChangeToken(ct)
		if err != nil {
			return fmt.Errorf("Error getting change token for waf Rule: %s", err)
		}

		opts := &waf.DeleteRuleInput{
			ChangeToken: resp.ChangeToken,
			RuleId:      v.RuleId,
		}
		if _, err := conn.DeleteRule(opts); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckAWSWafRuleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_waf_rule" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
		resp, err := conn.GetRule(
			&waf.GetRuleInput{
				RuleId: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if *resp.Rule.RuleId == rs.Primary.ID {
				return fmt.Errorf("WAF Rule %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the Rule is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "WAFNonexistentItemException" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccCheckAWSWafRuleExists(n string, v *waf.Rule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF Rule ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
		resp, err := conn.GetRule(&waf.GetRuleInput{
			RuleId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.Rule.RuleId == rs.Primary.ID {
			*v = *resp.Rule
			return nil
		}

		return fmt.Errorf("WAF Rule (%s) not found", rs.Primary.ID)
	}
}

func testAccAWSWafRuleConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_waf_rule" "wafrule" {
  depends_on = ["aws_waf_ipset.ipset"]
  name = "%s"
  metric_name = "%s"
  predicates {
    data_id = "${aws_waf_ipset.ipset.id}"
    negated = false
    type = "IPMatch"
  }
}`, name, name, name)
}

func testAccAWSWafRuleConfigChangeName(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_ipset" "ipset" {
  name = "%s"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_waf_rule" "wafrule" {
  depends_on = ["aws_waf_ipset.ipset"]
  name = "%s"
  metric_name = "%s"
  predicates {
    data_id = "${aws_waf_ipset.ipset.id}"
    negated = false
    type = "IPMatch"
  }
}`, name, name, name)
}
