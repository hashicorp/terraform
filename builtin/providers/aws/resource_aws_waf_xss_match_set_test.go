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

func TestAccAWSWafXssMatchSet_basic(t *testing.T) {
	var v waf.XssMatchSet
	xssMatchSet := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafXssMatchSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafXssMatchSetConfig(xssMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafXssMatchSetExists("aws_waf_xss_match_set.xss_match_set", &v),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "name", xssMatchSet),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.field_to_match.2316364334.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.field_to_match.2316364334.type", "QUERY_STRING"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.text_transformation", "NONE"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.field_to_match.3756326843.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.field_to_match.3756326843.type", "URI"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.text_transformation", "NONE"),
				),
			},
		},
	})
}

func TestAccAWSWafXssMatchSet_changeNameForceNew(t *testing.T) {
	var before, after waf.XssMatchSet
	xssMatchSet := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))
	xssMatchSetNewName := fmt.Sprintf("xssMatchSetNewName-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafXssMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafXssMatchSetConfig(xssMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafXssMatchSetExists("aws_waf_xss_match_set.xss_match_set", &before),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "name", xssMatchSet),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
				),
			},
			{
				Config: testAccAWSWafXssMatchSetConfigChangeName(xssMatchSetNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafXssMatchSetExists("aws_waf_xss_match_set.xss_match_set", &after),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "name", xssMatchSetNewName),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSWafXssMatchSet_disappears(t *testing.T) {
	var v waf.XssMatchSet
	xssMatchSet := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafXssMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafXssMatchSetConfig(xssMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafXssMatchSetExists("aws_waf_xss_match_set.xss_match_set", &v),
					testAccCheckAWSWafXssMatchSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSWafXssMatchSet_changeTuples(t *testing.T) {
	var before, after waf.XssMatchSet
	setName := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafXssMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafXssMatchSetConfig(setName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafXssMatchSetExists("aws_waf_xss_match_set.xss_match_set", &before),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "name", setName),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.field_to_match.2316364334.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.field_to_match.2316364334.type", "QUERY_STRING"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2018581549.text_transformation", "NONE"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.field_to_match.3756326843.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.field_to_match.3756326843.type", "URI"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2786024938.text_transformation", "NONE"),
				),
			},
			{
				Config: testAccAWSWafXssMatchSetConfig_changeTuples(setName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafXssMatchSetExists("aws_waf_xss_match_set.xss_match_set", &after),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "name", setName),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2893682529.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2893682529.field_to_match.4253810390.data", "GET"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2893682529.field_to_match.4253810390.type", "METHOD"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.2893682529.text_transformation", "HTML_ENTITY_DECODE"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.4270311415.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.4270311415.field_to_match.281401076.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.4270311415.field_to_match.281401076.type", "BODY"),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.4270311415.text_transformation", "CMD_LINE"),
				),
			},
		},
	})
}

func TestAccAWSWafXssMatchSet_noTuples(t *testing.T) {
	var ipset waf.XssMatchSet
	setName := fmt.Sprintf("xssMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafXssMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafXssMatchSetConfig_noTuples(setName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafXssMatchSetExists("aws_waf_xss_match_set.xss_match_set", &ipset),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "name", setName),
					resource.TestCheckResourceAttr(
						"aws_waf_xss_match_set.xss_match_set", "xss_match_tuples.#", "0"),
				),
			},
		},
	})
}

func testAccCheckAWSWafXssMatchSetDisappears(v *waf.XssMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafconn

		wr := newWafRetryer(conn, "global")
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

func testAccCheckAWSWafXssMatchSetExists(n string, v *waf.XssMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF XssMatchSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
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

func testAccCheckAWSWafXssMatchSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_waf_byte_match_set" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
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

func testAccAWSWafXssMatchSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_xss_match_set" "xss_match_set" {
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

func testAccAWSWafXssMatchSetConfigChangeName(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_xss_match_set" "xss_match_set" {
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

func testAccAWSWafXssMatchSetConfig_changeTuples(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_xss_match_set" "xss_match_set" {
  name = "%s"
  xss_match_tuples {
    text_transformation = "CMD_LINE"
    field_to_match {
      type = "BODY"
    }
  }

  xss_match_tuples {
    text_transformation = "HTML_ENTITY_DECODE"
    field_to_match {
      type = "METHOD"
      data = "GET"
    }
  }
}`, name)
}

func testAccAWSWafXssMatchSetConfig_noTuples(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_xss_match_set" "xss_match_set" {
  name = "%s"
}`, name)
}
