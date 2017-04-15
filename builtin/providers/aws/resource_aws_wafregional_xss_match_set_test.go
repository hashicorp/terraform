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

func TestAccAWSWafRegionalXssMatchSet_basic(t *testing.T) {
	var v waf.XssMatchSet
	xssMatchSet := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalXssMatchSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafRegionalXssMatchSetConfig(xssMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalXssMatchSetExists("aws_wafregional_xss_match_set.xss_match_set", &v),
					resource.TestCheckResourceAttr(
						"aws_wafregional_xss_match_set.xss_match_set", "name", xssMatchSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalXssMatchSet_changeNameForceNew(t *testing.T) {
	var before, after waf.XssMatchSet
	xssMatchSet := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))
	xssMatchSetNewName := fmt.Sprintf("xssMatchSetNewName-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalXssMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalXssMatchSetConfig(xssMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalXssMatchSetExists("aws_wafregional_xss_match_set.xss_match_set", &before),
					resource.TestCheckResourceAttr(
						"aws_wafregional_xss_match_set.xss_match_set", "name", xssMatchSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
				),
			},
			{
				Config: testAccAWSWafRegionalXssMatchSetConfigChangeName(xssMatchSetNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalXssMatchSetExists("aws_wafregional_xss_match_set.xss_match_set", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_xss_match_set.xss_match_set", "name", xssMatchSetNewName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalXssMatchSet_disappears(t *testing.T) {
	var v waf.XssMatchSet
	xssMatchSet := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalXssMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalXssMatchSetConfig(xssMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalXssMatchSetExists("aws_wafregional_xss_match_set.xss_match_set", &v),
					testAccCheckAWSWafRegionalXssMatchSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSWafRegionalXssMatchSetDisappears(v *waf.XssMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn

		wr := newWafRegionalRetryer(conn)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateXssMatchSetInput{
				ChangeToken:   token,
				XssMatchSetId: v.XssMatchSetId,
			}

			for _, xssMatchTuple := range v.XssMatchTuples {
				xssMatchTupleUpdate := &waf.XssMatchSetUpdate{
					Action: aws.String("DELETE"),
					XssMatchTuple: &waf.XssMatchTuple{
						FieldToMatch:       xssMatchTuple.FieldToMatch,
						TextTransformation: xssMatchTuple.TextTransformation,
					},
				}
				req.Updates = append(req.Updates, xssMatchTupleUpdate)
			}
			return conn.UpdateXssMatchSet(req)
		})
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error updating XssMatchSet: {{err}}", err)
		}

		_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
			opts := &waf.DeleteXssMatchSetInput{
				ChangeToken:   token,
				XssMatchSetId: v.XssMatchSetId,
			}
			return conn.DeleteXssMatchSet(opts)
		})
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error deleting XssMatchSet: {{err}}", err)
		}
		return nil
	}
}

func testAccCheckAWSWafRegionalXssMatchSetExists(n string, v *waf.XssMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF XssMatchSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetXssMatchSet(&waf.GetXssMatchSetInput{
			XssMatchSetId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.XssMatchSet.XssMatchSetId == rs.Primary.ID {
			*v = *resp.XssMatchSet
			return nil
		}

		return fmt.Errorf("WAF XssMatchSet (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckAWSWafRegionalXssMatchSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_wafregional_byte_match_set" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetXssMatchSet(
			&waf.GetXssMatchSetInput{
				XssMatchSetId: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if *resp.XssMatchSet.XssMatchSetId == rs.Primary.ID {
				return fmt.Errorf("WAF XssMatchSet %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the XssMatchSet is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "WAFNonexistentItemException" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccAWSWafRegionalXssMatchSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_xss_match_set" "xss_match_set" {
  name = "%s"
  xss_match_tuples {
    text_transformation = "NONE"
    field_to_match {
      type = "URI"
    }
  }

  xss_match_tuples {
    text_transformation = "NONE"
    field_to_match {
      type = "QUERY_STRING"
    }
  }
}`, name)
}

func testAccAWSWafRegionalXssMatchSetConfigChangeName(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_xss_match_set" "xss_match_set" {
  name = "%s"
  xss_match_tuples {
    text_transformation = "NONE"
    field_to_match {
      type = "URI"
    }
  }

  xss_match_tuples {
    text_transformation = "NONE"
    field_to_match {
      type = "QUERY_STRING"
    }
  }
}`, name)
}
