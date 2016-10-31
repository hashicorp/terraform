package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/waf"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/acctest"
)

func TestAccAWSWafSqlInjectionMatchSet_basic(t *testing.T) {
	var v waf.SqlInjectionMatchSet
	sqlInjectionMatchSet := fmt.Sprintf("sqlInjectionMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSqlInjectionMatchSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafSqlInjectionMatchSetConfig(sqlInjectionMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSqlInjectionMatchSetExists("aws_waf_sql_injection_match_set.sql_injection_match_set", &v),
					resource.TestCheckResourceAttr(
						"aws_waf_sql_injection_match_set.sql_injection_match_set", "name", sqlInjectionMatchSet),
					resource.TestCheckResourceAttr(
						"aws_waf_sql_injection_match_set.sql_injection_match_set", "sql_injection_match_tuples.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSWafSqlInjectionMatchSet_changeNameForceNew(t *testing.T) {
	var before, after waf.SqlInjectionMatchSet
	sqlInjectionMatchSet := fmt.Sprintf("sqlInjectionMatchSet-%s", acctest.RandString(5))
	sqlInjectionMatchSetNewName := fmt.Sprintf("sqlInjectionMatchSetNewName-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSqlInjectionMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafSqlInjectionMatchSetConfig(sqlInjectionMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSqlInjectionMatchSetExists("aws_waf_sql_injection_match_set.sql_injection_match_set", &before),
					resource.TestCheckResourceAttr(
						"aws_waf_sql_injection_match_set.sql_injection_match_set", "name", sqlInjectionMatchSet),
					resource.TestCheckResourceAttr(
						"aws_waf_sql_injection_match_set.sql_injection_match_set", "sql_injection_match_tuples.#", "1"),
				),
			},
			{
				Config: testAccAWSWafSqlInjectionMatchSetConfigChangeName(sqlInjectionMatchSetNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSqlInjectionMatchSetExists("aws_waf_sql_injection_match_set.sql_injection_match_set", &after),
					resource.TestCheckResourceAttr(
						"aws_waf_sql_injection_match_set.sql_injection_match_set", "name", sqlInjectionMatchSetNewName),
					resource.TestCheckResourceAttr(
						"aws_waf_sql_injection_match_set.sql_injection_match_set", "sql_injection_match_tuples.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSWafSqlInjectionMatchSet_disappears(t *testing.T) {
	var v waf.SqlInjectionMatchSet
	sqlInjectionMatchSet := fmt.Sprintf("sqlInjectionMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSqlInjectionMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafSqlInjectionMatchSetConfig(sqlInjectionMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSqlInjectionMatchSetExists("aws_waf_sql_injection_match_set.sql_injection_match_set", &v),
					testAccCheckAWSWafSqlInjectionMatchSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSWafSqlInjectionMatchSetDisappears(v *waf.SqlInjectionMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafconn

		var ct *waf.GetChangeTokenInput

		resp, err := conn.GetChangeToken(ct)
		if err != nil {
			return fmt.Errorf("Error getting change token: %s", err)
		}

		req := &waf.UpdateSqlInjectionMatchSetInput{
			ChangeToken:            resp.ChangeToken,
			SqlInjectionMatchSetId: v.SqlInjectionMatchSetId,
		}

		for _, sqlInjectionMatchTuple := range v.SqlInjectionMatchTuples {
			sqlInjectionMatchTupleUpdate := &waf.SqlInjectionMatchSetUpdate{
				Action: aws.String("DELETE"),
				SqlInjectionMatchTuple: &waf.SqlInjectionMatchTuple{
					FieldToMatch:       sqlInjectionMatchTuple.FieldToMatch,
					TextTransformation: sqlInjectionMatchTuple.TextTransformation,
				},
			}
			req.Updates = append(req.Updates, sqlInjectionMatchTupleUpdate)
		}
		_, err = conn.UpdateSqlInjectionMatchSet(req)
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error updating SqlInjectionMatchSet: {{err}}", err)
		}

		resp, err = conn.GetChangeToken(ct)
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error getting change token: {{err}}", err)
		}

		opts := &waf.DeleteSqlInjectionMatchSetInput{
			ChangeToken:            resp.ChangeToken,
			SqlInjectionMatchSetId: v.SqlInjectionMatchSetId,
		}
		if _, err := conn.DeleteSqlInjectionMatchSet(opts); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckAWSWafSqlInjectionMatchSetExists(n string, v *waf.SqlInjectionMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF SqlInjectionMatchSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
		resp, err := conn.GetSqlInjectionMatchSet(&waf.GetSqlInjectionMatchSetInput{
			SqlInjectionMatchSetId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.SqlInjectionMatchSet.SqlInjectionMatchSetId == rs.Primary.ID {
			*v = *resp.SqlInjectionMatchSet
			return nil
		}

		return fmt.Errorf("WAF SqlInjectionMatchSet (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckAWSWafSqlInjectionMatchSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_waf_byte_match_set" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
		resp, err := conn.GetSqlInjectionMatchSet(
			&waf.GetSqlInjectionMatchSetInput{
				SqlInjectionMatchSetId: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if *resp.SqlInjectionMatchSet.SqlInjectionMatchSetId == rs.Primary.ID {
				return fmt.Errorf("WAF SqlInjectionMatchSet %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the SqlInjectionMatchSet is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "WAFNonexistentItemException" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccAWSWafSqlInjectionMatchSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_sql_injection_match_set" "sql_injection_match_set" {
  name = "%s"
  sql_injection_match_tuples {
    text_transformation = "URL_DECODE"
    field_to_match {
      type = "QUERY_STRING"
    }
  }
}`, name)
}

func testAccAWSWafSqlInjectionMatchSetConfigChangeName(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_sql_injection_match_set" "sql_injection_match_set" {
  name = "%s"
  sql_injection_match_tuples {
    text_transformation = "URL_DECODE"
    field_to_match {
      type = "QUERY_STRING"
    }
  }
}`, name)
}
