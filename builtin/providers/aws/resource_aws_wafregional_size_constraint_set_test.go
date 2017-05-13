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

func TestAccAWSWafRegionalSizeConstraintSet_basic(t *testing.T) {
	var v waf.SizeConstraintSet
	sizeConstraintSet := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafRegionalSizeConstraintSetConfig(sizeConstraintSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalSizeConstraintSetExists("aws_wafregional_size_constraint_set.size_constraint_set", &v),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "name", sizeConstraintSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalSizeConstraintSet_changeNameForceNew(t *testing.T) {
	var before, after waf.SizeConstraintSet
	sizeConstraintSet := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))
	sizeConstraintSetNewName := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalSizeConstraintSetConfig(sizeConstraintSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalSizeConstraintSetExists("aws_wafregional_size_constraint_set.size_constraint_set", &before),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "name", sizeConstraintSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.#", "1"),
				),
			},
			{
				Config: testAccAWSWafRegionalSizeConstraintSetConfigChangeName(sizeConstraintSetNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalSizeConstraintSetExists("aws_wafregional_size_constraint_set.size_constraint_set", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "name", sizeConstraintSetNewName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalSizeConstraintSet_changeSizeConstraint(t *testing.T) {
	var sizeConstraintSet waf.SizeConstraintSet
	sizeConstraintSetName := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalSizeConstraintSetConfig(sizeConstraintSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalSizeConstraintSetExists("aws_wafregional_size_constraint_set.size_constraint_set", &sizeConstraintSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "name", sizeConstraintSetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.2029852522.comparison_operator", "EQ"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.2029852522.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.2029852522.field_to_match.281401076.data", ""),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.2029852522.field_to_match.281401076.type", "BODY"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.2029852522.size", "4096"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.2029852522.text_transformation", "NONE"),
				),
			},
			{
				Config: testAccAWSWafRegionalSizeConstraintSetConfig_changeSizeConstraint(sizeConstraintSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalSizeConstraintSetExists("aws_wafregional_size_constraint_set.size_constraint_set", &sizeConstraintSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "name", sizeConstraintSetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.539665067.comparison_operator", "GE"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.539665067.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.539665067.field_to_match.4253810390.data", "GET"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.539665067.field_to_match.4253810390.type", "METHOD"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.539665067.size", "2048"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.539665067.text_transformation", "LOWERCASE"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalSizeConstraintSet_noSizeConstraint(t *testing.T) {
	var sizeConstraintSet waf.SizeConstraintSet
	sizeConstraintSetName := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalSizeConstraintSetConfig_noSizeConstraint(sizeConstraintSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalSizeConstraintSetExists("aws_wafregional_size_constraint_set.size_constraint_set", &sizeConstraintSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "name", sizeConstraintSetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_size_constraint_set.size_constraint_set", "size_constraint.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalSizeConstraintSet_disappears(t *testing.T) {
	var v waf.SizeConstraintSet
	sizeConstraintSet := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalSizeConstraintSetConfig(sizeConstraintSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalSizeConstraintSetExists("aws_wafregional_size_constraint_set.size_constraint_set", &v),
					testAccCheckAWSWafRegionalSizeConstraintSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSWafRegionalSizeConstraintSetDisappears(v *waf.SizeConstraintSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		region := testAccProvider.Meta().(*AWSClient).region

		wr := newWafRegionalRetryer(conn, region)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateSizeConstraintSetInput{
				ChangeToken:         token,
				SizeConstraintSetId: v.SizeConstraintSetId,
			}

			for _, sizeConstraint := range v.SizeConstraints {
				sizeConstraintUpdate := &waf.SizeConstraintSetUpdate{
					Action: aws.String("DELETE"),
					SizeConstraint: &waf.SizeConstraint{
						FieldToMatch:       sizeConstraint.FieldToMatch,
						ComparisonOperator: sizeConstraint.ComparisonOperator,
						Size:               sizeConstraint.Size,
						TextTransformation: sizeConstraint.TextTransformation,
					},
				}
				req.Updates = append(req.Updates, sizeConstraintUpdate)
			}
			return conn.UpdateSizeConstraintSet(req)
		})
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error updating SizeConstraintSet: {{err}}", err)
		}

		_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
			opts := &waf.DeleteSizeConstraintSetInput{
				ChangeToken:         token,
				SizeConstraintSetId: v.SizeConstraintSetId,
			}
			return conn.DeleteSizeConstraintSet(opts)
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckAWSWafRegionalSizeConstraintSetExists(n string, v *waf.SizeConstraintSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF SizeConstraintSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetSizeConstraintSet(&waf.GetSizeConstraintSetInput{
			SizeConstraintSetId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.SizeConstraintSet.SizeConstraintSetId == rs.Primary.ID {
			*v = *resp.SizeConstraintSet
			return nil
		}

		return fmt.Errorf("WAF SizeConstraintSet (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckAWSWafRegionalSizeConstraintSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_wafregional_byte_match_set" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetSizeConstraintSet(
			&waf.GetSizeConstraintSetInput{
				SizeConstraintSetId: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if *resp.SizeConstraintSet.SizeConstraintSetId == rs.Primary.ID {
				return fmt.Errorf("WAF SizeConstraintSet %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the SizeConstraintSet is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "WAFNonexistentItemException" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccAWSWafRegionalSizeConstraintSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_size_constraint_set" "size_constraint_set" {
  name = "%s"
  size_constraint {
    text_transformation = "NONE"
    comparison_operator = "EQ"
    size = "4096"
    field_to_match {
      type = "BODY"
    }
  }
}`, name)
}

func testAccAWSWafRegionalSizeConstraintSetConfigChangeName(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_size_constraint_set" "size_constraint_set" {
  name = "%s"
  size_constraint {
    text_transformation = "NONE"
    comparison_operator = "EQ"
    size = "4096"
    field_to_match {
      type = "BODY"
    }
  }
}`, name)
}

func testAccAWSWafRegionalSizeConstraintSetConfig_changeSizeConstraint(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_size_constraint_set" "size_constraint_set" {
  name = "%s"
  size_constraint {
    text_transformation = "LOWERCASE"
    comparison_operator = "GE"
    size = "2048"
    field_to_match {
      type = "METHOD"
      data = "GET"
    }
  }
}`, name)
}

func testAccAWSWafRegionalSizeConstraintSetConfig_noSizeConstraint(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_size_constraint_set" "size_constraint_set" {
  name = "%s"
}`, name)
}
