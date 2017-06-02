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

func TestAccAWSWafRegionalByteMatchSet_basic(t *testing.T) {
	var v waf.ByteMatchSet
	byteMatchSet := fmt.Sprintf("byteMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalByteMatchSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSWafRegionalByteMatchSetConfig(byteMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalByteMatchSetExists("aws_wafregional_byte_match_set.byte_set", &v),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "name", byteMatchSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.target_string", "badrefer1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.text_transformation", "NONE"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.target_string", "badrefer2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.text_transformation", "NONE"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalByteMatchSet_changeNameForceNew(t *testing.T) {
	var before, after waf.ByteMatchSet
	byteMatchSet := fmt.Sprintf("byteMatchSet-%s", acctest.RandString(5))
	byteMatchSetNewName := fmt.Sprintf("byteMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalByteMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalByteMatchSetConfig(byteMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalByteMatchSetExists("aws_wafregional_byte_match_set.byte_set", &before),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "name", byteMatchSet),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.target_string", "badrefer1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.text_transformation", "NONE"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.target_string", "badrefer2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.text_transformation", "NONE"),
				),
			},
			{
				Config: testAccAWSWafRegionalByteMatchSetConfigChangeName(byteMatchSetNewName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalByteMatchSetExists("aws_wafregional_byte_match_set.byte_set", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "name", byteMatchSetNewName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.target_string", "badrefer1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.text_transformation", "NONE"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.target_string", "badrefer2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.text_transformation", "NONE"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalByteMatchSet_changeByteMatchTuple(t *testing.T) {
	var before, after waf.ByteMatchSet
	byteMatchSetName := fmt.Sprintf("byte-batch-set-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalByteMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalByteMatchSetConfig(byteMatchSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalByteMatchSetExists("aws_wafregional_byte_match_set.byte_set", &before),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "name", byteMatchSetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.target_string", "badrefer1"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2174619346.text_transformation", "NONE"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.data", "referer"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.field_to_match.2991901334.type", "HEADER"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.positional_constraint", "CONTAINS"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.target_string", "badrefer2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.839525137.text_transformation", "NONE"),
				),
			},
			{
				Config: testAccAWSWafRegionalByteMatchSetConfigChangeByteMatchTuple(byteMatchSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalByteMatchSetExists("aws_wafregional_byte_match_set.byte_set", &after),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "name", byteMatchSetName),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2397850647.field_to_match.4253810390.data", "GET"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2397850647.field_to_match.4253810390.type", "METHOD"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2397850647.positional_constraint", "STARTS_WITH"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2397850647.target_string", "badrefer1+"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.2397850647.text_transformation", "LOWERCASE"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.4153613423.field_to_match.3756326843.data", ""),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.4153613423.field_to_match.3756326843.type", "URI"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.4153613423.positional_constraint", "ENDS_WITH"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.4153613423.target_string", "badrefer2+"),
					resource.TestCheckResourceAttr(
						"aws_wafregional_byte_match_set.byte_set", "byte_match_tuple.4153613423.text_transformation", "LOWERCASE"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalByteMatchSet_noByteMatchTuples(t *testing.T) {
	var byteMatchSet waf.ByteMatchSet
	byteMatchSetName := fmt.Sprintf("byte-batch-set-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalByteMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalByteMatchSetConfig_noDescriptors(byteMatchSetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckAWSWafRegionalByteMatchSetExists("aws_wafregional_byte_match_set.byte_match_set", &byteMatchSet),
					resource.TestCheckResourceAttr("aws_wafregional_byte_match_set.byte_match_set", "name", byteMatchSetName),
					resource.TestCheckResourceAttr("aws_wafregional_byte_match_set.byte_match_set", "byte_match_tuple.#", "0"),
				),
			},
		},
	})
}

func TestAccAWSWafRegionalByteMatchSet_disappears(t *testing.T) {
	var v waf.ByteMatchSet
	byteMatchSet := fmt.Sprintf("byteMatchSet-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSWafRegionalByteMatchSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSWafRegionalByteMatchSetConfig(byteMatchSet),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSWafRegionalByteMatchSetExists("aws_wafregional_byte_match_set.byte_set", &v),
					testAccCheckAWSWafRegionalByteMatchSetDisappears(&v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckAWSWafRegionalByteMatchSetDisappears(v *waf.ByteMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		region := testAccProvider.Meta().(*AWSClient).region

		wr := newWafRegionalRetryer(conn, region)
		_, err := wr.RetryWithToken(func(token *string) (interface{}, error) {
			req := &waf.UpdateByteMatchSetInput{
				ChangeToken:    token,
				ByteMatchSetId: v.ByteMatchSetId,
			}

			for _, ByteMatchTuple := range v.ByteMatchTuples {
				ByteMatchUpdate := &waf.ByteMatchSetUpdate{
					Action: aws.String("DELETE"),
					ByteMatchTuple: &waf.ByteMatchTuple{
						FieldToMatch:         ByteMatchTuple.FieldToMatch,
						PositionalConstraint: ByteMatchTuple.PositionalConstraint,
						TargetString:         ByteMatchTuple.TargetString,
						TextTransformation:   ByteMatchTuple.TextTransformation,
					},
				}
				req.Updates = append(req.Updates, ByteMatchUpdate)
			}

			return conn.UpdateByteMatchSet(req)
		})
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error updating ByteMatchSet: {{err}}", err)
		}

		_, err = wr.RetryWithToken(func(token *string) (interface{}, error) {
			opts := &waf.DeleteByteMatchSetInput{
				ChangeToken:    token,
				ByteMatchSetId: v.ByteMatchSetId,
			}
			return conn.DeleteByteMatchSet(opts)
		})
		if err != nil {
			return errwrap.Wrapf("[ERROR] Error deleting ByteMatchSet: {{err}}", err)
		}

		return nil
	}
}

func testAccCheckAWSWafRegionalByteMatchSetExists(n string, v *waf.ByteMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF ByteMatchSet ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetByteMatchSet(&waf.GetByteMatchSetInput{
			ByteMatchSetId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.ByteMatchSet.ByteMatchSetId == rs.Primary.ID {
			*v = *resp.ByteMatchSet
			return nil
		}

		return fmt.Errorf("WAF ByteMatchSet (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckAWSWafRegionalByteMatchSetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_wafregional_byte_match_set" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).wafregionalconn
		resp, err := conn.GetByteMatchSet(
			&waf.GetByteMatchSetInput{
				ByteMatchSetId: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if *resp.ByteMatchSet.ByteMatchSetId == rs.Primary.ID {
				return fmt.Errorf("WAF ByteMatchSet %s still exists", rs.Primary.ID)
			}
		}

		// Return nil if the ByteMatchSet is already destroyed
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "WAFNonexistentItemException" {
				return nil
			}
		}

		return err
	}

	return nil
}

func testAccAWSWafRegionalByteMatchSetConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_byte_match_set" "byte_set" {
  name = "%s"
  byte_match_tuple {
    text_transformation = "NONE"
    target_string = "badrefer1"
    positional_constraint = "CONTAINS"
    field_to_match {
      type = "HEADER"
      data = "referer"
    }
  }

  byte_match_tuple {
    text_transformation = "NONE"
    target_string = "badrefer2"
    positional_constraint = "CONTAINS"
    field_to_match {
      type = "HEADER"
      data = "referer"
    }
  }
}`, name)
}

func testAccAWSWafRegionalByteMatchSetConfigChangeName(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_byte_match_set" "byte_set" {
  name = "%s"
  byte_match_tuple {
    text_transformation = "NONE"
    target_string = "badrefer1"
    positional_constraint = "CONTAINS"
    field_to_match {
      type = "HEADER"
      data = "referer"
    }
  }

  byte_match_tuple {
    text_transformation = "NONE"
    target_string = "badrefer2"
    positional_constraint = "CONTAINS"
    field_to_match {
      type = "HEADER"
      data = "referer"
    }
  }
}`, name)
}

func testAccAWSWafRegionalByteMatchSetConfig_noDescriptors(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_byte_match_set" "byte_match_set" {
	name = "%s"
}`, name)
}

func testAccAWSWafRegionalByteMatchSetConfigChangeByteMatchTuple(name string) string {
	return fmt.Sprintf(`
resource "aws_wafregional_byte_match_set" "byte_set" {
  name = "%s"
  byte_match_tuple {
    text_transformation = "LOWERCASE"
    target_string = "badrefer1+"
    positional_constraint = "STARTS_WITH"
    field_to_match {
      type = "METHOD"
      data = "GET"
    }
  }

  byte_match_tuple {
    text_transformation = "LOWERCASE"
    target_string = "badrefer2+"
    positional_constraint = "ENDS_WITH"
    field_to_match {
      type = "URI"
    }
  }
}`, name)
}
