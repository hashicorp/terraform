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

func TestAccAWSWafSizeConstraintSet_basic(t *testing.T) {
	var v waf.SizeConstraintSet
	sizeConstraintSet := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafSizeConstraintSetConfig(sizeConstraintSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSizeConstraintSetExists("aws_waf_size_constraint_set.size_constraint_set", &v),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "name", sizeConstraintSet),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.comparison_operator", "EQ"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.field_to_match.281401076.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.field_to_match.281401076.type", "BODY"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.size", "4096"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.text_transformation", "NONE"),
				),
			},
		},
	})
}

func TestAccAWSWafSizeConstraintSet_changeNameForceNew(t *testing.T) {
	var before, after waf.SizeConstraintSet
	sizeConstraintSet := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))
	sizeConstraintSetNewName := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafSizeConstraintSetConfig(sizeConstraintSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSizeConstraintSetExists("aws_waf_size_constraint_set.size_constraint_set", &before),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "name", sizeConstraintSet),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.#", "1"),
				),
			},
			{
				Config: testAccAWSWafSizeConstraintSetConfigChangeName(sizeConstraintSetNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSizeConstraintSetExists("aws_waf_size_constraint_set.size_constraint_set", &after),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "name", sizeConstraintSetNewName),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSWafSizeConstraintSet_disappears(t *testing.T) {
	var v waf.SizeConstraintSet
	sizeConstraintSet := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafSizeConstraintSetConfig(sizeConstraintSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafSizeConstraintSetExists("aws_waf_size_constraint_set.size_constraint_set", &v),
					testAccCheckAWSWafSizeConstraintSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSWafSizeConstraintSet_changeConstraints(t *testing.T) {
	var before, after waf.SizeConstraintSet
	setName := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafSizeConstraintSetConfig(setName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafSizeConstraintSetExists("aws_waf_size_constraint_set.size_constraint_set", &before),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "name", setName),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.comparison_operator", "EQ"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.field_to_match.281401076.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.field_to_match.281401076.type", "BODY"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.size", "4096"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.2029852522.text_transformation", "NONE"),
				),
			},
			{
				Config: testAccAWSWafSizeConstraintSetConfig_changeConstraints(setName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafSizeConstraintSetExists("aws_waf_size_constraint_set.size_constraint_set", &after),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "name", setName),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.3222308386.comparison_operator", "GE"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.3222308386.field_to_match.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.3222308386.field_to_match.281401076.data", ""),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.3222308386.field_to_match.281401076.type", "BODY"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.3222308386.size", "1024"),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.3222308386.text_transformation", "NONE"),
				),
			},
		},
	})
}

func TestAccAWSWafSizeConstraintSet_noConstraints(t *testing.T) {
	var ipset waf.SizeConstraintSet
	setName := fmt.Sprintf("sizeConstraintSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafSizeConstraintSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafSizeConstraintSetConfig_noConstraints(setName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafSizeConstraintSetExists("aws_waf_size_constraint_set.size_constraint_set", &ipset),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "name", setName),
					resource.TestCheckResourceAttr(
						"aws_waf_size_constraint_set.size_constraint_set", "size_constraints.#", "0"),
				),
			},
		},
	})
}

func testAccCheckAWSWafSizeConstraintSetDisappears(v *waf.SizeConstraintSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafconn

		wr := newWafRetryer(conn, "global")
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

func testAccCheckAWSWafSizeConstraintSetExists(n string, v *waf.SizeConstraintSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF SizeConstraintSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
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

func testAccCheckAWSWafSizeConstraintSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_waf_byte_match_set" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafconn
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

func testAccAWSWafSizeConstraintSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_size_constraint_set" "size_constraint_set" {
  name = "%s"
  size_constraints {
    text_transformation = "NONE"
    comparison_operator = "EQ"
    size = "4096"
    field_to_match {
      type = "BODY"
    }
  }
}`, name)
}

func testAccAWSWafSizeConstraintSetConfigChangeName(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_size_constraint_set" "size_constraint_set" {
  name = "%s"
  size_constraints {
    text_transformation = "NONE"
    comparison_operator = "EQ"
    size = "4096"
    field_to_match {
      type = "BODY"
    }
  }
}`, name)
}

func testAccAWSWafSizeConstraintSetConfig_changeConstraints(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_size_constraint_set" "size_constraint_set" {
  name = "%s"
  size_constraints {
    text_transformation = "NONE"
    comparison_operator = "GE"
    size = "1024"
    field_to_match {
      type = "BODY"
    }
  }
}`, name)
}

func testAccAWSWafSizeConstraintSetConfig_noConstraints(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_size_constraint_set" "size_constraint_set" {
  name = "%s"
}`, name)
}
